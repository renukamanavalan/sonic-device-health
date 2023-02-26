package lomipc


import(
    . "lomcommon"
    "net/rpc"
)

const server_address = "localhost"

type ClientTx struct {
    ClientRpc   *rpc.Client
    ClientName  string
}


func (tx *ClientTx) RegisterClient(client string, timeoutSeconds int) error {
    r, err := rpc.DialHTTP("tcp", server_address+":1234")
    if (err != nil) {
        LogError("Failed to call rpc.DialHTTP err:(%v)", err)
        return err
    }
    req := &LoMRequest { TypeRegClient, client, timeoutSeconds, MsgRegClient{} }
    // req := &LoMRequest { TypeRegClient, client, timeoutSeconds }
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

