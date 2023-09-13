package lomtelemetry

import (
    cmn "lom/src/lib/lomcommon"

    "encoding/json"
    "fmt"

    zmq "github.com/pebbe/zmq4"
)


const ZMQ_REQ_REP_PORT = 5650
const ZMQ_XPUB_START_PORT = 5750
const ZMQ_XSUB_START_PORT = 5850
const ZMQ_PROXY_CTRL_PORT = 5950

const ZMQ_ADDRESS = "tcp://localhost:%d"


func getAddress(mode channelMode, chType ChannelType) (path string, sType zmq.Type, err error) {
    port := 0
    switch mode {
    case CHANNEL_PUBLISHER, CHANNEL_SUBSCRIBER:
        switch chType {
        case CHANNEL_TYPE_EVENTS, CHANNEL_TYPE_COUNTERS, CHANNEL_TYPE_REDBUTTON:
        default:
            err = cmn.LogError("Unknown channel type (%d) for mode(%d)", chType, mode)
            return

        }
    case CHANNEL_REQUEST, CHANNEL_RESPONSE:
    case CHANNEL_PROXY_CTRL_PUB, CHANNEL_PROXY_CTRL_SUB:
    default:
        err = cmn.LogError("Unknown channel mode (%v)", mode)
        return
    }

    switch mode {
    case CHANNEL_PUBLISHER:
        port = ZMQ_XSUB_START_PORT + int(chType)
        sType = zmq.ZMQ_PUB
    case CHANNEL_SUBSCRIBER:
        port = ZMQ_XPUB_START_PORT + int(chType)
        sType = zmq.ZMQ_SUB
    case CHANNEL_REQUEST:
        port = ZMQ_REQ_REP_PORT
        sType = zmq.ZMQ_REQ
    case CHANNEL_RESPONSE:
        port = ZMQ_REQ_REP_PORT
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
 */
var zctx = nil

/*
 * Collect all open sockets. Ctx termination is blocked by any
 * open socket.
 */
var socketsList = map[*zmq.Socket]string{}

/*
 * Each socket close writes into this channel
 * During shutdown term contex sleep on this channel until all
 * all sockets are closed, hence the sockets list is empty.
 */
chan chSocksClose = make(chan int)

func getContext() (*zmq.Context, error) {
    if zctx == nil {
        if zctx, err := zmq.NewContext(); err != nil {
            return nil, cmn.LogError("Failed to get zmq context (%v)", err)
        }
        go terminateContext()
    }   
    return zctx, nil 
}   

func terminateContext() {
    chShutdown := RegisterForSysShutdown("terminate context")
    <- chShutdown 

    /* System shutdown initiated; Wait for open sockets to close */
    for len(socketsList) != 0 {
        var pending []string
        for _, v := range socketsList {
            pending = append(pending, v)
        }
        cmn.LogError("Waiting for [%d] socks to close pending(%v)", len(pending), pending)
        <- chSocksClose
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
    defer func() {
        if err != nil {
            close(sock)
            sock = nil
        } else {
            socketsList[sock] = fmt.Sprintf("mode(%d)_chType(%d)_(%s)", mode, chType, requester)
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

func managePublish(chType ChannelType, topic string, chReq <-chan string,
        chRet chan<- error) {
    defer func() {
        if chRet != nil {
            cmn.LogError("Internal code error. Expect chRet ==  nil")
        }
    }()

    
    requester := fmt.Sprintf("publisher_topic(%s)_type(%d)", topic, chType)
    sock, err := getSocket(CHANNEL_PUBLISHER, chType, requester)

    defer closeSocket(sock)

    /* Inform the caller that function has initialized successfully or not */
    chRet <- err
    /* Sender close the channel */
    close(chRet)
    chRet = nil

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

func manageSubscribe(chType ChannelType, topic string, chRes chan<- string,
        chRet chan<- error) {
    
    defer func() {
        if chRet != nil {
            cmn.LogError("Internal code error. Expect chRet ==  nil")
        }
    }()

    requester := fmt.Sprintf("subscriber_topic(%s)_type(%d)", topic, chType)
    sock, err := getSocket(CHANNEL_SUBSCRIBER, chType, requester)

    defer closeSocket(sock)

    if err = sock.SetSubscribe(topic); err != nil {
        err = cmn.LogError("Failed to subscribe filter(%s) err(%v)", topic, err)
    }
    if topic != "" {
        err = sock.SetSubscribe(QUITTOPIC); err != nil {
            err = cmn.LogError("Failed to subscribe filter(%s) err(%v)", QUITTOPIC, err)
        }
    }

    /* Inform the caller that function has initialized successfully or not */
    chRet <- err
    /* Sender close the channel */
    close(chRet)
    chRet = nil
    
    if err != nil {
        return
    }
    

    /* From here on the routine runs forever until shutdown */

    chShutdown := RegisterForSysShutdown("ZMQ-Subscriber")
    shutDownRequested := false

    go func() {
        /* Wait for shutdown signal */
        <- chShutdown
        shutDownRequested = true

        /* Send a message to wake up subscriber */
        sock, err := getSocket(CHANNEL_PUBLISHER, chTyp, fmt.Sprintf("To_Shut_%s", requester))
        if err == nil {
            defer closeSocket(sock)
            _, err = sock.SendMessage([QUITTOPIC, ""]);
        }
        if err != nil {
            cmn.LogError("Failed to send quit to %s", requester)
        }
    } ()


    for {
        if data, err = sock.RecvMessage([topic, data]); err != nil {
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
func runubSubChannel(mode channelMode_t, chType ChannelType_t, topic string,
        chData chan string) (err error) {

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


func runPubSubProxy(chType ChannelType) error {
    chRet := make(chan err)
    go runPubSubProxyInt(chType, chRet)
    return <- chRet
}


