package lomipc

import (
    "errors"
    "fmt"
    "log"
    "net"
    "net/http"
    "net/rpc"
)

/*
 *  Transport i/f via Go RPC https://pkg.go.dev/net/rpc
 *
 *  NOTE:
 *      This is used as only IPC between processes running inside a single
 *      container as single system; Tightly coupled with static set of APIs
 *      
 *  Multiple clients (PluginMgr) send requests concurrently to a server/engine.
 *  Server process each sequentially/concurrently and respond back to each as succeeded
 *  or failed.
 *  
 *  Server send requests to multiple clients sequentially/concurrently as a request
 *  addressed to a client only. A single client may receive multiple requests at any time.
 *  For each received request, client confirm back as succeeded / failed, synchronously.
 *  The client may just do some basic validation on the request.
 *
 *  Server creates transport with a channel for all clients to send their request.
 *  Similarly each client creates a channel for all server requests.
 *
 *  The request carry a channel for response.
 *
 *  The channel creator/owner holds the read end and the remote process holds the write end.
 *
 *  Channels are buffered with estimated count of clients for server and known count of
 *  plugins managed by client.
 */
type MsgType int

const (
    TypeNone = iota
    TypeRegClient
    TypeRegAction
    TypeDeregClient
    TypeDeregAction
    TypeActionInput
    TypeActionOutput
    TypeActionHeartbeat
    TypeReceiverResponse
    TypeShutdown
    TypeCount
)

var MsgTypeToStr = [TypeCount]string {
    "None",
    "RegisterClient",
    "RegisterAction",
    "DeregisterClient",
    "DeregisterAction",
    "ActionInput",
    "ActionOutput",
    "ActionHeartbeat",
    "ReceiverResponse",
    "Shutdown" }

type Msg struct {
    Type    MsgType
    Client  string
    Action  string

    chResponse  chan interface{}
}

type MsgActionInput struct {
    Msg
    InstanceId          string
    AnomalyInstanceId   string
    AnomalyKey          string
    Context             string
    timeout             int
}

type MsgActionOutput struct {
    Msg
    InstanceId          string
    AnomalyInstanceId   string
    AnomalyKey          string
    Response            string
    ResultCode          int
    ResultStr           string
}


/*
 * Each proc has a channel for remote end to write request.
 * Each request carry a channel for response to request.
 */
type LoMTransport struct {
    ServerCh    chan interface{}
    ClientsCh   map[string]chan interface{}
}

type Reply struct {
    ResultCode  int
    ResultStr   string
}


/* RPC call from client */
func (tr *LoMTransport) SendToServer(msg *Msg, reply *Reply) error {

    mtype := msg.Type
    clientName := msg.Client

    if mtype == TypeRegClient {
        delete(tr.ClientsCh, clientName)
        tr.ClientsCh[clientName] = make (chan interface{})
    } else if _, ok := tr.ClientsCh[clientName]; !ok {
        return errors.New("Client is not registered yet " + clientName + ": " + MsgTypeToStr(mtype))
    }
    msg.chResponse = make(chan Reply)
    tr.ServerCh <- msg

    /* Wait for server response */
    *reply = <- msg.chResponse

    if mtype == TypeDeregClient {
        delete(tr.ClientsCh, clientName)
    }

    return nil
}

/* RPC call from client */
func (tr *LoMTransport) readFromServer(msg *Msg, reply *Msg) error {
    if ch, ok := tr.ClientsCh[client]; ok {
        *reply = <-ch
        if msg.chResponse {
            /* If server need response, send upon client reading it */
            msg.chResponse <- "Done"
            msg.chResponse = nil
        }
        return nil
    } else {
        return errors.New("Client is not registered yet " + msg.Client)
    }
}

/* Local call from server to read client request. */
func (tr *LoMTransport) ReadFromClient(ch chan interface{}) *Msg {
    select {
    case p := <-tr.ServerCh:
        return p
        /* Let server return response upon processing, via channel embedded in msg. */
    case <- ch:
        /* Aborting per instruction */
        return nil
    }
}


/*
 * Request for action or shutdown to client.
 * Local call from server.
 * If server needs some response, it gets sent when client is reading it.
 */
func (tr *LoMTransport) SendToClient(msg *Msg) error {
    client := msg.Client

    if ch, ok := tr.ClientsCh[client]; ok {
        ch <- msg
    } else {
        return errors.New("Client is not registered yet " + client + " " + MsgTypeToStr(msg.Type))
    }
}

func ServerInit() (*LoMTransport, error) {
    tr := new(LoMTransport)
    
    tr.ServerCh = make(chan interface{})

    rpc.Register(tr)
    rpc.HandleHTTP()
    l, e := net.Listen("tcp", ":1234")
    if e != nil {
        log.Printf("listen error:(%v)", e)
        return nil, e
    }
    go http.Serve(l, nil)
    return tr, nil
}



