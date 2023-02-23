package ipc


import(
    "lomcmn"
    "errors"
    "fmt"
    "net/rpc"
)

const server_address = "127.0.0.1"

type ServerResult struct {
    ResultCode  int
    ResultStr   string
}

type ClientTx struct {
    clientRpc   *rpc.Client
    clientName  string
}


func (tx *ClientTx) RegisterClient(client string) (*ServerResult, error) {
    msg := &Msg { TypeRegClient, client, "", nil }
    r, err := rpc.DialHTTP("tcp", server_address+":1234")
    if (err != nil) {
        lomcmn.log_error("Failed to call rpc.DialHTTP err:(%v)", err)
        return nil, err
    }
    reply := &Reply 
    err = r.Call("LoMTransport.SendToServer", msg, reply)
    if (err != nil) {
        lomcmn.log_error("Failed to call SendToServer for RegClient (%s)", client)
        return nil, err
    }
    if (reply.ResultCode != 0) {
        lomcmn.log_error("Server failed to register client (%v) result(%d/%%s)", client,
                reply.ResultCode, reply.ResultStr)
        return nil, err
    }

    tx.clientRpc = r
    tx.clientName = client
    log_info("Registered client (%s)", client)
    return &ServerResult{0, ""}, nil
}

