package lomipc

import (
    "encoding/gob"
    . "lib/lomcommon"
    "net"
    "net/http"
    "net/rpc"
    "reflect"
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
type ServerReqDataType int

const (
    TypeNone = iota
    TypeRegClient
    TypeDeregClient
    TypeRegAction
    TypeDeregAction
    TypeRecvServerRequest
    TypeSendServerResponse
    TypeNotifyActionHeartbeat
    TypeCount
)

const (
    TypeServerRequestAction = iota
    TypeServerRequestShutdown
    TypeServerRequestCount
)

var ReqTypeToStr = map[ReqDataType]string {
    TypeNone: "None",
    TypeRegClient: "RegisterClient",
    TypeDeregClient: "DeregisterClient",
    TypeRegAction: "RegisterAction",
    TypeDeregAction: "DeregisterAction",
    TypeRecvServerRequest: "RecvServerRequest",
    TypeSendServerResponse: "SendServerResponse",
    TypeNotifyActionHeartbeat: "NotifyActionHeartbeat",
}

var ServerReqTypeToStr = map[ServerReqDataType]string {
    TypeServerRequestAction: "RecvServerRequestAction",
    TypeServerRequestShutdown: "RecvServerRequestShutdown",
}

/* Request from client to server over RPC */
type LoMRequest struct {
    ReqType     ReqDataType
    Client      string
    TimeoutSecs int
    ReqData     interface{}
}

/* Internal req object that is sent over chan */
type LoMRequestInt struct {
    Req         *LoMRequest
    /* LoMResponse to this request is sent via this chan */
    ChResponse  chan interface{}
} 

/*
 * Response from server to client.
 *
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

type MsgRecvServerRequest struct {
}

type ServerRequestData struct {
    ReqType             ServerReqDataType
    ReqData             interface {}
}

type ServerResponseData struct {
    ReqType             ServerReqDataType
    ResData             interface {}
}

type ActionResponseData struct {
    Action              string
    InstanceId          string
    AnomalyInstanceId   string
    AnomalyKey          string
    Response            string
    ResultCode          int
    ResultStr           string
}

type ActionRequestBaseData struct {
    Action              string
    InstanceId          string
    AnomalyInstanceId   string
    AnomalyKey          string
}

/* Data sent as response via RespData for MsgRecvServerRequest */
type ActionRequestData struct {
    ActionRequestBaseData
    Context             []ActionResponseData
}

type ShutdownRequestData struct {
}

type EpochSecs int64

type MsgNotifyHeartbeat struct {
    Action      string
    Timestamp   EpochSecs
}

type MsgShutdownData struct {
}

type MsgEmptyResp struct {
}

func SlicesComp(p []ActionResponseData, q []ActionResponseData) bool {
    if (len(p) != len(q)) {
        LogDebug("Slice len differ %d != %d\n", len(p), len(q))
        return false
    }

    for i, v := range(p) {
        if (v != q[i]) {
            LogDebug("p[%d] (%v) != q[%d] (%v)\n", i, v, i, q[i])
            return false
        }
    }
    return true
}

func (r *ActionRequestData) Equal(p *ActionRequestData) bool {
    if ((r.Action == p.Action) &&
        (r.InstanceId == p.InstanceId) &&
        (r.AnomalyInstanceId == p.AnomalyInstanceId) &&
        (r.AnomalyKey == p.AnomalyKey) &&
        SlicesComp(r.Context, p.Context)) {
        return true
    } else {
        return false
    }
}


func (r *ServerRequestData) Equal(p *ServerRequestData) bool {
    if r.ReqType != p.ReqType {
        LogDebug("Differing Req types %s vs %s", ServerReqTypeToStr[r.ReqType], 
                ServerReqTypeToStr[p.ReqType])
        return false
    }
    rr := r.ReqData
    pr := p.ReqData
    if reflect.TypeOf(rr) != reflect.TypeOf(pr) {
        LogDebug("Differing ReqData types %T vs %T", rr, pr)
        return false
    }
    switch rr.(type) {
    case ActionRequestData:
        rq, ok1 := rr.(ActionRequestData) 
        pq, ok2 := pr.(ActionRequestData) 
        if (!ok1 || !ok2 || !(&rq).Equal(&pq)) {
            LogDebug("ActionRequestData mismatch ok1=%v ok2=%v", ok1, ok2)
            return false
        }
        return true
    case ShutdownRequestData:
        return true
    default:
        LogError("Unkown ReqData type (%T) in ServerRequestData", rr)
        return false
    }
}


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
            LogInfo("SUCCESS: SendToServer cl(%s) mtype(%s) result(%d)/(%s)", req.Client,
                     ReqTypeToStr[req.ReqType], reply.ResultCode, reply.ResultStr)
        }
    } ()

    rpcReq := LoMRequestInt { req, make(chan interface{}) }
    tr.ServerCh <- &rpcReq

    LogDebug("Req sent to server client(%s) type(%s). Waiting for response...",
            req.Client, ReqTypeToStr[req.ReqType])

    /* Wait for server response */
    p := <- rpcReq.ChResponse
    if x, ok := p.(*LoMResponse); ok {
        *reply = *x
    } else {
        return LogError("Server response message (%T) != *LoMResponse", x)
    }

    return nil
}

/* Local call from server to read client request. */
func (tr *LoMTransport) ReadClientRequest(chAbort chan interface{}) (*LoMRequestInt, error) {
    /* Return on non-null request or upon abort */
    select {
    case p := <-tr.ServerCh:
        if x, ok := p.(*LoMRequestInt); ok {
            LogDebug("Server: Read from client (%s) type(%s)", x.Req.Client, ReqTypeToStr[x.Req.ReqType])
            return x, nil
            /* Let server return response upon processing, via channel embedded in msg. */
        } else {
            return nil, LogError("Client request message (%T) != *Msg", x)
        }
        
    case <- chAbort:
        return nil, LogError("Server: Aborting read via abort channel")
        /* Aborting per instruction */
    }
}


func init_encoding() {
    gob.Register(MsgRegClient{})
    gob.Register(MsgDeregClient{})
    gob.Register(MsgRegAction{})
    gob.Register(MsgDeregAction{})
    gob.Register(MsgRecvServerRequest{})
    gob.Register(ServerRequestData{})
    gob.Register(ServerResponseData{})
    gob.Register(ActionRequestBaseData{})
    gob.Register(ActionRequestData{})
    gob.Register(ActionResponseData{})
    gob.Register(ShutdownRequestData{})
    gob.Register(MsgNotifyHeartbeat{})
    gob.Register(MsgEmptyResp{})
}

func ServerInit() (*LoMTransport, error) {
    init_encoding()
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



