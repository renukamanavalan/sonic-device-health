package lomtelemetry

import (
    cmn "lom/src/lib/lomcommon"

    "fmt"
    "sync"
    "time"

    zmq "github.com/pebbe/zmq4"
)

/*
 * NOTE;
 * ZMQ sockets are not thread safe. Hence restrict use of a socket within
 * the same goroutine that created it, until its close.
 */

/*
 * Each ChannelType_t uses a dedicated channel
 * Compute the port by adding chType to start port
 */
const ZMQ_REQ_REP_START_PORT = 5650
const ZMQ_XPUB_START_PORT = 5750 /* Subscribers connect to xpub */
const ZMQ_XSUB_START_PORT = 5850 /* Publishers connect to xsub */
const ZMQ_PROXY_CTRL_PORT = 5950

const ZMQ_ADDRESS = "tcp://127.0.0.1:%d"

var SUB_CHANNEL_TIMEOUT = time.Duration(10) * time.Second
var ZMQ_ASYNC_CONNECT_PAUSE = time.Duration(300) * time.Millisecond

/* Logical grouping of ChannelType_t values for validation use */
type chTypes_t map[ChannelType_t]bool

var pubsub_types = chTypes_t{
    CHANNEL_TYPE_EVENTS:    true,
    CHANNEL_TYPE_COUNTERS:  true,
    CHANNEL_TYPE_REDBUTTON: true,
}

var reqrep_types = chTypes_t{
    CHANNEL_TYPE_ECHO: true,
    CHANNEL_TYPE_SCS:  false,
}

var NA_types = chTypes_t{CHANNEL_TYPE_NA: true}

type chModeData_t struct {
    types     chTypes_t
    startPort int
    sType     zmq.Type
    isConnect bool /* connect / bind socket */
}

/* Mapping mode to acceptable types for validation */
var chModeInfo = map[channelMode_t]chModeData_t{
    CHANNEL_MODE_PUBLISHER:      chModeData_t{pubsub_types, ZMQ_XSUB_START_PORT, zmq.PUB, true},
    CHANNEL_MODE_SUBSCRIBER:     chModeData_t{pubsub_types, ZMQ_XPUB_START_PORT, zmq.SUB, true},
    CHANNEL_MODE_REQUEST:        chModeData_t{reqrep_types, ZMQ_REQ_REP_START_PORT, zmq.REQ, true},
    CHANNEL_MODE_RESPONSE:       chModeData_t{reqrep_types, ZMQ_REQ_REP_START_PORT, zmq.REP, false},
    CHANNEL_MODE_PROXY_CTRL_PUB: chModeData_t{NA_types, ZMQ_PROXY_CTRL_PORT, zmq.PUB, true},
    CHANNEL_MODE_PROXY_CTRL_SUB: chModeData_t{NA_types, ZMQ_PROXY_CTRL_PORT, zmq.SUB, false},
}

/* Is sytem shutdown initiated yet? */
var shutdownYet = false

type sockInfo_t struct {
    address   string
    sType     zmq.Type
    isConnect bool
}

func getAddress(mode channelMode_t, chType ChannelType_t) (sockInfo *sockInfo_t, err error) {

    /* Cross validation between mode & ChannelType_t */
    if info, ok := chModeInfo[mode]; !ok {
        err = cmn.LogError("Unknown channel mode (%v)", mode)
    } else if _, ok := info.types[chType]; !ok {
        err = cmn.LogError("Unknown channel type (%d) for mode(%d)", chType, mode)
    } else {
        sockInfo = &sockInfo_t{
            fmt.Sprintf(ZMQ_ADDRESS, info.startPort+int(chType)),
            info.sType,
            info.isConnect}
    }
    return
}

/*
 * Single context shared by all threads & routines.
 * Ctx is threadsafe, but not sockets
 * Hence one context per process.
 */
var zctx *zmq.Context

/*
 * Track all open sockets.
 * Terminate context blocks until this goes 0
 */
var socketsList = sync.Map{}

/*
 * Collect all open sockets. Ctx termination is blocked by any
 * open socket.
 * string is some friendly identification of caller to help track
 * who is not closing, upon leak.
 */

/*
 * Each socket close writes into this channel
 * During shutdown term contex sleep on this channel until all
 * all sockets are closed, hence the sockets list is empty.
 */
var chSocksClose = make(chan int)

func getContext() (*zmq.Context, error) {
    if shutdownYet {
        return nil, cmn.LogError("System is shutting down. No context")
    }
    var err error
    if zctx == nil {
        if zctx, err = zmq.NewContext(); err != nil {
            return nil, cmn.LogError("Failed to get zmq context (%v)", err)
        }
        /* Terminate on system shutdown */
        go terminateContext()
    }
    return zctx, nil
}

func terminateContext() {
    chShutdown := cmn.RegisterForSysShutdown("terminate ZMQ context")

    /* Sleep till shutdown */
    for !shutdownYet {
        select {
        case <-chShutdown:
            shutdownYet = true
        case <-chSocksClose:
            /*
             * Some socket closed. Nothing to do
             * Yet must read to drain, else writer blocks.
             */
        }
    }

    var pending []string
    /* System shutdown initiated; Wait for open sockets to close */
    for {
        pending = []string{}
        socketsList.Range(func(k, v any) bool {
            pending = append(pending, v.(string))
            return true
        })
        if len(pending) == 0 {
            break
        }
        cmn.LogError("Waiting for [%d] socks to close pending(%v)", len(pending), pending)

        /* Sleep until someone closes or timeout */
        select {
        case <-time.After(time.Second):
            cmn.LogError("Timeout upon waiting; exiting w/o context termination")
            return
        case _, ok := <-chSocksClose:
            if !ok {
                cmn.LogError("Internal error: chSocksClose is not expected to be closed.")
                return
            }
            /* go back & check the list */
        }
    }
    cmn.LogInfo("terminating context. pending(%d)(%v)", len(pending), pending)
    zctx.Term()
    zctx = nil
    cmn.LogInfo("terminated context.")
}

/*
 * create socket; connect/bind; Add to active socket list used by terminate context.
 */
func getSocket(mode channelMode_t, chType ChannelType_t, requester string) (sock *zmq.Socket, err error) {
    if shutdownYet {
        return nil, cmn.LogError("System is shutting down. No new socket")
    }
    defer func() {
        if err != nil {
            sock.Close()
            sock = nil
        }
    }()

    var info *sockInfo_t
    if info, err = getAddress(mode, chType); err != nil {
        return
    }

    if sock, err = zctx.NewSocket(info.sType); err != nil {
        err = cmn.LogError("Failed to get socket mode(%v) err(%v)", mode, err)
        return
    }
    /*
     * All pub & sub connect to xsub/xpub end points.
     * Request connect & response binds
     * control pub channel connect and sub binds
     */
    if info.isConnect {
        err = sock.Connect(info.address)
    } else {
        err = sock.Bind(info.address)
    }

    if err != nil {
        err = cmn.LogError("Failed to bind/connect mode(%d) info(%+v) err(%v)", mode, *info, err)
    } else if err = sock.SetLinger(time.Duration(100) * time.Millisecond); err != nil {
        /* Context termination will sleep this long, for any message drain */
        err = cmn.LogError("Failed to call set linger mode(%d) chType(%d) info(%+v) err(%v)",
            mode, chType, *info, err)
    } else {
        socketsList.Store(sock, fmt.Sprintf("mode(%d)_chType(%d)_(%s)", mode, chType, requester))
    }
    return
}

func closeSocket(s *zmq.Socket) {
    if s != nil {
        s.Close()
        socketsList.Delete(s)
        /* In case terminate context is waiting */
        select {
        case chSocksClose <- 1:
        default:
            cmn.LogError("Unable to write into chSocksClose")
        }
    }
}

/*
 * managePublish
 *
 * Manages a publish ZMQ channel for a proc for a channel type.
 * Creates the socket. Connect to corresponding XSUB point.
 * Sleeps on i/p channel for data to publish.
 * The data is expected as JSON string.
 * Runs until either system is being shutdown or the i/p request channel
 * is closed, whichever occurs early.
 *
 * Caller invokes this as a Go routine.
 *
 * Input:
 *  chType - Type of channel
 *  topic - Topic to prefic every publish data. A subscriber may use this
 *          for selective hearing.
 *  chReq - I/p channel for incoming publish data
 *  chRet - Send any error or nil, before diving into forever loop.
 *          The caller wait until it gets error value
 *
 * Output:
 *      None
 *
 * Return:
 *  Nothig as it is invoked as go routine.
 */
func managePublish(chType ChannelType_t, topic string, chReq <-chan JsonString_t,
    chRet chan<- error, cleanupFn func()) {

    defer cleanupFn()

    requester := fmt.Sprintf("publisher_topic(%s)_type(%d)", topic, chType)
    sock, err := getSocket(CHANNEL_MODE_PUBLISHER, chType, requester)

    defer closeSocket(sock)

    /* Inform the caller that function has initialized successfully or not */
    chRet <- err
    /* Sender close the channel */
    close(chRet)

    if err != nil {
        return
    }

    /* From here on the routine runs forever until shutdown */

    chShutdown := cmn.RegisterForSysShutdown(fmt.Sprintf(
        "ZMQ-Publisher. chType={%s}", CHANNEL_TYPE_STR[chType]))

    cmn.LogDebug("Started managePublish for chType=(%s)", CHANNEL_TYPE_STR[chType])

    /*
     * ZMQ connect/bind is asynchronous. There is no indication on when it
     * completes. Any data written into this socket before connect/bind completes
     * is silently dropped.
     * So make an explicit pause.
     * If not, you may create a dummy connection to a REQ socket where REP end
     * is expected and make a transaction. REQ & REP are synchronous. The time
     * taken is *most* likely sufficient for PUB socket connection completion.
     *
     * In either case, it is a pause. As well make it simpler.
     */
    time.Sleep(ZMQ_ASYNC_CONNECT_PAUSE)

    for {
        select {
        case <-chShutdown:
            cmn.LogInfo("Shutting down publisher")
            return

        case data, ok := <-chReq:
            if !ok {
                cmn.LogWarning("(%s) i/p channel closed. No more publish possible",
                    requester)
                return
            }
            if _, err = sock.SendMessage(topic, data); err != nil {
                /* Don't return; Just log error */
                cmn.LogError("Failed to publish err(%v) requester(%s) data(%s)",
                    err, requester, data)
            }
        }
    }
    cmn.LogDebug("Stopped managePublish for chType=(%s)", CHANNEL_TYPE_STR[chType])
}

/*
 * manageSubscribe
 *
 * Manages a Subscribe ZMQ channel for a proc for a channel type.
 * Creates the socket. Connect to corresponding XPUB point.
 * Sleeps on zmq Sub point for data to send to client.
 * The data is expected as JSON string.
 * Runs until either system is being shutdown.
 *
 * Caller invokes this as a Go routine.
 *
 * Input:
 *  chType - Type of channel
 *  topic - Topic to filter incoming publish data by. An empty string receives all.
 *  chRes - O/p channel for sending received data
 *  chRet - Send any error or nil, before diving into forever loop.
 *          The caller wait until it gets error value
 *
 * Output:
 *      None
 *
 * Return:
 *  Nothig as it is invoked as go routine.
 */
/*
 * As the routine, sleeps forever on ZMQ sub path for incoming data as blocked,
 * it is tought to abort on system shutdown.
 *
 * To assist, we kickoff a routine that sleeps on shutdown and upon shutdown
 * send a dummy message for publish. This would wake up the reader which can
 * see if shutdown initiated or not.
 *
 * QUITTOPIC is the message sent. Make sure the subscription is enabled to
 * receive this message.
 */
const QUITTOPIC = "quit"

func manageSubscribe(chType ChannelType_t, topic string, chRes chan<- JsonString_t,
    chRet chan<- error, cleanupFn func()) {

    defer cleanupFn()
    defer close(chRes)

    requester := fmt.Sprintf("subscriber_topic(%s)_type(%d)", topic, chType)
    sock, err := getSocket(CHANNEL_MODE_SUBSCRIBER, chType, requester)

    defer closeSocket(sock)

    if err != nil {
        /* err is good enough */
    } else if err = sock.SetSubscribe(topic); err != nil {
        err = cmn.LogError("Failed to subscribe filter(%s) err(%v)", topic, err)
    } else if topic != "" {
        /* To receive alert message on shutdown */
        if err = sock.SetSubscribe(QUITTOPIC); err != nil {
            err = cmn.LogError("Failed to subscribe filter(%s) err(%v)", QUITTOPIC, err)
        }
    }

    if err != nil {
        /* Inform the caller the failure to init and terminate this routine.*/
        chRet <- err
        close(chRet)
        return
    }

    shutDownRequested := false
    chShutErr := make(chan error) /* Track init error in following go func */

    /* Rouitine to publish dummy message on system shutdown */
    go func() {
        /*
         * Pre-create a publisher channel to alert subscribing channel
         * on shutdown.
         * You can't create a socket upon shutdown process start. So get it ahead.
         */
        shutSock, err := getSocket(CHANNEL_MODE_PUBLISHER, chType,
            fmt.Sprintf("To_Shut_%s", requester))

        /* Let the caller know error status */
        chShutErr <- err
        close(chShutErr)
        if err != nil {
            /* Terminate this routine */
            cmn.LogError("Alert go routine failed to get pub sock (%v)", err)
            return
        }

        defer closeSocket(shutSock)

        chShutdown := cmn.RegisterForSysShutdown(fmt.Sprintf(
            "ZMQ-Subscriber. chType={%s}", CHANNEL_TYPE_STR[chType]))

        /* Wait for shutdown signal */
        <-chShutdown
        shutDownRequested = true

        /* Send a message to wake up subscriber */
        if _, err = shutSock.SendMessage(QUITTOPIC, ""); err != nil {
            cmn.LogError("Failed to send quit to %s", requester)
        }
    }()

    /* read any error or nil from the above go func's init */
    err = <-chShutErr
    chRet <- err /* Forward the final init status to caller of this routine */
    close(chRet) /* Sender close the channel */

    if err != nil {
        return
    }

    cmn.LogDebug("Started manageSubscribe for chType=(%s) topic(%s)", CHANNEL_TYPE_STR[chType], topic)
    /* From here on the routine runs forever until shutdown */
    for {
        if data, e := sock.RecvMessage(0); e != nil {
            cmn.LogError("Failed to receive msg err(%v) for (%s)", e, requester)
        } else if len(data) != 2 {
            cmn.LogError("Expect 2 parts. requester(%s) data(%v)", requester, data)
        } else if shutDownRequested {
            cmn.LogInfo("Subscriber shutting down requester:(%s)", requester)
            /* Writer close the channel */
            return
        } else {
            /* Handle possibility of no one to read message */
            select {
            case chRes <- JsonString_t(data[1]):
                /* There is an active reader */
            case <-time.After(SUB_CHANNEL_TIMEOUT):
                /* No reader.Close this channel */
                cmn.LogInfo("Closing channel (%s) for no active reader. timeout(%v)",
                    requester, SUB_CHANNEL_TIMEOUT.Seconds())
                return
            }
        }
    }
    cmn.LogDebug("Stopped manageSubscribe for chType=(%s)", CHANNEL_TYPE_STR[chType])
}

/*
 * runPubSubChannel
 *
 * Meant for publisher & subscriber
 *
 * The created channels run forever ready for publishing/subscribing.
 * They run forever until system shutdown.
 *
 * As dedicated routines run forever, we don't allow duplicates
 * more to conserve resources. Hence no close call.
 *
 * chData could be used by multiple client routines.
 *
 * Input:
 *  mode -  Publisher or subscriber
 *  chType -Type of data like events, counters, red-button.
 *          Each type has a dedicated channel
 *  topic - Topic for publishing, which subscriber could use to filter upon.
 *
 *  chData -It is used as i/p channel for publish data and as o/p channel
 *          for data read from subscription
 *
 * Output:  None
 *
 * Return: Error as nil or non nil
 */

/* Map[id]bool to avoid duplicate open channels, which will drain resources */
var openChannels = sync.Map{}

func openChannel(mode channelMode_t, chType ChannelType_t, topic string,
    chData chan JsonString_t) (err error) {

    /* Sockets are opened per chType */
    id := fmt.Sprintf("%d_%d", mode, chType)
    if _, ok := openChannels.Load(id); ok {
        err = cmn.LogError("Duplicate req mode=%d chType=%d topic=%s pre-exists",
            mode, chType, topic)
        return
    }

    switch mode {
    case CHANNEL_MODE_PUBLISHER, CHANNEL_MODE_SUBSCRIBER:
    default:
        err = cmn.LogError("Expect mode (%d) as pub/sub only", mode)
        return
    }

    if _, err = getContext(); err != nil {
        return
    }
    chRet := make(chan error)
    openChannels.Store(id, true)

    if mode == CHANNEL_MODE_PUBLISHER {
        go managePublish(chType, topic, chData, chRet,
            func() { openChannels.Delete(id) })
    } else {
        go manageSubscribe(chType, topic, chData, chRet,
            func() { openChannels.Delete(id) })
    }

    /* Wait till routines get their init done */
    err = <-chRet
    return
}

func runPubSubProxyInt(chType ChannelType_t, chRet chan<- error) {
    var sock_xsub *zmq.Socket
    var sock_xpub *zmq.Socket
    var sock_ctrl_sub *zmq.Socket
    var err error

    defer func() {
        if sock_xsub != nil {
            sock_xsub.Close()
        }
        if sock_xpub != nil {
            sock_xpub.Close()
        }
        if chRet != nil {
            chRet <- err
            close(chRet)
        }
        /* Close tracked sockets appropriately */
        closeSocket(sock_ctrl_sub)
    }()

    var ctx *zmq.Context

    if ctx, err = getContext(); err != nil {
        return
    }

    /*
     * Note: We don't track xsub & xpub in socketsList as they are controlled
     * control socket, which is tracked.
     */
    var info *sockInfo_t

    if sock_xsub, err = ctx.NewSocket(zmq.XSUB); err != nil {
        err = cmn.LogError("Failed to get zmq xsub socket (%v)", err)
    } else if sock_xpub, err = ctx.NewSocket(zmq.XPUB); err != nil {
        err = cmn.LogError("Failed to get zmq xpub socket (%v)", err)
    } else if info, err = getAddress(CHANNEL_MODE_PUBLISHER, chType); err != nil {
        /* err is well described */
    } else if err = sock_xsub.Bind(info.address); err != nil {
        err = cmn.LogError("Failed to bind xsub socket (%v) address(%s)", err, info.address)
    } else if info, err = getAddress(CHANNEL_MODE_SUBSCRIBER, chType); err != nil {
        /* err is well described */
    } else if err = sock_xpub.Bind(info.address); err != nil {
        err = cmn.LogError("Failed to bind xpub socket (%v)", err)
    } else if sock_ctrl_sub, err = getSocket(CHANNEL_MODE_PROXY_CTRL_SUB, CHANNEL_TYPE_NA,
        "ctrl-sub-for-proxy"); err != nil {
        err = cmn.LogError("Failed to setup proxy err(%v)", err)
    }

    if err != nil {
        return
    }

    chShutErr := make(chan error) /* Track init error in following go func */

    go func() {
        /*
         * Pre-create a publisher channel to alert subscribing channel
         * on shutdown.
         * You can't create a socket upon shutdown process start. So get it ahead.
         */
        var sock_ctrl_pub *zmq.Socket
        defer closeSocket(sock_ctrl_pub)

        if sock_ctrl_pub, err = getSocket(CHANNEL_MODE_PROXY_CTRL_PUB, CHANNEL_TYPE_NA,
            "ctrl-pub-for-proxy"); err != nil {
            err = cmn.LogError("Failed to create proxy control publisher to terminate proxy(%v)", err)
        }
        chShutErr <- err
        close(chShutErr)
        if err != nil {
            /* Terminate this routine */
            cmn.LogError("Alert go routine failed to get ctrl pub sock (%v)", err)
            return
        }

        /* Watch for shutdown */
        chShutdown := cmn.RegisterForSysShutdown(fmt.Sprintf(
            "PubSubProxy chType={%s}", CHANNEL_TYPE_STR[chType]))
        <-chShutdown

        /* Terminate proxy. Just a write breaks the zmq.Proxy loop. */
        if _, err = sock_ctrl_pub.Send("TERMINATE", 0); err != nil {
            cmn.LogError("Failed to write proxy control publisher to terminate proxy(%v)", err)
        }
    }()

    err = <-chShutErr
    if err != nil {
        return
    }
    /* Inform caller successful init */
    chRet <- nil
    close(chRet)
    chRet = nil

    cmn.LogDebug("Started zmq.ProxySteerable for chType=(%s)", CHANNEL_TYPE_STR[chType])
    /* Run until shutdown which is indicated via ctrl socket */
    if err = zmq.ProxySteerable(sock_xsub, sock_xpub, nil, sock_ctrl_sub); err != nil {
        cmn.LogError("Failing to run zmq.Proxy err(%v)", err)
    }
    cmn.LogDebug("Stopped zmq.ProxySteerable for chType=(%s)", CHANNEL_TYPE_STR[chType])
    return
}

/* Map of chType vs bool */
var runningPubSubProxy = sync.Map{}

func doRunPubSubProxy(chType ChannelType_t) error {
    if _, ok := runningPubSubProxy.Load(chType); ok {
        return cmn.LogError("Duplicate runPubSubProxy for chType(%d)", chType)
    }
    chRet := make(chan error)
    go runPubSubProxyInt(chType, chRet)
    err := <-chRet
    if err == nil {
        runningPubSubProxy.Store(chType, true)
    }
    return err
}

/*
 * clientRequestHandler
 *
 * A single handler per process to stream in all client requests to server
 * via req/rep zmq channel and return corresponding response to channel
 * associated with the request.
 *
 * A go routine per request type
 *
 * Input:
 *  reqType - Type of request
 *  chReq - Channel to read incoming requests
 *  chRet - Way to return to caller any error associated with initialization.
 *
 * Output:
 *  None
 *
 * Return:
 *  None -- As it is a go routine forever until system shutdown
 */
type reqInfo_t struct {
    reqData   ClientReq_t
    chResData chan<- *ClientRes_t
}

func clientRequestHandler(reqType ChannelType_t, chReq <-chan *reqInfo_t,
    chRet chan<- error) {

    requester := fmt.Sprintf("clientRequestHandler_type(%d)", reqType)
    sock, err := getSocket(CHANNEL_MODE_REQUEST, reqType, requester)

    defer closeSocket(sock)

    /* Inform the caller that function has initialized successfully or not */
    chRet <- err
    /* Sender close the channel */
    close(chRet)

    if err != nil {
        return
    }

    /* From here on the routine runs forever until shutdown */

    chShutdown := cmn.RegisterForSysShutdown(fmt.Sprintf(
        "clientRequestHandler reqType={%s}", CHANNEL_TYPE_STR[reqType]))

    for {
        select {
        case <-chShutdown:
            cmn.LogInfo("Shutting down %s", requester)
            return

        case data := <-chReq:
            rcvData := ""
            if _, err = sock.Send(string(data.reqData), 0); err != nil {
                /* Don't return; Just log error */
                err = cmn.LogError("Failed to send request err(%v) requester(%s) data(%s)",
                    err, requester, data)
            } else if rcvData, err = sock.Recv(0); err != nil {
                err = cmn.LogError("Failed to recv response err(%v) requester(%s)", err, requester)
            }

            data.chResData <- &ClientRes_t{ServerRes_t(rcvData), err}
            close(data.chResData) /* No more writes as it is per request */
        }
    }
}

/*
 * getclientReqChan
 *
 * We open one channel per request type. The opened channel live until
 * system shutdown. Hence cache the opened channels for all future requests
 * of same type.
 *
 * Input:
 *  reqType - Type of request
 *
 * Output:
 *  None
 *
 * Return:
 *  ch - A writable of type *reqInfo_t. It carries req data and a channel to
 *       to get the response back.
 *  err - Error value
 */
/*  sync.Map[ChannelType_t]chan<- *reqInfo_t */
var clientReqChanList = sync.Map{}

func getclientReqChan(reqType ChannelType_t) (chReq chan<- *reqInfo_t, err error) {
    if v, ok := clientReqChanList.Load(reqType); !ok {
        ch := make(chan *reqInfo_t)
        chRet := make(chan error)
        go clientRequestHandler(reqType, ch, chRet)
        err = <-chRet
        if err == nil {
            chReq = ch
            clientReqChanList.Store(reqType, chReq)
        }
    } else if ch, ok := v.(chan<- *reqInfo_t); !ok {
        err = cmn.LogError("Internal error. Type(%T) != chan<- *reqInfo_t", v)
    } else {
        chReq = ch
    }
    return
}

/*
 * processRequest
 *
 * Send a client request to handler and get a channel to read the
 * response asynchronously.
 *
 * Input:
 *  req - Request to send
 *
 * Output:
 *  None
 *
 * Return:
 *  ch  - Channel to read response
 *  err - Error object
 */
func processRequest(reqType ChannelType_t, req ClientReq_t, chRes chan<- *ClientRes_t) (err error) {
    if ch, e := getclientReqChan(reqType); e == nil {
        ch <- &reqInfo_t{req, chRes}
    } else {
        err = e
    }
    return
}

/*
 * A handler register for certain req types.
 *
 * All requests of that type will be sent to it via req channel and expect
 * response via resp channel.
 *
 * Input:
 *  chType - Request type to handle
 *
 * Output:
 *  None
 *
 * Return:
 *  chReq - Channel for incoming requests
 *  chRes - channel for returning responses
 *  err - nil on failure
 */

func serverRequestHandler(reqType ChannelType_t, chReq chan<- ClientReq_t,
    chRes <-chan ServerRes_t, chRet chan<- error) {

    requester := fmt.Sprintf("server_request_handler_type(%d)", reqType)
    var sock *zmq.Socket
    var err error
    defer closeSocket(sock)

    if sock, err = getSocket(CHANNEL_MODE_RESPONSE, reqType, requester); err != nil {
        /* Inform the caller the failure to init and terminate this routine.*/
        chRet <- err
        close(chRet)
        return
    }

    /*
     * As this sleeps on chReq channel, have a dedicated routine to
     * watch for system shutdown and send mock request to wake it up
     * and hence enable to see the shutdown in progress.
     */
    shutDownRequested := false
    chShutErr := make(chan error) /* Track init error in following go func */

    go func() {
        /*
         * Pre-create qa publisher channel to alert subscribing channel
         * on shutdown.
         * You can't create a socket upon shutdown process start. So get it ahead.
         */
        shutSock, err := getSocket(CHANNEL_MODE_REQUEST, reqType,
            fmt.Sprintf("To_Shut_%s", requester))
        chShutErr <- err
        close(chShutErr)
        if err != nil {
            /* Terminate this routine */
            return
        }

        /* err == nil, hence socket is valid hence add defer for close */
        defer closeSocket(shutSock)

        chShutdown := cmn.RegisterForSysShutdown(fmt.Sprintf(
            "serverRequestHandler reqType={%s}", CHANNEL_TYPE_STR[reqType]))

        /* Wait for shutdown signal */
        <-chShutdown
        shutDownRequested = true

        /* Send a message to wake up subscriber */
        if _, err = shutSock.Send(QUITTOPIC, 0); err != nil {
            cmn.LogError("Failed to send quit to %s", requester)
        }
    }()

    /* read any error or nil from the above go func's init */
    err = <-chShutErr
    chRet <- err /* Forward the final init status to caller of this routine */
    close(chRet) /* Sender close the channel */
    if err != nil {
        return
    }

    /* From here on the routine runs forever until shutdown */
    for {
        /* Receive request from client's REQ socket which is ClientReq_t */
        rcvData := ""
        if rcvData, err = sock.Recv(0); err != nil {
            cmn.LogError("Failed to receive msg err(%v for (%s)", requester)
        } else if shutDownRequested {
            cmn.LogInfo("Subscriber shutting down requester:(%s)", requester)
            /* Writer close the channel */
            close(chReq)
            break
        } else {
            chReq <- ClientReq_t(rcvData)
            resData := <-chRes
            if _, err = sock.Send(string(resData), 0); err != nil {
                cmn.LogError("Failed to send response err(%v) req(%s) res(%s)", err, rcvData, resData)
            }
        }
    }
}

var serverReqHandlerList = sync.Map{}

func initServerRequestHandler(reqType ChannelType_t, chReq chan<- ClientReq_t,
    chRes <-chan ServerRes_t) (err error) {
    if _, ok := serverReqHandlerList.Load(reqType); !ok {
        chRet := make(chan error)
        go serverRequestHandler(reqType, chReq, chRes, chRet)
        err = <-chRet
        if err == nil {
            serverReqHandlerList.Store(reqType, true)
        }
    } else {
        return cmn.LogError("Duplicate initServerRequestHandler for reqType=(%d)", reqType)
    }
    return
}
