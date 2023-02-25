package lomipc

import (
    . "lomcommon"
    "net"
    "net/http"
    "net/rpc"
    "time"
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

type ReqDataType int

const (
    TypeNone = iota
    TypeRegClient
    TypeDeregClient
    TypeRegAction
    TypeDeregAction
    TypeActionRequest
    TypeActionResponse
    TypeActionHeartbeat
    TypeShutdown
    TypeReadServerRequest
    TypeCount
)

var ReqTypeToStr = map[ReqDataType]string {
    TypeNone: "None",
    TypeRegClient: "RegisterClient",
    TypeDeregClient: "DeregisterClient",
    TypeRegAction: "RegisterAction",
    TypeDeregAction: "DeregisterAction",
    TypeActionRequest: "ActionRequest",
    TypeActionResponse: "ActionResponse",
    TypeActionHeartbeat: "ActionHeartbeat",
    TypeShutdown: "Shutdown",
    TypeReadServerRequest: "ReadServerRequest",
}

type LoMRequest struct {
    ReqType     ReqDataType
    Client      string
    TimeoutSecs int
    ReqData     interface{}
}

type LoMRequestRPC struct {
    Req         *LoMRequest
    /* LoMResponse to this request is sent via this chan */
    ChResponse  chan interface{}
} 

/*
 * LoMResponse is tied to request closely as sent back via chan
 * embedded in the request. Hence does not need any more additional
 * data 
 *
 * RespData is specific to request. It could be nil.
 */
type LoMResponse struct {
    ResultCode  int
    ResultStr   string

    RespData    interface{}
}

/* All kinds of request data matching request type */
type MsgRegClient struct {
}

type MsgDeregClient struct {
}

type MsgRegAction struct {
    Action  string
}

type MsgDeregAction struct {
    Action  string
}

type MsgActionRequest struct {
    Action              string
    InstanceId          string
    AnomalyInstanceId   string
    AnomalyKey          string
    Context             string
}

type MsgActionResponse struct {
    Action              string
    InstanceId          string
    AnomalyInstanceId   string
    AnomalyKey          string
    Response            string
    ResultCode          int
    ResultStr           string
}

type MsgRegHeartbeat struct {
    Action      string
    Timestamp   int
}

type MsgShutdown struct {
}

type MsgReadServerReq struct {
}

const DefeultTimeoutSeconds = 2    /* Default timeout for any pending call */

/*
 * Each proc has a channel for remote end to write request.
 * Each request carry a channel for response to that request.
 */
type LoMTransport struct {
    ServerCh    chan interface{}
}

/* RPC call from client */
func (tr *LoMTransport) SendToServer(req *LoMRequest, reply *LoMResponse) (err error) {

    defer func() {
        if err != nil {
            LogError("SendToServer cl(%s) mtype(%s) failed (%v)", 
                    req.Client, ReqTypeToStr[req.ReqType], err)
        } else {
            LogInfo("SendToServer cl(%s) mtype(%s) result(%d)/(%s)", req.Client,
                     ReqTypeToStr[req.ReqType], reply.ResultCode, reply.ResultStr)
        }
    } ()

    rpcReq := LoMRequestRPC { req, make(chan interface{}) }
    tr.ServerCh <- rpcReq

    LogDebug("Req sent to server client(%s) type(%s). Waiting for response...",
            req.Client, ReqTypeToStr[req.ReqType])

    /* Wait for server response */
    p := <- rpcReq.ChResponse
    if x, ok := p.(*LoMResponse); ok {
        *reply = *x
    } else {
        return GetError("Server response message (%T) != LoMResponse", x)
    }

    return nil
}

/* Local call from server to read client request. */
func (tr *LoMTransport) ReadClientRequest(timeout int) *LoMRequestRPC {
    select {
    case p := <-tr.ServerCh:
        if x, ok := p.(*LoMRequestRPC); ok {
            LogDebug("Server: Read from client (%s) type(%s)", x.Req.Client, ReqTypeToStr[x.Req.ReqType])
            return x;
            /* Let server return response upon processing, via channel embedded in msg. */
        } else {
            LogError("Client request message (%T) != *Msg", x)
            return nil
        }
    case <- time.After(time.Duration(timeout) * time.Second):
        LogError("Server: Aborting read from client timeout=%d", timeout)
        /* Aborting per instruction */
        return nil
    }
}


func ServerInit() (*LoMTransport, error) {
    tr := new(LoMTransport)
    
    tr.ServerCh = make(chan interface{})

    rpc.Register(tr)
    rpc.HandleHTTP()
    l, e := net.Listen("tcp", ":1234")
    if e != nil {
        LogPanic("listen error:(%v)", e)
        return nil, e
    }
    go http.Serve(l, nil)
    LogDebug("Server: Started serving")
    return tr, nil
}



