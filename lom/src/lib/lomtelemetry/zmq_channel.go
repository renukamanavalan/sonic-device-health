package lomtelemetry

import (
    cmn "lom/src/lib/lomcommon"

    "encoding/json"
    "fmt"

    zmq "github.com/pebbe/zmq4"
)


/*
 * Each ChannelType_t uses a dedicated channel
 * Compute the port by adding chType to start port
 */
const ZMQ_REQ_REP_START_PORT = 5650
const ZMQ_XPUB_START_PORT = 5750
const ZMQ_XSUB_START_PORT = 5850
const ZMQ_PROXY_CTRL_PORT = 5950

const ZMQ_ADDRESS = "tcp://localhost:%d"

/* Logical grouping of ChannelType_t values for validation use */
type chTypes_t map[ChannelType_t]bool
var pubsub_types = chTypes_t{
    CHANNEL_TYPE_EVENTS: true,
    CHANNEL_TYPE_COUNTERS: true,
    CHANNEL_TYPE_REDBUTTON: true,
}

var reqrep_types = chTypes_t{
    CHANNEL_TYPE_ECHO: true,
    CHANNEL_TYPE_SCS: false,
}

var NA_types = chTypes_t{CHANNEL_TYPE_NA: true }

/* Mapping mode to acceptable types for validation */
var typesValidator = map[channelMode_t]chTypes_t {
    CHANNEL_PUBLISHER: pubsub_types,
    CHANNEL_SUBSCRIBER: pubsub_types,
    CHANNEL_REQUEST: reqrep_types,
    CHANNEL_RESPONSE: reqrep_types,
    CHANNEL_PROXY_CTRL_PUB: NA_types,
    CHANNEL_PROXY_CTRL_SUB: NA_types,
}

/* Is sytem shutdown initiated yet? */
var shutdownYet = false

func getAddress(mode channelMode, chType ChannelType) (path string, sType zmq.Type, err error) {
    port := 0

    /* Cross validation between mode & ChannelType_t */
    if types, ok := typesValidator[mode]; !ok {
        return cmn.LogError("Unknown channel mode (%v)", mode)
    } else if _, ok := types[chType]; !ok {
        return cmn.LogError("Unknown channel type (%d) for mode(%d)", chType, mode)
    }

    switch mode {
    case CHANNEL_PUBLISHER:
        port = ZMQ_XSUB_START_PORT + int(chType)
        sType = zmq.ZMQ_PUB
    case CHANNEL_SUBSCRIBER:
        port = ZMQ_XPUB_START_PORT + int(chType)
        sType = zmq.ZMQ_SUB
    case CHANNEL_REQUEST:
        port = ZMQ_REQ_REP_PORT + int(chType)
        sType = zmq.ZMQ_REQ
    case CHANNEL_RESPONSE:
        port = ZMQ_REQ_REP_PORT + int(chType)
        sType = zmq.ZMQ_REP
    case CHANNEL_PROXY_CTRL_PUB:
        port = ZMQ_PROXY_CTRL_PORT
        sType = zmq.ZMQ_PUB
    case CHANNEL_PROXY_CTRL_SUB:
        port = ZMQ_PROXY_CTRL_PORT
        sType = zmq.ZMQ_SUB
    }
    path = fmt.Sprintf(ZMQ_ADDRESS, port)
    return
}

/*
 * Single context shared by all threads & routines.
 * Ctx is threadsafe, but not sockets
 * Hence one context per process.
 */
var zctx = nil

/*
 * Collect all open sockets. Ctx termination is blocked by any
 * open socket.
 * string is some friendly identification of caller to help track
 * who is not closing, upon leak.
 */
var socketsList = map[*zmq.Socket]string{}

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
    if zctx == nil {
        if zctx, err := zmq.NewContext(); err != nil {
            return nil, cmn.LogError("Failed to get zmq context (%v)", err)
        }
        /* Terminate on system shutdown */
        go terminateContext()
    }   
    return zctx, nil 
}   

func terminateContext() {
    chShutdown := RegisterForSysShutdown("terminate context")

    /* Sleep till shutdown */
    for !shutdownYet {
        select {
        case <- chShutdown:
            shutdownYet = true
        case <- chSocksClose:
            /*
             * Some socket closed. Nothing to do */
             * Yet must read, else writer blocks as it is not buffered channel
             */
        }
    }

    /* System shutdown initiated; Wait for open sockets to close */
    for len(socketsList) != 0 {
        var pending []string
        for _, v := range socketsList {
            pending = append(pending, v)
        }
        cmn.LogError("Waiting for [%d] socks to close pending(%v)", len(pending), pending)

        /* Sleep until someone closes or timeout */
        select {
        case <- time.After(time.Second):
            cmn.LogError("Timeout upon waiting; exiting w/o context termination")
            return
        case <- chSocksClose:
            /* go back & check the list */
        }
    }
    cmn.LogInfo("terminating context. pending(%v)", pending)
    zctx.Term()
    zctx = nil
    cmn.LogInfo("terminated context.")
}


/*
 * create socket; connect/bind; Add to active socket list used by terminate context.
 */
func getSocket(mode channelMode, chType ChannelType, requester string) (sock *zmq.Socket, err error) {
    if shutdownYet {
        return nil, cmn.LogError("System is shutting down. No new socket")
    }
    defer func() {
        if err != nil {
            close(sock)
            sock = nil
        }
    }()

    address = ""
    sType = 0
    if address, sType, err = getAddress(mode, chType); err != nil {
        return
    }

    if sock, err = zctx.NewSocket(sType); err != nil { 
        err = cmn.LogError("Failed to get socket mode(%v) err(%v)", mode, err)
        return
    }
    /*
     * All pub & sub connect to xsub/xpub end points. 
     * Request connect & response binds
     * control pub channel connect and sub binds
     */
    switch mode {
    case CHANNEL_PROXY_CTRL_SUB, CHANNEL_RESPONSE:
        err = sock.Bind(address)
    default:
        err = sock.Connect(address)
    }

    if err != nil {
        err = cmn.LogError("Failed to bind/connect mode(%d) address(%s) err(%v)", mode, address, err)
    } else if err = sock.SetLinger(time.Duration(100) * time.Millisecond); err != nil {
        /* Context termination will sleep this long, for any message drain */
        err = cmn.LogError("Failed to call set linger chType(%d) address(%s) err(%v)",
                chType, address, err)
    } else {
        socketsList[sock] = fmt.Sprintf("mode(%d)_chType(%d)_(%s)", mode, chType, requester)
    }
    return
}

func closeSocket(s *zmqSocket) {
    if s != nil {
        close(s)
        delete(socketsList, s)
        /* In case terminate context is waiting */
        chSocksClose <- 1
    }
}

func managePublish(chType ChannelType, topic string, chReq <-chan JsonString_t,
        chRet chan<- error) {
    
    requester := fmt.Sprintf("publisher_topic(%s)_type(%d)", topic, chType)
    sock, err := getSocket(CHANNEL_PUBLISHER, chType, requester)

    defer closeSocket(sock)

    /* Inform the caller that function has initialized successfully or not */
    chRet <- err
    /* Sender close the channel */
    close(chRet)

    if err != nil {
        return
    }
    

    /* From here on the routine runs forever until shutdown */

    chShutdown := RegisterForSysShutdown("ZMQ-Publisher")

    for {
        select {
        case <- chShutdown:
            cmn.LogInfo("Shutting down publisher")
            return

        case data := <-chReq:
            if _, err = sock.SendMessage([topic, data]); err != nil {
                /* Don't return; Just log error */
                cmn.LogError("Failed to publish err(%v) requester(%s) data(%s)", 
                        err, requester, data)
            }
        }
    }
}


const QUITTOPIC = "quit"

func manageSubscribe(chType ChannelType, topic string, chRes chan<- JsonString_t,
        chRet chan<- error) {
    
    requester := fmt.Sprintf("subscriber_topic(%s)_type(%d)", topic, chType)
    var sock *zmq.Socket
    var err error
    defer closeSocket(sock)

    if err = getSocket(CHANNEL_SUBSCRIBER, chType, requester); err != nil {
        /* err is good enough */
    } else if err = sock.SetSubscribe(topic); err != nil {
        err = cmn.LogError("Failed to subscribe filter(%s) err(%v)", topic, err)
    } else if topic != "" {
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
    chShutErr = make(chan error) /* Track init error in following go func */

    go func() {
        /*
         * Pre-create qa publisher channel to alert subscribing channel
         * on shutdown.
         * You can't create a socket upon shutdown process start. So get it ahead.
         */
        shutSock, err := getSocket(CHANNEL_PUBLISHER, chType,
                    fmt.Sprintf("To_Shut_%s", requester))
        chShutErr <- err
        close(chShutErr)
        if err != nil {
            /* Terminate this routine */
            return
        }

        defer closeSocket(shutSock)

        chShutdown := RegisterForSysShutdown("ZMQ-Subscriber")

        /* Wait for shutdown signal */
        <- chShutdown
        shutDownRequested = true

        /* Send a message to wake up subscriber */
        if _, err = shutSock.SendMessage([QUITTOPIC, ""]); err != nil {
            cmn.LogError("Failed to send quit to %s", requester)
        }
    } ()

    /* read any error or nil from the above go func's init */
    err <- chShutErr
    chRet <- err    /* Forward the final init status to caller of this routine */
    close(chRet) /* Sender close the channel */

    /* From here on the routine runs forever until shutdown */
    for {
        if data, err = sock.RecvMessage(0); err != nil {
            cmn.LogError("Failed to receive msg err(%v for (%s)", requester)
        } else if len(data) != 2 {
            cmn.LogError("Expect 2 parts. requester(%s) data(%v)", requester, data)
        } else if shutDownRequested {
            cmn.LogInfo("Subscriber shutting down requester:(%s)", requester)
            /* Writer close the channel */
            close(chRes)
            break
        } else {
            chRes <- data[1]
        }
    }
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

trackList = map[string]bool {}
func openChannel(mode channelMode_t, chType ChannelType_t, topic string,
        chData chan JsonString_t) (err error) {

    id := fmt.Sprintf("%d_%d_%s", mode, chType, topic)
    if _, ok := trackList[id]; ok {
        err = cmn.LogError("Duplicate req mode=%d chType=%d topic=%s pre-exists",
                mode, chType, topic)
        return
    }

    switch mode {
    case CHANNEL_PUBLISHER, CHANNEL_SUBSCRIBER:
    default:
        err = cmn.LogError("Expect mode (%d) as pub/sub only", mode)
        return
    }

    if _, err = getContext(); err != nil {
        return
    }
    chRet = make(Chan error)

    if mode == CHANNEL_PUBLISHER {
        go managePublish(chType, topic, chData, chRet)
    } else {
        go manageSubscribe(chType, topic, chData, chRet)
    }

    /* Wait till routines get their init done */
    err = <-chRet
    if err != nil {
        trackList[id] = true
    }
    return
}

func runPubSubProxyInt(chType ChannelType, chRet chan<- int) {
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
    }

    var sock_xsub *zmq.Socket
    var sock_xpub *zmq.Socket
    var sock_ctrl_sub *zmq.Socket
    var err error
    address := ""
    ctx := nil

    if ctx, err = getContext(); err != nil {
        return
    }

    /*
     * Note: We don't track xsub & xpub in socketsList as they are controlled
     * control socket, which is tracked.
     */
    if sock_xsub, err = ctx.NewSocket(zmq.XMQ_XSUB); err != nil {
        err = cmn.LogError("Failed to get zmq xsub socket (%v)", err)
    } else if sock_xpub, err = ctx.NewSocket(zmq.XMQ_XPUB); err != nil {
        err = cmn.LogError("Failed to get zmq xpub socket (%v)", err)
    } else if address, _, err = getAddress(CHANNEL_PUBLISHER, chType); err != nil {
        /* err is well described */
    } else if err = sock_xsub.Bind(address); err != nil {
        err = cmn.LogError("Failed to bind xsub socket (%v)", err)
    } else if address, _, err = getAddress(CHANNEL_SUBSCRIBER, chType); err != nil {
        /* err is well described */
    } else if err = sock_xpub.Bind(address); err != nil {
        err = cmn.LogError("Failed to bind xpub socket (%v)", err)
    } else if sock_ctrl_sub, err = getSocket(CHANNEL_PROXY_CTRL_SUB, CHANNEL_TYPE_NA,
                "ctrl-sub-for-proxy"); err != nil {
        err = cmn.LogError("Failed to setup proxy err(%v)", err)
    } 
    if err != nil {
        return
    }
    chRet <- err    /* Inform caller, init is complete successfully */
    close(chRet)
    chRet = nil

    go func() {
        /* Watch for shutdown */
        chShutdown := RegisterForSysShutdown("terminate context")
        <- chShutdown

        var sock_ctrl_pub *zmq.Socket
        defer closeSocket(sock_ctrl_pub)

        /* Terminate proxy */
        if sock_ctrl_pub, err = getSocket(CHANNEL_PROXY_CTRL_PUB, CHANNEL_TYPE_NA,
                "ctrl-pub-for-proxy"); err != nil {
            err = cmn.LogError("Failed to create proxy control publisher to terminate proxy(%v)", err)
        } else if _, err = sock_ctrl_pub.Send("TERMINATE", 0); err != nil {
            err = cmn.LogError("Failed to write proxy control publisher to terminate proxy(%v)", err)
        }
    } 

    if err = zmq.Proxy(sock_xsub, sock_xpub, nil, sock_ctrl_sub); err != nil {
        cmn.LogError("Failing to run zmq.Proxy err(%v)", err)
    }
    return nil
}


var prxyList = map[chType]bool

func runPubSubProxy(chType ChannelType) error {
    if _, ok := prxyList[chType]; ok {
        return cmn.LogError("Proxy pre-exist for chType(%d)", chType)
    }
    chRet := make(chan error)
    go runPubSubProxyInt(chType, chRet)
    err = <- chRet
    if err == nil {
        prxyList[chType] = true
    }
    return err
}


type reqInfo_t struct {
    reqData     ClientReq_t
    chResData   chan<- *ClientRes_t
}


func clientRequestHandler(reqType ChannelType_t, chReq chan<- *reqInfo_t,
        chRet chan<- error) {
    
    requester := fmt.Sprintf("clientRequestHandler_type(%d)", reqType)
    sock, err := getSocket(CHANNEL_REQUEST, reqType, requester)

    defer closeSocket(sock)

    /* Inform the caller that function has initialized successfully or not */
    chRet <- err
    /* Sender close the channel */
    close(chRet)

    if err != nil {
        return
    }
    

    /* From here on the routine runs forever until shutdown */

    chShutdown := RegisterForSysShutdown("ZMQ-Publisher")

    for {
        select {
        case <- chShutdown:
            cmn.LogInfo("Shutting down %s", requester)
            return

        case data := <-chReq:
            res = ClientRes_t{}
                
            if _, res.err = sock.Send(data.reqData); res.err != nil {
                /* Don't return; Just log error */
                cmn.LogError("Failed to send request err(%v) requester(%s) data(%s)", 
                        res.err, requester, data)
            } else if res.res, res.err = sock.Recv(0); res.err != nil {
                cmn.LogError("Failed to recv response err(%v) requester(%s)", res.err, requester)
            }
            data.chResData <- &res
            close(data.chResData) /* No more writes as it is per request */
        }
    }
}


clientReqHandlerList = map[ChannelType_t]chan<- *reqInfo_t

func getRequestHandler(reqType ChannelType_t) (ch chan<- *reqInfo_t, err error) {
    ok = false
    if ch, ok = clientReqHandlerList[reqType]; !ok {
        ch = make(chan *reqInfo_t)
        chRet = make(chan error)
        go clientRequestHandler(reqType, ch, chRet)
        err = <- chRet
        if err == nil {
            clientReqHandlerList[reqType] = ch
        }
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
    var sock *zmq.Socket
    
    if ch, err = getRequestHandler(reqType); err == nil {
        ch <- &reqInfo_t{req, chRes)
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
var reqHandlers = map[chType]bool{}

func ServerRequestHandler(reqType ChannelType_t, chReq chan<- ClientReq_t,
            chRes <-chan *ServerRes_t, chRet chan<- error) {
    
    requester := fmt.Sprintf("server_request_handler(%s)_type(%d)", topic, reqType)
    var sock *zmq.Socket
    var err error
    defer closeSocket(sock)

    if err = getSocket(CHANNEL_RESPONSE, reqType, requester); err != nil {
        /* Inform the caller the failure to init and terminate this routine.*/
        chRet <- err
        close(chRet)
        return
    }

    shutDownRequested := false
    chShutErr = make(chan error) /* Track init error in following go func */

    go func() {
        /*
         * Pre-create qa publisher channel to alert subscribing channel
         * on shutdown.
         * You can't create a socket upon shutdown process start. So get it ahead.
         */
        shutSock, err := getSocket(CHANNEL_REQUEST, reqType,
                    fmt.Sprintf("To_Shut_%s", requester))
        chShutErr <- err
        close(chShutErr)
        if err != nil {
            /* Terminate this routine */
            return
        }

        defer closeSocket(shutSock)

        chShutdown := RegisterForSysShutdown("ZMQ-Subscriber")

        /* Wait for shutdown signal */
        <- chShutdown
        shutDownRequested = true

        /* Send a message to wake up subscriber */
        if _, err = shutSock.Send(QUITTOPIC); err != nil {
            cmn.LogError("Failed to send quit to %s", requester)
        }
    } ()

    /* read any error or nil from the above go func's init */
    err <- chShutErr
    chRet <- err    /* Forward the final init status to caller of this routine */
    close(chRet) /* Sender close the channel */

    /* From here on the routine runs forever until shutdown */
    for {
        if data, err = sock.Recv(0); err != nil {
            cmn.LogError("Failed to receive msg err(%v for (%s)", requester)
        } else if shutDownRequested {
            cmn.LogInfo("Subscriber shutting down requester:(%s)", requester)
            /* Writer close the channel */
            close(chRes)
            break
        } else {
            chRes <- data
        }
    }
} 


serverReqHandlerList = map[ChannelType_t]bool

func getRequestHandler(reqType ChannelType_t, chReq chan<- ClientReq_t,
        chRes <-chan *ServerRes_t) (err error) {
    ok = false
    if _, ok = serverReqHandlerList[reqType]; !ok {
        chRet = make(chan error)
        go clientRequestHandler(reqType, chReq, chRes, chRet)
        err = <- chRet
        if err == nil {
            serverReqHandlerList[reqType] = true
        }
    }
    return
}


