package lomipc


import(
    . "lomcommon"
    "net/rpc"
)

const server_address = "localhost"

var RPCDialHttp = rpc.DialHTTP

type ClientTx struct {
    ClientRpc   *rpc.Client
    ClientName  string
    TimeoutSecs int
}

func txCallClient(tx *ClientTx, serviceMethod string, args any, reply any) error {
    if tx.ClientRpc == nil {
        return LogError("txCallClient: No Transport; Need to register first")
    }
    return tx.ClientRpc.Call(serviceMethod, args, reply)
}

var ClientCall = txCallClient

func (tx *ClientTx) RegisterClient(client string) error {
    r, err := RPCDialHttp("tcp", server_address+":1234")

    if (err != nil) {
        LogError("RegisterClient: Failed to call rpc.DialHTTP err:(%v)", err)
        return err
    }

    defer func() {
        if err != nil {
            tx.ClientRpc = nil
            tx.ClientName = ""
        }
    }()
    
    tx.ClientRpc = r
    tx.ClientName = client
    req := &LoMRequest { TypeRegClient, client, tx.TimeoutSecs, MsgRegClient{} }
    reply := &LoMResponse{}
    err = ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("RegisterClient: Failed to call SendToServer (%s) (%v)", client, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("RegisterClient: Server failed client (%v) result(%d/%s)", client,
                reply.ResultCode, reply.ResultStr)
    }

    res := reply.RespData
    if x, ok := res.(MsgEmptyResp); !ok {
        return LogError("RegisterClient: Expect empty resp. (%T) (%v)", x, x)
    }

    LogInfo("Registered client (%s)", client)
    return nil
}


func (tx *ClientTx) DeregisterClient() error {
    defer func() {
        tx.ClientRpc = nil
        tx.ClientName = ""
    }()

    req := &LoMRequest { TypeDeregClient, tx.ClientName, tx.TimeoutSecs, MsgDeregClient{} }
    reply := &LoMResponse{}
    err := ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("DeregisterClient: Failed to call SendToServer (%s) (%v)", tx.ClientName, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("DeregisterClient: Server failed (%v) result(%d/%s)", tx.ClientName,
                reply.ResultCode, reply.ResultStr)
    }

    res := reply.RespData
    if x, ok := res.(MsgEmptyResp); !ok {
        return LogError("DeegisterClient: Expect empty resp. (%T) (%v)", x, x)
    }

    LogInfo("Deregistered client (%s)", tx.ClientName)
    return nil
}


func (tx *ClientTx) RegisterAction(action string) error {

    req := &LoMRequest { TypeRegAction, tx.ClientName, tx.TimeoutSecs, MsgRegAction{action} }
    reply := &LoMResponse{}
    err := ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("RegisterAction: Failed to call SendToServer (%s/%s) (%v)", tx.ClientName,
                action, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("RegisterAction: Server failed (%s/%s) result(%d/%s)", tx.ClientName,
                action, reply.ResultCode, reply.ResultStr)
    }

    res := reply.RespData
    if x, ok := res.(MsgEmptyResp); !ok {
        return LogError("RegisterAction: Expect empty resp. (%T) (%v)", x, x)
    }

    LogInfo("Registered action (%s/%s)", tx.ClientName, action)
    return nil
}


func (tx *ClientTx) DeregisterAction(action string) error {

    req := &LoMRequest { TypeDeregAction, tx.ClientName, tx.TimeoutSecs, MsgDeregAction{action} }
    reply := &LoMResponse{}
    err := ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("DeregisterAction: Failed to call SendToServer (%s/%s) (%v)", tx.ClientName,
                action, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("DeregisterAction: Server failed (%s/%s) result(%d/%s)", tx.ClientName,
                action, reply.ResultCode, reply.ResultStr)
    }

    res := reply.RespData
    if x, ok := res.(MsgEmptyResp); !ok {
        return LogError("DeegisterAction: Expect empty resp. (%T) (%v)", x, x)
    }

    LogInfo("Deregistered action (%s/%s)", tx.ClientName, action)
    return nil
}

func (tx *ClientTx) RecvServerRequest() (*ServerRequestData, error) {

    req := &LoMRequest { TypeRecvServerRequest, tx.ClientName, tx.TimeoutSecs, MsgRecvServerRequest{} }
    reply := &LoMResponse{}
    err := ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("RecvServerRequest: Failed to call SendToServer (%s) (%v)", tx.ClientName, err)
        return nil, err
    }
    if (reply.ResultCode != 0) {
        return nil, LogError("RecvServerRequest: Server failed (%s) result(%d/%s)", tx.ClientName,
                reply.ResultCode, reply.ResultStr)
    }

    p := reply.RespData
    res, ok := p.(ServerRequestData)
    if !ok {
        return nil, LogError("RecvServerRequest: RespData (%T) != *ActionRequestData", res)
    }

    LogInfo("RecvServerRequest: succeeded (%s/%s)", tx.ClientName,
                    ServerReqTypeToStr[res.ReqType])
    return &res, nil
}

func (tx *ClientTx) SendServerResponse(res *ServerResponseData) error {
    req := &LoMRequest { TypeSendServerResponse, tx.ClientName, tx.TimeoutSecs, res }
    reply := &LoMResponse{}
    err := ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("SendServerResponse: Failed to call SendToServer (%s) (%v)", tx.ClientName, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("SendServerResponse: Server failed (%s) result(%d/%s)", tx.ClientName,
                reply.ResultCode, reply.ResultStr)
    }

    resD := reply.RespData
    if x, ok := resD.(MsgEmptyResp); !ok {
        return LogError("SendServerResponse: Expect empty resp. (%T) (%v)", x, x)
    }

    LogInfo("SendServerResponse: succeeded (%s/%s)", tx.ClientName, ServerReqTypeToStr[res.ReqType])
    return nil

}


func (tx *ClientTx) NotifyHeartbeat(action string, tstamp EpochSecs) error {
    req := &LoMRequest { TypeNotifyActionHeartbeat, tx.ClientName, tx.TimeoutSecs, 
                MsgNotifyHeartbeat { action, tstamp }}
    reply := &LoMResponse{}
    err := ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("NotifyHeartbeat: Failed to call SendToServer (%s/%s) (%v)", tx.ClientName,
                action, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("NotifyHeartbeat: Server failed (%s/%s) result(%d/%s)", tx.ClientName,
                action, reply.ResultCode, reply.ResultStr)
    }

    res := reply.RespData
    if x, ok := res.(MsgEmptyResp); !ok {
        return LogError("NotifyHeartbeat: Expect empty resp. (%T) (%v)", x, x)
    }

    LogInfo("Notified heartbeat from action (%s/%s)", tx.ClientName, action)
    return nil
}


