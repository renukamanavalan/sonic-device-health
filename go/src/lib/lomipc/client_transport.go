package lomipc


import(
    . "lib/lomcommon"
    "net/rpc"
)

const server_address = "localhost"

var RPCDialHttp = rpc.DialHTTP

/* Client Transport object that has methods needed by clients */
type ClientTx struct {
    clientRpc   *rpc.Client
    clientName  string
    timeoutSecs int
}

func GetClientTx(tout int) *ClientTx {
    return &ClientTx{nil, "", tout}
}

func txCallClient(tx *ClientTx, serviceMethod string, args any, reply any) error {
    if tx.clientRpc == nil {
        return LogError("txCallClient: No Transport; Need to register first")
    }
    return tx.clientRpc.Call(serviceMethod, args, reply)
}

var ClientCall = txCallClient

/*
 * RegisterClient
 *  Registers the client.
 *  Creates the ClientTx object and register with engine.
 *
 * Input:
 *  client - Name of the client.
 *
 * Output:
 *  none
 *
 * Return:
 *  nil on success
 *  Appropriate error object on failure
 */
func (tx *ClientTx) RegisterClient(client string) error {
    r, err := RPCDialHttp("tcp", server_address+":1234")

    if (err != nil) {
        LogError("RegisterClient: Failed to call rpc.DialHTTP err:(%v)", err)
        return err
    }

    defer func() {
        if err != nil {
            tx.clientRpc = nil
            tx.clientName = ""
        }
    }()
    
    tx.clientRpc = r
    tx.clientName = client
    req := &LoMRequest { TypeRegClient, client, tx.timeoutSecs, MsgRegClient{} }
    reply := &LoMResponse{}
    err = ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("RegisterClient: Failed to call sendToServer (%s) (%v)", client, err)
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


/*
 * DeregisterClient
 *  Deregisters the client.
 *  Destroys the ClientTx object and after deregister with engine.
 *
 * Input:
 *  none
 *
 * Output:
 *  none
 *
 * Return:
 *  nil on success
 *  Appropriate error object on failure
 */
func (tx *ClientTx) DeregisterClient() error {
    defer func() {
        tx.clientRpc = nil
        tx.clientName = ""
    }()

    req := &LoMRequest { TypeDeregClient, tx.clientName, tx.timeoutSecs, MsgDeregClient{} }
    reply := &LoMResponse{}
    err := ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("DeregisterClient: Failed to call sendToServer (%s) (%v)", tx.clientName, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("DeregisterClient: Server failed (%v) result(%d/%s)", tx.clientName,
                reply.ResultCode, reply.ResultStr)
    }

    res := reply.RespData
    if x, ok := res.(MsgEmptyResp); !ok {
        return LogError("DeegisterClient: Expect empty resp. (%T) (%v)", x, x)
    }

    LogInfo("Deregistered client (%s)", tx.clientName)
    return nil
}


/*
 * RegisterAction
 *  Registers the Action.
 *  Sends the request to engine; wait for engine's response and return the same.
 *
 * Input:
 *  action - Name of the action.
 *
 * Output:
 *  none
 *
 * Return:
 *  nil on success
 *  Appropriate error object on failure
 */
func (tx *ClientTx) RegisterAction(action string) error {

    req := &LoMRequest { TypeRegAction, tx.clientName, tx.timeoutSecs, MsgRegAction{action} }
    reply := &LoMResponse{}
    err := ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("RegisterAction: Failed to call sendToServer (%s/%s) (%v)", tx.clientName,
                action, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("RegisterAction: Server failed (%s/%s) result(%d/%s)", tx.clientName,
                action, reply.ResultCode, reply.ResultStr)
    }

    res := reply.RespData
    if x, ok := res.(MsgEmptyResp); !ok {
        return LogError("RegisterAction: Expect empty resp. (%T) (%v)", x, x)
    }

    LogInfo("Registered action (%s/%s)", tx.clientName, action)
    return nil
}


/*
 * DeregisterAction
 *  Deregisters the Action.
 *  Sends the request to engine; wait for engine's response and return the same.
 *
 * Input:
 *  action - Name of the action.
 *
 * Output:
 *  none
 *
 * Return:
 *  nil on success
 *  Appropriate error object on failure
 */
func (tx *ClientTx) DeregisterAction(action string) error {

    req := &LoMRequest { TypeDeregAction, tx.clientName, tx.timeoutSecs, MsgDeregAction{action} }
    reply := &LoMResponse{}
    err := ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("DeregisterAction: Failed to call sendToServer (%s/%s) (%v)", tx.clientName,
                action, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("DeregisterAction: Server failed (%s/%s) result(%d/%s)", tx.clientName,
                action, reply.ResultCode, reply.ResultStr)
    }

    res := reply.RespData
    if x, ok := res.(MsgEmptyResp); !ok {
        return LogError("DeegisterAction: Expect empty resp. (%T) (%v)", x, x)
    }

    LogInfo("Deregistered action (%s/%s)", tx.clientName, action)
    return nil
}

/*
 * RecvServerRequest
 *  Receive request from the engine for an action registered by this client
 *  or a request to the client, like shutdown. The call blocks until engine
 *  raises a request.
 *
 * Input:
 *  none
 *
 * Output:
 *  none
 *
 * Return:
 *  Non nil ServerRequestData on success
 *  Appropriate error object on failure
 */
func (tx *ClientTx) RecvServerRequest() (*ServerRequestData, error) {

    req := &LoMRequest { TypeRecvServerRequest, tx.clientName, tx.timeoutSecs, MsgRecvServerRequest{} }
    reply := &LoMResponse{}
    err := ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("RecvServerRequest: Failed to call sendToServer (%s) (%v)", tx.clientName, err)
        return nil, err
    }
    if (reply.ResultCode != 0) {
        return nil, LogError("RecvServerRequest: Server failed (%s) result(%d/%s)", tx.clientName,
                reply.ResultCode, reply.ResultStr)
    }

    p := reply.RespData
    res, ok := p.(ServerRequestData)
    if !ok {
        return nil, LogError("RecvServerRequest: RespData (%T) != *ActionRequestData", res)
    }

    LogInfo("RecvServerRequest: succeeded (%s/%s)", tx.clientName,
                    ServerReqTypeToStr[res.ReqType])
    return &res, nil
}

/*
 * SendServerResponse
 *  Client uses this to send Action returned response to the engine.
 *  Wait till engine's response and return
 *
 * Input:
 *  res - Response to send back.
 *
 * Output:
 *  none
 *
 * Return:
 *  nil on success
 *  Appropriate error object on failure
 */
func (tx *ClientTx) SendServerResponse(res *MsgSendServerResponse) error {
    req := &LoMRequest { TypeSendServerResponse, tx.clientName, tx.timeoutSecs, res }
    reply := &LoMResponse{}
    err := ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("SendServerResponse: Failed to call sendToServer (%s) (%v)", tx.clientName, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("SendServerResponse: Server failed (%s) result(%d/%s)", tx.clientName,
                reply.ResultCode, reply.ResultStr)
    }

    resD := reply.RespData
    if x, ok := resD.(MsgEmptyResp); !ok {
        return LogError("SendServerResponse: Expect empty resp. (%T) (%v)", x, x)
    }

    LogInfo("SendServerResponse: succeeded (%s/%s)", tx.clientName, ServerReqTypeToStr[res.ReqType])
    return nil

}


/*
 * NotifyHeartbeat
 *  Client conveys heartbeats from action to engine.
 *  Wait till engine's response and return
 *
 * Input:
 *  action - Name of the action.
 *  tstamp - Timestamp of heartbeat
 *
 * Output:
 *  none
 *
 * Return:
 *  nil on success
 *  Appropriate error object on failure
 */
func (tx *ClientTx) NotifyHeartbeat(action string, tstamp int64) error {
    req := &LoMRequest { TypeNotifyActionHeartbeat, tx.clientName, tx.timeoutSecs, 
                MsgNotifyHeartbeat { action, tstamp }}
    reply := &LoMResponse{}
    err := ClientCall(tx, "LoMTransport.SendToServer", req, reply)
    if (err != nil) {
        LogError("NotifyHeartbeat: Failed to call sendToServer (%s/%s) (%v)", tx.clientName,
                action, err)
        return err
    }
    if (reply.ResultCode != 0) {
        return LogError("NotifyHeartbeat: Server failed (%s/%s) result(%d/%s)", tx.clientName,
                action, reply.ResultCode, reply.ResultStr)
    }

    res := reply.RespData
    if x, ok := res.(MsgEmptyResp); !ok {
        return LogError("NotifyHeartbeat: Expect empty resp. (%T) (%v)", x, x)
    }

    LogInfo("Notified heartbeat from action (%s/%s)", tx.clientName, action)
    return nil
}


