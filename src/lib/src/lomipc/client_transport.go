package lomipc


import(
    . "lomcommon"
    "net/rpc"
)

const server_address = "localhost"

type ClientTx struct {
    ClientRpc   *rpc.Client
    ClientName  string
    TimeoutSecs int
}


func (tx *ClientTx) RegisterClient(client string) error {
    r, err := rpc.DialHTTP("tcp", server_address+":1234")
    if (err != nil) {
        LogError("RegisterClient: Failed to call rpc.DialHTTP err:(%v)", err)
        return err
    }
    req := &LoMRequest { TypeRegClient, client, tx.TimeoutSecs, MsgRegClient{} }
    reply := &LoMResponse{}
    err = r.Call("LoMTransport.SendToServer", req, reply)
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

    tx.ClientRpc = r
    tx.ClientName = client
    LogInfo("Registered client (%s)", client)
    return nil
}


func (tx *ClientTx) DeregisterClient() error {
    defer func() {
        tx.ClientRpc = nil
        tx.ClientName = ""
    }()

    if tx.ClientRpc == nil {
        return LogError("DeregisterClient: No Transport; Need to register first")
    }

    req := &LoMRequest { TypeDeregClient, tx.ClientName, tx.TimeoutSecs, MsgDeregClient{} }
    reply := &LoMResponse{}
    err := tx.ClientRpc.Call("LoMTransport.SendToServer", req, reply)
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
    if tx.ClientRpc == nil {
        return LogError("RegisterAction: No Transport; Need to register first")
    }

    req := &LoMRequest { TypeRegAction, tx.ClientName, tx.TimeoutSecs, MsgRegAction{action} }
    reply := &LoMResponse{}
    err := tx.ClientRpc.Call("LoMTransport.SendToServer", req, reply)
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
    if tx.ClientRpc == nil {
        return LogError("DeregisterAction: No Transport; Need to register first")
    }

    req := &LoMRequest { TypeDeregAction, tx.ClientName, tx.TimeoutSecs, MsgDeregAction{action} }
    reply := &LoMResponse{}
    err := tx.ClientRpc.Call("LoMTransport.SendToServer", req, reply)
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

func (tx *ClientTx) RecvActionRequest() (*ActionRequestData, error) {
    if tx.ClientRpc == nil {
        return nil, LogError("RecvActionRequest: No Transport; Need to register first")
    }

    req := &LoMRequest { TypeRecvActionRequest, tx.ClientName, tx.TimeoutSecs, MsgRecvActionRequest{} }
    reply := &LoMResponse{}
    err := tx.ClientRpc.Call("LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("RecvActionRequest: Failed to call SendToServer (%s) (%v)", tx.ClientName, err)
        return nil, err
    }
    if (reply.ResultCode != 0) {
        return nil, LogError("RecvActionRequest: Server failed (%s) result(%d/%s)", tx.ClientName,
                reply.ResultCode, reply.ResultStr)
    }

    p := reply.RespData
    res, ok := p.(ActionRequestData)
    if !ok {
        return nil, LogError("RecvActionRequest: RespData (%T) != *ActionRequestData", res)
    }
    LogInfo("RecvActionRequest: succeeded (%s/%s)", tx.ClientName, res.Action)
    return &res, nil
}

func (tx *ClientTx) SendActionResponse(res *ActionResponseData) error {
    if tx.ClientRpc == nil {
        return LogError("SendActionResponse: No Transport; Need to register first")
    }

    req := &LoMRequest { TypeSendActionResponse, tx.ClientName, tx.TimeoutSecs, res }
    reply := &LoMResponse{}
    err := tx.ClientRpc.Call("LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("SendActionResponse: Failed to call SendToServer (%s) (%v)", tx.ClientName, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("SendActionResponse: Server failed (%s) result(%d/%s)", tx.ClientName,
                reply.ResultCode, reply.ResultStr)
    }

    resD := reply.RespData
    if x, ok := resD.(MsgEmptyResp); !ok {
        return LogError("SendActionResponse: Expect empty resp. (%T) (%v)", x, x)
    }

    LogInfo("SendActionResponse: succeeded (%s/%s)", tx.ClientName, res.Action)
    return nil

}


func (tx *ClientTx) NotifyHeartbeat(action string, tstamp EpochSecs) error {
    if tx.ClientRpc == nil {
        return LogError("RegisterAction: No Transport; Need to register first")
    }

    req := &LoMRequest { TypeNotifyActionHeartbeat, tx.ClientName, tx.TimeoutSecs, 
                MsgNotifyHeartbeat { action, tstamp }}
    reply := &LoMResponse{}
    err := tx.ClientRpc.Call("LoMTransport.SendToServer", req, reply)
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


