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
        LogError("Failed to call rpc.DialHTTP err:(%v)", err)
        return err
    }
    req := &LoMRequest { TypeRegClient, client, tx.TimeoutSecs, MsgRegClient{} }
    reply := &LoMResponse{}
    err = r.Call("LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("Failed to call SendToServer for RegClient (%s) (%v)", client, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("Server failed to register client (%v) result(%d/%%s)", client,
                reply.ResultCode, reply.ResultStr)
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
        return LogError("No Transport; Need to register first")
    }

    req := &LoMRequest { TypeDeregClient, tx.ClientName, tx.TimeoutSecs, MsgDeregClient{} }
    reply := &LoMResponse{}
    err := tx.ClientRpc.Call("LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("Failed to call SendToServer for DeregClient (%s) (%v)", tx.ClientName, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("Server failed to deregister client (%v) result(%d/%%s)", tx.ClientName,
                reply.ResultCode, reply.ResultStr)
    }

    LogInfo("Deregistered client (%s)", tx.ClientName)
    return nil
}


func (tx *ClientTx) RegisterAction(action string) error {
    if tx.ClientRpc == nil {
        return LogError("No Transport; Need to register first")
    }

    req := &LoMRequest { TypeRegAction, tx.ClientName, tx.TimeoutSecs, MsgRegAction{action} }
    reply := &LoMResponse{}
    err := tx.ClientRpc.Call("LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("Failed to call SendToServer for RegAction (%s/%s) (%v)", tx.ClientName,
                action, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("Server failed to register action (%s/s%) result(%d/%%s)", tx.ClientName,
                action, reply.ResultCode, reply.ResultStr)
    }

    LogInfo("Registered action (%s/%s)", tx.ClientName, action)
    return nil
}

