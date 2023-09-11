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
const ZMQ_ADDRESS = "tcp://localhost:%d"


func getAddress(mode channelMode, chType ChannelType) (path string, sType zmq.Type, err error) {
    port := 0
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
    case CHANNEL_REQUEST, CHANNEL_RESPONSE:
        port = ZMQ_REQ_REP_PORT
        sType = zmq.ZMQ_REP
    default:
        err = cmn.LogError("Unknown channel mode (%v)", mode)
    }
    path = fmt.Sprintf(ZMQ_ADDRESS, port)
    return
}

var zctx = nil

func getContext() (*zmq.Context, error) {
    if zctx != nil {
        if c, err := zmq.NewContext(); err != nil {
            return nil, cmn.LogError("Failed to get zmq context (%v)", err)
        }
        }   
    }   
    return zctx, nil 
}   


type zmqHandle struct {
    sockType    zmq.Type
    sock        *zmq.Socket
    topic       string
}


func getHandle(mode channelMode, chType ChannelType, prefix string) (handle any, err error) {
    defer func() {
        if (err != nil) && (sock != nil) {
            sock.Close()
        }
    }()

    sock := nil
    sockType := 0
    address := ""
    ctx := nil

    if address, sockType, err = getAddress(mode, chType); err != nil {
        return
    }

    if ctx, err = getContext(); err != nil {
        return
    }

    if sock, err = ctx.NewSocket(socktype); err != nil {
        err = cmn.LogError("Failed to get zmq socket (%v)", err)
        return
    }

    /* Writer connects, reader binds - Just an internal norm */
    switch sockType {
    case zmq.ZMQ_PUB:
        if prefix == "" {
            err = cmn.LogError("Publisher need non empty prefix")
        } else if err = sock.Connect(address); err != nil {
            err = cmn.LogError("Failed to connect to (%v) err (%v)", address, err)
        }
    case zmq.ZMQ_SUB:
        if err = sock.Connect(address); err != nil {
            err = cmn.LogError("Failed to Connect to (%v) err (%v)", address, err)
        } else if err = sock.SetSubscribe(prefix); err != nil {
            err = cmn.LogError("Failed to subscribe filter(%s) err(%v)", filter, err)
        }
    case zmq.ZMQ_REQ:
        if err = sock.Connect(address); err != nil {
            err = cmn.LogError("Failed to Connect to (%v) err (%v)", address, err)
        }
    case zmq.ZMQ_REP:
        if err = sock.Bind(address); err != nil {
            err = cmn.LogError("Failed to Bind to (%v) err (%v)", address, err)
        }
    default:
        err = cmn.LogError("Unsupported type (%v)", sockType)
    }
    if err == nil {
        handle = &zmqHandle { sockType, sock, prefix }
    }
    return
}


func sendPart(d string, f zmq.Flag) error {
    if l, err := sock.Send(d, f); err != nil {
        return cmn.LogError("Failed to send part (%v) err(%v)", d, err)
    } else if len(d) != l {
        return cmn.LogError("Failed to send part (%v) len(%d) != sent(%d)", d, len(d), l)
    }
    return nil
}

func getHandle((handle any) (*zmqHandle, err) {
    h, ok := handle.(*zmqHandle)
    if !ok {
        return nil, cmn.LogError("Invalid handle type %T", handle)
    }
    return h, nil
}

func WriteHandle(handle any, data string) error {
    h, err := getHandle(handle)
    if err != nil {
        return err
    }
    dataSlice := []string{}
    if h.topic != "" {
        dataSlice = []string { h.topic, data }
    } else {
        dataSlice = []string { data }
    }
    return h.sock.SendMessage(dataSlice)
}


func ReadHandle(handle any) ([]string, error) {
    h, err := getHandle(handle)
    if err != nil {
        return nil, err
    }
    if data, err := h.sock.RecvMessage(0); err != nil {
        return nil, err
    }
    /*
     * Only subscribe expects 2 parts with first part holding
     * address. Hence return the last part
     */
    return data[len(data)-1], nil
}


func CloseHandlehandle any) {
    h, err := getHandle(handle)
    if err == nil {
        h.sock.Close()
        /* Don't nullify it. Nil will crash the code if called by
         * mistake, where as closed handle will fail gracefully 
         * with appropriate error 
         */
    }
}


func runPubSubProxy(chType ChannelType, chAbort <-chan int) (err error) {
    defer func() {
        if sock_xsub != nil {
            sock_xsub.Close()
        }
        if sock_xpub != nil {
            sock_xpub.Close()
        }
    }

    var sock_xsub any
    var sock_xpub any
    var err error
    address := ""
    ctx := nil

    if ctx, err = getContext(); err != nil {
        return
    }

    if sock_xsub, err = ctx.NewSocket(zmq.XMQ_XSUB); err != nil {
        err = cmn.LogError("Failed to get zmq xsub socket (%v)", err)
        return
    }

    if sock_xpub, err = ctx.NewSocket(zmq.XMQ_XPUB); err != nil {
        err = cmn.LogError("Failed to get zmq xpub socket (%v)", err)
        return
    }

    /* Bind xSUB to PUB sockets */
    if address, _, err := getAddress(CHANNEL_PUBLISHER, chType); err != nil {
        return
    }
    if err = sock_xsub.Bind(address); err != nil {
        err = cmn.LogError("Failed to bind xsub socket (%v)", err)
        return
    }

    /* Bind xPUB to SUB sockets */
    if address, _, err := getAddress(CHANNEL_SUBSCRIBER, chType); err != nil {
        return
    }
    if err = sock_xpub.Bind(address); err != nil {
        err = cmn.LogError("Failed to bind xpub socket (%v)", err)
        return
    }

    if err = zmq.Proxy(sock_xsub, sock_xpub, nil); err != nil {
        err = cmn.LogError("Failing to run zmq.Proxy err(%v)", err)
        return
    }
    return nil
}
