package lomtelemetry

import (
    cmn "lom/src/lib/lomcommon"

    "fmt"
    "log/syslog"
    "sync"
    "syscall"
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

var HALF_SECOND = time.Duration(500) * time.Millisecond

var SOCK_SND_TIMEOUT = HALF_SECOND /* Send max blocks .5 sec */
var SOCK_RCV_TIMEOUT = HALF_SECOND /* Recv max blocks .5 sec */

var RES_CHANNEL_TIMEOUT = HALF_SECOND /* Client req handler timeout to write res */
var SUB_CHANNEL_TIMEOUT = HALF_SECOND /* Sub handler timeout to write rcvd data */

var ZMQ_ASYNC_CONNECT_PAUSE = time.Duration(300) * time.Millisecond

/* Logical grouping of ChannelType_t values for validation use */
type chTypes_t map[ChannelType_t]bool

var pubsub_types = chTypes_t{
    CHANNEL_TYPE_EVENTS:    true,
    CHANNEL_TYPE_COUNTERS:  true,
    CHANNEL_TYPE_REDBUTTON: true,
}

var reqrep_types = chTypes_t{
    CHANNEL_TYPE_ECHO:     true,
    CHANNEL_TYPE_SCS:      true,
    CHANNEL_TYPE_TEST_REQ: true,
}

const (
    SOCK_MODE_SEND = 1
    SOCK_MODE_RECV = 2
)

func getSockMode(sType zmq.Type) int {
    switch sType {
    case zmq.PUB:
        return SOCK_MODE_SEND
    case zmq.SUB:
        return SOCK_MODE_RECV
    case zmq.REQ, zmq.REP:
        return SOCK_MODE_SEND | SOCK_MODE_RECV
    default:
        cmn.LogPanic("Unknown socket type (%v)", sType)
        return 0
    }
}

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
    CHANNEL_MODE_PROXY_CTRL_PUB: chModeData_t{pubsub_types, ZMQ_PROXY_CTRL_PORT, zmq.PUB, false},
    CHANNEL_MODE_PROXY_CTRL_SUB: chModeData_t{pubsub_types, ZMQ_PROXY_CTRL_PORT, zmq.SUB, true},
}

type sockInfo_t struct {
    address   string
    sType     zmq.Type
    isConnect bool
}

/* Global variables tracking all active objects */
/*
 * Single context shared by all threads & routines.
 * Ctx is threadsafe, but not sockets
 * Hence one context per process.
 */
var zctx *zmq.Context

/*
 * Each socket close writes into this channel
 * During shutdown term contex sleep on this channel until all
 * all sockets are closed, hence the sockets list is empty.
 */
var chSocksClose = make(chan int)

/*
 * Track all open sockets.
 * Terminate context blocks until this goes 0
 */
var socketsList = sync.Map{}

/* Map[id]bool to avoid duplicate open channels, which will drain resources */
var pubChannels = sync.Map{}

/* Map[id]bool to avoid duplicate open channels, which will drain resources */
var subChannels = sync.Map{}

/* Map of chType vs bool */
var runningPubSubProxy = sync.Map{}

/*  sync.Map[ChannelType_t]chan *reqInfo_t */
var clientReqChanList = sync.Map{}

var serverReqHandlerList = sync.Map{}

var globalHandlesMaps = map[string]*sync.Map{
    "socketsList":          &socketsList,
    "pubChannels":          &pubChannels,
    "subChannels":          &subChannels,
    "runningPubSubProxy":   &runningPubSubProxy,
    "clientReqChanList":    &clientReqChanList,
    "serverReqHandlerList": &serverReqHandlerList,
}

func isZMQIdle() (ret bool) {
    ret = true
    if zctx != nil {
        i := 0
        for k, m := range globalHandlesMaps {
            m.Range(func(e, v any) bool { i++; return true })
            if i != 0 {
                ret = false
                cmn.LogInfo("ZMQ active for (%s) with cnt(%d)", k, i)
                break
            }
        }
    }
    return
}

func getAddress(mode channelMode_t, chType ChannelType_t) (sockInfo *sockInfo_t, err error) {

    /* Cross validation between mode & ChannelType_t */
    info, ok := chModeInfo[mode]
    if ok {
        _, ok = info.types[chType]
    }
    if !ok {
        err = cmn.LogError("Unknown channel mode(%v) or type (%d)", mode, chType)
    } else {
        sockInfo = &sockInfo_t{
            fmt.Sprintf(ZMQ_ADDRESS, info.startPort+int(chType)),
            info.sType,
            info.isConnect}
        cmn.LogDebug("Address: (%+v) mode=(%v) chType(%s)\n", *sockInfo, mode, CHANNEL_TYPE_STR[chType])
    }
    return
}

/*
 * Collect all open sockets. Ctx termination is blocked by any
 * open socket.
 * string is some friendly identification of caller to help track
 * who is not closing, upon leak.
 */

func getContext() (*zmq.Context, error) {
    var err error
    if zctx == nil {
        if !cmn.IsSysShuttingDown() {
            zctx, err = zmq.NewContext()
        }
        if zctx == nil {
            err = cmn.LogError("Failed to get zmq context (%v) IsSysShuttingDown(%v)",
                err, cmn.IsSysShuttingDown())
        } else {
            /* Terminate on system shutdown */
            go terminateContext()
        }
    }
    return zctx, err
}

func terminateContext() {
    shutdownId := "terminate ZMQ context"
    chShutdown := cmn.RegisterForSysShutdown(shutdownId)

    defer func() {
        cmn.DeregisterForSysShutdown(shutdownId)
    }()

shutLoop:
    /* Sleep till shutdown */
    for {
        select {
        case <-chShutdown:
            break shutLoop
        case <-chSocksClose:
            /*
             * Some socket closed. Nothing to do
             * Yet must read to drain, else writer blocks.
             */
            cmn.LogDebug("A socket closed")
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
    if cmn.IsSysShuttingDown() || zctx == nil {
        return nil, cmn.LogError("System is shutting down. No new socket")
    }
    var info *sockInfo_t

    if info, err = getAddress(mode, chType); err == nil {
        sock, err = zctx.NewSocket(info.sType)
    }

    if err != nil {
        err = cmn.LogError("Failed to get socket sock(%p) info(%+v) mode(%v) type(%v) err(%v)",
            sock, info, mode, chType, err)
        return
    }

    defer func() {
        if err != nil {
            sock.Close()
            sock = nil
        }
    }()

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
    if err == nil {
        err = sock.SetLinger(time.Duration(100) * time.Millisecond)
        /* Context termination will sleep this long, for any message drain */
    }

    txMode := getSockMode(info.sType)
    if err == nil {
        if (txMode & SOCK_MODE_SEND) == SOCK_MODE_SEND {
            err = sock.SetSndtimeo(SOCK_SND_TIMEOUT)
        }
    }
    if err == nil {
        if (txMode & SOCK_MODE_RECV) == SOCK_MODE_RECV {
            err = sock.SetRcvtimeo(SOCK_RCV_TIMEOUT)
        }
    }
    if err == nil {
        socketsList.Store(sock, fmt.Sprintf("mode(%d)_chType(%d)_(%s)", mode, chType, requester))
        cmn.LogDebug("getSocket: sock(%v) requester(%s)", sock, requester)
    } else {
        err = cmn.LogError("Failed to bind/connect sock(%p) mode(%d) info(%+v) err(%v)",
            sock, mode, info, err)
    }
    return
}

func closeSocket(s *zmq.Socket) {
    if s != nil {
        if r, ok := socketsList.Load(s); !ok {
            cmn.LogMessageWithSkip(1, syslog.LOG_ERR, "***INTERNAL ERROR*** calling for non-existing sock(%p)(%v)", s, s)
        } else {
            cmn.LogDebug("List: closeSocket: sock(%p)(%v) r=(%v)", s, s, r)
            socketsList.Delete(s)
        }
        s.Close()
        /* In case terminate context is waiting */
        chSocksClose <- 1
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
 * GoRoutune:
 *  Yes.
 *  shutdown:
 *      1. On System shutdown
 *      2. On close of its i/p channel chReq
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

    /* Inform the caller that function has initialized successfully or not */
    chRet <- err
    /* Sender close the channel */
    close(chRet)

    if err != nil {
        return
    }

    defer closeSocket(sock)

    /* From here on the routine runs forever until shutdown */
    shutdownId := fmt.Sprintf("ZMQ-Publisher. chType={%s}", CHANNEL_TYPE_STR[chType])
    chShutdown := cmn.RegisterForSysShutdown(shutdownId)

    defer func() {
        cmn.DeregisterForSysShutdown(shutdownId)
    }()

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

Loop:
    for {
        select {
        case <-chShutdown:
            cmn.LogInfo("Shutting down publisher on system shutdown")
            break Loop

        case data, ok := <-chReq:
            if !ok {
                cmn.LogInfo("(%s) i/p channel closed. No more publish possible",
                    requester)
                break Loop
            }
            if _, err = sock.SendMessage(topic, data); err != nil {
                /*
                 * Error could be timeout as we set SNDTIMEO. But we set it for
                 * 0.5 second (SOCK_SND_TIMEOUT). So no point in retrying even if
                 * it is timeout which comes as zmq.Errno(syscall.EAGAIN)
                 */
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
 * GoRoutune:
 *  Yes.
 *  shutdown:
 *      1. On System shutdown
 *      2. On close of chCtrl
 *  The blocking read on sock has timeout. Hence on every timeout, it
 *  checks for the above 2 shutdown triggers.
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

func manageSubscribe(chType ChannelType_t, topic string, chRes chan<- JsonString_t,
    chCtrl <-chan int, chRet chan<- error, cleanupFn func()) {

    defer func() {
        cleanupFn()
        close(chRes)
    }()

    requester := fmt.Sprintf("subscriber_topic(%s)_type(%d)", topic, chType)
    sock, err := getSocket(CHANNEL_MODE_SUBSCRIBER, chType, requester)

    defer closeSocket(sock)

    if err == nil {
        err = sock.SetSubscribe(topic)
    }

    if err != nil {
        err = cmn.LogError("Failed to init sub socket sock(%p) err(%v) topic(%s)",
            sock, err, topic)
    }
    /* Inform the caller the state of init */
    chRet <- err
    close(chRet)

    if err != nil {
        return
    }

    /* Register for system shutdown */
    shutdownId := fmt.Sprintf("ZMQ-Subscriber. chType={%s}", CHANNEL_TYPE_STR[chType])
    chShutdown := cmn.RegisterForSysShutdown(shutdownId)

    defer func() {
        cmn.DeregisterForSysShutdown(shutdownId)
    }()

    cmn.LogDebug("Started manageSubscribe for chType=(%s) topic(%s)", CHANNEL_TYPE_STR[chType], topic)

    /* From here on the routine runs forever until shutdown */
Loop:
    for {
        /* Check for shutdown at start of loop */
        select {
        case <-chShutdown:
            cmn.LogInfo("Subscriber shutting down requester:(%s)", requester)
            break Loop
        case <-chCtrl:
            cmn.LogInfo("Subscriber control channel closed: (%s)", requester)
            break Loop
        default:
        }

        if data, e := sock.RecvMessage(0); e == zmq.Errno(syscall.EAGAIN) {
            /* Continue the loop. RCVTIMEO  is set for SOCK_RCV_TIMEOUT */
        } else if e != nil {
            cmn.LogError("Failed to receive msg err(%v) for (%s)", e, requester)
        } else if len(data) != 2 {
            cmn.LogError("Expect 2 parts. requester(%s) data(%v)", requester, data)
        } else {
            /* Handle possibility of no one to read message */
            select {
            case chRes <- JsonString_t(data[1]):
                /* There is an active reader */
            case <-time.After(SUB_CHANNEL_TIMEOUT):
                /* No reader. Drop the messsage */
                cmn.LogInfo("%s: Dropped message for no reader after (%d) seconds",
                    requester, SUB_CHANNEL_TIMEOUT.Seconds())
            }
        }
    }
    cmn.LogDebug("Stopped manageSubscribe for chType=(%s)", CHANNEL_TYPE_STR[chType])
}

/*
 * openPubChannel
 *
 * The created channels run forever ready for publishing.
 * They run forever until system shutdown or write end of the channel is closed.
 *
 * chData could be used by multiple client routines.
 *
 * Input:
 *  chType -Type of data like events, counters, red-button.
 *          Each type has a dedicated channel
 *  topic - Topic for publishing, which subscriber could use to filter upon.
 *
 *  chData -It is used as i/p channel for publish data. Caller writes the data to publish.
 *
 * Output:  None
 *
 * Return: Error as nil or non nil
 */

func openPubChannel(chType ChannelType_t, topic string, chData <-chan JsonString_t) (err error) {

    /* Sockets are opened per chType */
    /* A publisher expected to use one topic only. So restricted per channel type */
    id := fmt.Sprintf("PubChanne:%d", chType)
    if _, ok := pubChannels.Load(id); ok {
        err = cmn.LogError("Duplicate req for pub channel chType=%d topic=%s pre-exists", chType, topic)
        return
    }

    if _, err = getContext(); err != nil {
        return
    }
    chRet := make(chan error)
    pubChannels.Store(id, true)

    go managePublish(chType, topic, chData, chRet, func() { pubChannels.Delete(id) })

    /* Wait till routines get their init done */
    err = <-chRet
    return
}

/*
 * openSubChannel
 *
 * The created channels run forever subscribing to given topic.
 * They run forever until system shutdown or ctrl channel is closed.
 *
 * Input:
 *  chType -Type of data like events, counters, red-button.
 *          Each type has a dedicated channel
 *  topic - Topic for subscription. An empty string receives all.
 *
 *  chData - This is writable channel where all received messages are written into.
 *  chCtrl - Closing this closes underlying network connection and hence cancel the
 *              subscription. Caller keeps the write end to close.
 *
 * Output:  None
 *
 * Return: Error as nil or non nil
 */

func openSubChannel(chType ChannelType_t, topic string, chData chan<- JsonString_t,
    chCtrl <-chan int) (err error) {

    /* Sockets are opened per chType */
    /* Callers are interested in all or a topic per channel type */
    id := fmt.Sprintf("SubChannel:%d", chType)
    if _, ok := subChannels.Load(id); ok {
        err = cmn.LogError("Duplicate req for sub channel chType=%d topic=%s pre-exists", chType, topic)
        return
    }

    if _, err = getContext(); err != nil {
        return
    }
    chRet := make(chan error)
    subChannels.Store(id, true)

    go manageSubscribe(chType, topic, chData, chCtrl, chRet, func() { subChannels.Delete(id) })

    /* Wait till routines get their init done */
    err = <-chRet
    return
}

/*
 * All publishers for a channel type connect to single XSub point.
 * All subscribers for a channel type connect to single XPub point.
 * A proxy is started per channel type to connect the XPub & XSub.
 * This proxy is a simple dumb & no-latency pipe.
 *
 * GoRoutune:
 *  Yes.
 *  shutdown:
 *      1. On System shutdown
 *      2. On close of chCtrl
 *  The proxy is started with Ctrl pub/sub sockets. A "TERMINATE" message on ctrl-pub
 *  will stop the proxy. A dedicated Go routine watches the above 2 shutdown venues
 *  and send Terminate on either.
 */
func runPubSubProxyInt(chType ChannelType_t, chCtrl <-chan int, chRet chan<- error, cleanupFn func()) {
    var sock_xsub *zmq.Socket
    var sock_xpub *zmq.Socket
    var sock_ctrl_sub *zmq.Socket
    var err error

    defer func() {
        cmn.LogDebug("Ending runPubSubProxyInt ....")
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
        cleanupFn()
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

    if sock_xsub, err = ctx.NewSocket(zmq.XSUB); err == nil {
        if info, err = getAddress(CHANNEL_MODE_PUBLISHER, chType); err == nil {
            err = sock_xsub.Bind(info.address)
        }
    }
    if err == nil {
        if sock_xpub, err = ctx.NewSocket(zmq.XPUB); err == nil {
            if info, err = getAddress(CHANNEL_MODE_SUBSCRIBER, chType); err == nil {
                err = sock_xpub.Bind(info.address)
            }
        }
    }
    if err == nil {
        if sock_ctrl_sub, err = getSocket(CHANNEL_MODE_PROXY_CTRL_SUB, chType,
            "ctrl-sub-for-proxy"); err == nil {
            err = sock_ctrl_sub.SetSubscribe("")
        }
    }
    if err != nil {
        err = cmn.LogError("Failed to get sock(%p, %p, %p) info(%+v) err(%v)",
            sock_xsub, sock_xpub, sock_ctrl_sub, info, err)
        return
    }

    chShutErr := make(chan error) /* Track init error in following go func */

    go func() {
        /*
         * Routine to signal proxy to go down on shutdown.
         *
         * Pre-create a publisher channel to alert subscribing channel
         * on shutdown.
         * You can't create a socket upon shutdown process start. So get it ahead.
         */
        var sock_ctrl_pub *zmq.Socket

        sock_ctrl_pub, err = getSocket(CHANNEL_MODE_PROXY_CTRL_PUB, chType, "ctrl-pub-for-proxy")

        chShutErr <- err
        close(chShutErr)
        if err != nil {
            /* Terminate this routine */
            cmn.LogError("Alert go routine failed to get ctrl pub sock (%v)", err)
            return
        }
        defer closeSocket(sock_ctrl_pub)

        /* Register for shutdown signal. */
        shutdownId := fmt.Sprintf("PubSubProxy chType={%s}", CHANNEL_TYPE_STR[chType])
        chShutdown := cmn.RegisterForSysShutdown(shutdownId)

        defer func() {
            cmn.DeregisterForSysShutdown(shutdownId)
        }()

        /* Watch for system/user shutdown */
        select {
        case <-chShutdown:
            cmn.LogInfo("proxy: System shutdown for (%s)", shutdownId)
        case <-chCtrl:
            cmn.LogInfo("proxy: User shutdown for (%s)", shutdownId)
        }
        cmn.LogInfo("Signalling down zmq proxy")

        /* Terminate proxy. Just a write breaks the zmq.Proxy loop. */
        if _, err = sock_ctrl_pub.Send("TERMINATE", 0); err != nil {
            cmn.LogError("Failed to write proxy control publisher to terminate proxy(%v)", err)
        } else {
            cmn.LogInfo("Signaled down zmq proxy")
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

func doRunPubSubProxy(chType ChannelType_t, chCtrl <-chan int) (err error) {
    if _, err = getContext(); err != nil {
        return err
    }
    if _, ok := runningPubSubProxy.Load(chType); ok {
        return cmn.LogError("Duplicate runPubSubProxy for chType(%d)", chType)
    }
    chRet := make(chan error)
    runningPubSubProxy.Store(chType, true)
    go runPubSubProxyInt(chType, chCtrl, chRet, func() { runningPubSubProxy.Delete(chType) })
    err = <-chRet
    return
}

/*
 * clientRequestHandler
 *
 * A single handler per process to stream in all client requests to server
 * via req/rep zmq channel and return corresponding response to channel
 * associated with the request.
 *
 * GoRoutune:
 *  Yes.
 *  shutdown:
 *      1. On System shutdown
 *      2. On close of chReq
 *  As this blocks on chan read, it watches both and shutsdown upon chan close
 *  of either.
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
    chRet chan<- error, cleanupFn func()) {

    requester := fmt.Sprintf("clientRequestHandler_type(%s)", CHANNEL_TYPE_STR[reqType])
    sock, err := getSocket(CHANNEL_MODE_REQUEST, reqType, requester)

    defer func() {
        closeSocket(sock)
        cleanupFn()
    }()

    if err == nil {
        /* Recv blocks on resp. Nake it smaller to enable not to miss future reqs */
        err = sock.SetRcvtimeo(time.Duration(50) * time.Millisecond)
    }
    if err != nil {
        err = cmn.LogError("Failed to init clientRequestHandler sock(%p) err(%v) requester(%s)",
            sock, err, requester)
    }

    /* Inform the caller that function has initialized successfully or not */
    chRet <- err
    /* Sender close the channel */
    close(chRet)

    if err != nil {
        return
    }

    /* From here on the routine runs forever until shutdown */

    shutdownId := fmt.Sprintf("clientRequestHandler reqType={%s}", CHANNEL_TYPE_STR[reqType])
    chShutdown := cmn.RegisterForSysShutdown(shutdownId)

    defer func() {
        cmn.DeregisterForSysShutdown(shutdownId)
    }()

    /* Run forever until shutdown */
    reqList := []*reqInfo_t{}
    tout := 0
    state_recv := false
    var req *reqInfo_t

Loop:
    for {
        select {
        case <-chShutdown:
            cmn.LogInfo("System Shutting down %s", requester)
            break Loop

        case data, ok := <-chReq:
            if !ok {
                cmn.LogInfo("I/p request channel closed. Shutting down for {%s}", requester)
                break Loop
            }
            reqList = append(reqList, data)
            /*
             * TODO: Add a timestamp to req entry as time of read.
             * If it can't be sent out within set timeout, abort & fail it
             * w/o sending to server.
             */
        case <-time.After(time.Duration(tout) * time.Second):
        }

        /* Process request */
        res := ""
        err = nil

        if !state_recv {
            if len(reqList) != 0 {
                tout = 0 /* No time to pause until all requests are processed */

                req = reqList[0]      /* Take first request */
                reqList = reqList[1:] /* Remove first */

                if _, err = sock.Send(string(req.reqData), 0); err != nil {
                    /*
                     * This could be timeout (EAGAIN). But timeout is 0.5 sec,
                     * so treat it as failure. On failure this req gets dropped.
                     */
                    /* Don't return; Just log error */
                    err = cmn.LogError("Failed to send request err(%v) requester(%s) data(%s)",
                        err, requester, req.reqData)
                    /* code below will send response */
                } else {
                    state_recv = true
                }
            } else {
                tout = 300 /* Nothing todo until request */
            }
        }
        if state_recv {
            /*
             * Request successfully sent. Wait for response. You can't timeout here
             * and move on to next, as it being REQ-REP sock, you can't send again
             * until we receive.
             */
            if res, err = sock.Recv(0); err == zmq.Errno(syscall.EAGAIN) {
                /* Do nothing. Until recv, nothing can be sent. So stay in this state. */
            } else {
                /* Receive complete with or w/o error */
                state_recv = false
            }
        }

        if !state_recv && req != nil {
            /* Reach here upon successfully receiving response or on send failure */
            select {
            case req.chResData <- &ClientRes_t{ServerRes_t(res), err}:
                /* Response sent back to caller via channel provided in request */
            case <-time.After(RES_CHANNEL_TIMEOUT):
                /*
                 * Client who gave this chan is neither reading it nor gave a
                 * buffered channel. Drop it and move on.
                 */
            }
            close(req.chResData) /* No more writes as it is per request */
            req = nil
        }
    }
    if req != nil {
        close(req.chResData) /* No more writes as it is per request */
    }
    for _, r := range reqList {
        close(r.chResData) /* All waiting clients gets unblocked */
    }
    cmn.LogInfo("Terminating clientRequestHandler for (%s)", requester)
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
func getclientReqChan(reqType ChannelType_t) (chReq chan<- *reqInfo_t, err error) {
    var ch chan *reqInfo_t
    if v, ok := clientReqChanList.Load(reqType); !ok {
        ch = make(chan *reqInfo_t)
        chRet := make(chan error)
        clientReqChanList.Store(reqType, ch)
        go clientRequestHandler(reqType, ch, chRet, func() { clientReqChanList.Delete(reqType) })
        err = <-chRet
    } else if ch, ok = v.(chan *reqInfo_t); !ok {
        err = cmn.LogError("Internal error. Type(%T) != chan<- *reqInfo_t", v)
    }
    if err == nil {
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
 *  reqType - Request type to sebd
 *  req - Request to send
 *  chRes - Channel to send response for this request.
 *
 * Output:
 *  None
 *
 * Return:
 *  ch  - Channel to read response
 *  err - Error object
 */
func processRequest(reqType ChannelType_t, req ClientReq_t, chRes chan<- *ClientRes_t) (err error) {
    if _, err = getContext(); err != nil {
        return
    }
    if ch, e := getclientReqChan(reqType); e == nil {
        ch <- &reqInfo_t{req, chRes}
    } else {
        err = e
    }
    return
}

/*
 * closeRequestChannel
 *
 * Close channel for given request type.
 *
 * Input:
 *  reqType - Request type to close
 *
 * Output:
 *  None
 *
 * Return:
 *  None
 */
func closeRequestChannel(reqType ChannelType_t) (err error) {
    if v, ok := clientReqChanList.Load(reqType); ok {
        if ch, ok := v.(chan *reqInfo_t); ok {
            clientReqChanList.Delete(reqType)
            close(ch)
            cmn.LogDebug("closed client req channel for type (%s)", CHANNEL_TYPE_STR[reqType])
        } else {
            err = cmn.LogError("val for req(%d) is incorrect type. (chan *reqInfo_t) != (%T)",
                reqType, v)
        }
    } else {
        err = cmn.LogError("Failed to find open chan for reqType(%d) for close", reqType)
    }
    return
}

/*
 * A handler register for certain req types.
 *
 * All requests of that type will be sent to it via req channel and expect
 * response via resp channel.
 *
 * Listens on sock for request. Pass read request to handler via chReq.
 * Wait for handler to write its response via chRes and write the same into sock.
 *
 * GoRoutune:
 *  Yes.
 *  shutdown:
 *      1. On System shutdown
 *      2. On close of chRes
 *  The blocking read on sock has timeout. Hence on every timeout, it
 *  checks for the above 2 shutdown triggers.
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
    chRes <-chan ServerRes_t, chRet chan<- error, cleanupFn func()) {

    requester := fmt.Sprintf("server_request_handler_type(%d)", reqType)
    var sock *zmq.Socket
    var err error

    sock, err = getSocket(CHANNEL_MODE_RESPONSE, reqType, requester)
    defer func() {
        closeSocket(sock)
        /* Writer close the channel */
        close(chReq)
        cleanupFn()
    }()

    if err != nil {
        err = cmn.LogError("Failed to init. sock(%p) requester(%s) err(%v)",
            sock, requester, err)
    }

    /* Inform the caller the result of init */
    chRet <- err
    close(chRet)
    if err != nil {
        /* Return as failed to init */
        return
    }

    /* Register for system shutdown */
    shutdownId := fmt.Sprintf("serverRequestHandler reqType={%s}", CHANNEL_TYPE_STR[reqType])
    chShutdown := cmn.RegisterForSysShutdown(shutdownId)

    defer func() {
        cmn.DeregisterForSysShutdown(shutdownId)
    }()

    /* From here on the routine runs forever until shutdown */

    /*
     * Read request from clients via socket
     * Socket calls are not blocking. They return with EAGAIN on timeout.
     *
     * Read request is sent to registered handler via channel chReq
     * This can *block* if handler is not reading it.
     *
     * Upon sending request, wait on response from handler via channel chRes.
     * reading is unblocking via select
     *
     * Upon reading write it back to client via socket. Socket write is async
     * hence the call does not block in any scenario.
     *
     * Shutdown could be initiated by closing
     *      chShutdown -- Done by system
     *      chRes -- By registered handler to de-register
     * So both these channels to be watched periodically.
     */
    type LState_t int
    const (
        LState_ReadReq  LState_t = iota /* Read via sock from client */
        LState_WriteReq                 /* Write req to server via chan */
        LState_ReadRes                  /* Read res from server via chan */
        /* LState_WriteRes -- Not needed as write is non-blocking */
    )
    ctState := LState_ReadReq

    rcvReq := ""
Loop:
    for {
        switch ctState {
        case LState_ReadReq:
            /* Stay here until read request / shutdown */
            for {
                /* Watch for shutdown or unsolicited response or close res channel */
                select {
                case <-chShutdown:
                    cmn.LogInfo("serverRequestHandler shutting down requester:(%s)", requester)
                    break Loop
                case v, ok := <-chRes:
                    cmn.LogInfo("input response channel closed/unsolicited ok=(%v) v(%v); Close this handler", ok, v)
                    break Loop
                default:
                }
                /* Receive blocks for SOCK_RCV_TIMEOUT */
                if rcvReq, err = sock.Recv(0); err == zmq.Errno(syscall.EAGAIN) {
                    /* Continue the loop */
                } else if err != nil {
                    cmn.LogError("Failed to receive msg err(%v for (%s)", requester)
                } else {
                    ctState = LState_WriteReq
                    break /* Break from this for loop */
                }
            }
        case LState_WriteReq:
            /* Wait here until successful write / shutdown */
            select {
            case <-chShutdown:
                cmn.LogInfo("serverRequestHandler shutting down requester:(%s)", requester)
                break Loop
            case v, ok := <-chRes:
                if !ok {
                    cmn.LogInfo("input response channel closed; Close this handler")
                } else {
                    cmn.LogError("Receiving response w/o request (%v)(%T)", v, v)
                }
                break Loop
            case chReq <- ClientReq_t(rcvReq):
                /* Send request to registered server side handler */
                ctState = LState_ReadRes
            }
        case LState_ReadRes:
            /* Stay here until response from handler or shutdown */
            select {
            case <-chShutdown:
                cmn.LogInfo("serverRequestHandler shutting down requester:(%s)", requester)
                break Loop
            case v, ok := <-chRes:
                if !ok {
                    cmn.LogInfo("input response channel closed; Close this handler")
                    break Loop
                } else {
                    if _, err = sock.Send(string(v), 0); err != nil {
                        cmn.LogError("Failed to send response err(%v) req(%s) res(%s)", err, rcvReq, v)
                    } else {
                        cmn.LogDebug("Sent response back to client (%v)(%T)", v, v)
                    }
                    ctState = LState_ReadReq
                }
            }
        }
    }
    cmn.LogInfo("serverRequestHandler terminating ctState=%d", ctState)
}

func initServerRequestHandler(reqType ChannelType_t, chReq chan<- ClientReq_t,
    chRes <-chan ServerRes_t) (err error) {
    if _, err = getContext(); err != nil {
        return
    }
    if _, ok := serverReqHandlerList.Load(reqType); !ok {
        chRet := make(chan error)
        serverReqHandlerList.Store(reqType, true)
        go serverRequestHandler(reqType, chReq, chRes, chRet,
            func() { serverReqHandlerList.Delete(reqType) })
        err = <-chRet
    } else {
        return cmn.LogError("Duplicate initServerRequestHandler for reqType=(%d)", reqType)
    }
    return
}
