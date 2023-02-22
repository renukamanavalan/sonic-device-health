package client_transport


import(
    "errors"
    "fmt"
    "net/rpc"
    "transport"
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

func (tx *ClientTx) RegisterClient(client string) *ServerResult, errors {
    msg := &transport.Msg {
        transport.TypeRegClient,
        client,
        ""}
    r, err := rpc.DialHTTP("tcp", server_address+":1234")
    if (err != nil) {
        log.Printf("Failed to call rpc.DialHTTP err:(%v)", err)
        return nil, err
    }
    reply := &transport.Reply 
    err = r.Call("LoMTransport.SendToServer", msg, reply)
    if (err != nil) {
        log.Printf("Failed to call SendToServer for RegClient (%s)", client)
        return nil, err
    }
    if (reply.ResultCode != 0) {
        log.Printf("Server failed to register client (%v) result(%d/%%s)", client,
                reply.ResultCode, reply.ResultStr)
        return nil, err
    }

    tx.clientRpc = r
    tx.clientName = client
    return &ServerResult{0, ""}, nil
}

