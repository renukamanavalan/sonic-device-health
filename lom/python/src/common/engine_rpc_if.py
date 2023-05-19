#! /usr/bin/env python3

#
# Client lib to access engine via RPC-JSON i/f
#
# Connects to RPC JSON listener and communicate all client requests via
# JSON string and receive response as JSON too.
#

from dataclasses import dataclass
import json
import socket
import sys
import syslog
from types import SimpleNamespace

sys.path.append("../common")
from common import *
import gvars

# Port the engine listens at for RPC using JSON over raw sockets.
LOM_RPC_JSON_PORT = 1235

# Max bufsz for sock receive
# Any overflow is a bug, will result in transacion failure and get reported.
MAX_RECV_BUFSZ = 10240

# Every request to Engine is of the type LoMRequest.
# Sent to Engine has Json string (class dumped as JSON string).
# The contents of ReqData varies per ReqType.
# The various mapping types are listed below.
# If not specified for a ReqType, empty dict is sent.
#
@dataclass
class LoMRequest:
    ReqType:        gvars.TypeLoMReq
    Client:         str     # Client name
    TimeoutSecs:    int     # Req timeout. ==0 implies no timeout.
                            # Honored only for TypeRecvServerRequest
    ReqData:        {}    


# ReqData for register & de-register action.
# ReqType = TypeRegAction & TypeDeregAction
@dataclass
class MsgRegAction:
    Action:         str


# Response from plugin/action is sent to engine for ReqType = TypeSendServerResponse
# This is set on LomRequuest:ReqData
# ResType can be TypeServerRequestAction or TypeServerRequestShutdown
# For ResType == TypeServerRequestAction, the ResData is set to
# ActionResponseData. For shutdown, ResData is empty
#
@dataclass
class MsgSendServerResponse:
    ResType:        int
    ResData:        {}



# Response sent as ReqData for ReqType = TypeNotifyActionHeartbeat
@dataclass
class MsgNotifyHeartbeat:
    Action:     str
    Timestamp:  int


# MsgSendServerResponse:ResData for ResType == TypeServerRequestAction
@dataclass
class ActionResponseData:
    Action:             str
    InstanceId:         str
    AnomalyInstanceId:  str
    AnomalyKey:         str
    Response:           str
    ResultCode:         int
    ResultStr:          str


# Response sent by engine is of the type LoMResponse
# RespData is empty for all requests except for TypeRecvServerRequest
# For request with ReqType = TypeRecvServerRequest, RespData carries
# ServerRequestData.
@dataclass
class LoMResponse:
    ResultCode:         int
    ResultStr:          str
    RespData:           {}


# Request to plugin/action or Plugin Mgr is sent as response to
# client's request with ReqType = TypeRecvServerRequest
# ReqType can be TypeServerRequestAction or TypeServerRequestShutdown
# For ReqType == TypeServerRequestAction, the ReqData is set to
# ActionRequestData. For shutdown, ReqData is empty
#
@dataclass
class ServerRequestData:
    ReqType:        int
    ReqData:        {}


# ActionRequestData carries the detailed request to action/plugin.
#
@dataclass
class ActionRequestData:
    Action:             str
    InstanceId:         str
    AnomalyInstanceId:  str
    AnomalyKey:         str
    Timeout:            int
    Context:            [ActionResponseData]


# Client registration is once per session. Both connection & name are cached
# and reused until de-reg / re-reg client.
#
clientName = None
clientConn = None

def connectServer() -> bool:
    global clientConn

    # Register client *always* connect. Close if any existing.
    disconnectServer()

    try:
        conn = socket.create_connection(("localhost", LOM_RPC_JSON_PORT))
        clientConn = conn
    except socket.gaierror as e: 
        log_error("Address-related error connecting to server: %s" % e)
    except socket.error as e:
        log_error("Connection error: %s" % e)
    return clientConn != None


def disconnectServer():
    global clientConn

    if clientConn != None:
        clientConn.close()
        clientConn = None


def retFailedResponse(req: LoMRequest, code: int, msg: str) -> LoMResponse:
    msg = "Failed: Code={} msg={} req={}".format(code, msg, req)
    log_error(msg)
    return LoMResponse(code, msg, {})


ReqId = 0

# Send LoMRequest as JSON string
# Recv LoMResponse as JSON string
# Load parses string as SimpleNamespace with members matching LoMResponse
#
def sendAndReceive(req: LoMRequest) -> SimpleNamespace:
    # Just a uniq ID for RPC message to check against response
    # Rotating after 10K is more than good enough.
    global ReqId

    ReqId += 1
    if ReqId > 10240:
        ReqId = 1

    if clientConn == None:
        return retFailedResponse(req, -1, "No existing conn")

    msg = json.dumps(req, default=vars)
    rpcMsg = json.dumps({"id": ReqId, "method": "LoMTransport.LoMRPCRequest", "params": [msg]})
    print("DROP: send: ({})".format(rpcMsg))
    try:
        clientConn.sendall(rpcMsg.encode())
        recv = clientConn.recv(MAX_RECV_BUFSZ).decode()
    except socket.error as e: 
        return retFailedResponse(req, -1, "Error sending/receiving data: {}".format(e))

    if len(recv) == MAX_RECV_BUFSZ:
        # This signals a serious issue. Disconnect.
        disconnectServer()
        msg = "Message overflow(%d) s[{} ... {}]".format(len(recv),
            recv[0:40], recv[-40:])
        return retFailedResponse(req, -1, msg)

    print("DROP: recv: ({})".format(recv))
    try:
        rcvMsg = json.loads(recv, object_hook=lambda d: SimpleNamespace(**d))
    except json.JSONDecodeError as e:
        return retFailedResponse (req, -1, "Failing to decode: e:{} recv:{}".format(e, recv))

    if rcvMsg.error != None:
        return retFailedResponse(req, -1, "err:{}".format(rcvMsg.error))

    if rcvMsg.id != ReqId:
        return retFailedResponse(req, -1, "err: ID mismatch {} != {}".format(rcvMsg.id, ReqId))

    try:
        ret = json.loads(rcvMsg.result, object_hook=lambda d: SimpleNamespace(**d))
    except json.JSONDecodeError as e:
        return retFailedResponse (req, -1, "Failing to decode: e:{} recv:{}".format(e, rcvMsg.result))

    if ret.ResultCode != 0:
        return retFailedResponse (req, ret.ResultCode, "msg: {}".format(ret.ResultStr))

    return ret


# Register client
# First call in session.
# Creates a new connection to server and send the request.
# Returns success or not.
#
def register_client(clName: str) -> bool:
    global clientName

    connectServer()
    recv = sendAndReceive(LoMRequest(gvars.TypeLoMReq.TypeRegClient, clName, 0, {}))
    if recv.ResultCode == 0:
        clientName = clName
    else:
        disconnectServer()

    return recv.ResultCode == 0


# Deregister client
# Reset clientName & disconnect from server/engine.
#
def deregister_client():
    global clientName

    sendAndReceive(LoMRequest(gvars.TypeLoMReq.TypeDeregClient, clientName, 0, {}))
    clientName = None
    disconnectServer()
    return True


# Send register action request
def register_action(act: str) -> bool:
    recv = sendAndReceive(LoMRequest(gvars.TypeLoMReq.TypeRegAction, clientName, 0, MsgRegAction(act)))
    return recv.ResultCode == 0


# Send deregister action request
def deregister_action(act: str) -> bool:
    recv = sendAndReceive(LoMRequest(gvars.TypeLoMReq.TypeDeregAction, clientName, 0, MsgRegAction(act)))
    return recv.ResultCode == 0


# Send Heartbeat from action.
def regNotifyHB(act: str, ts: int):
    recv = sendAndReceive(LoMRequest(gvars.TypeLoMReq.TypeNotifyActionHeartbeat, clientName, 0, MsgNotifyHeartbeat(act, ts)))
    return recv.ResultCode == 0


# A blocking request to read a request from server.
# The request is either for a specific plugin/action or uber shutdown request.
# A non-zero timeout sets max time to wait. timeout == 0 blocks forever.
# Parsed JSON string is sent as SimpleNamespace
# This allows access all members as in LoMResponse.
#
def read_action_request(timeout: int = 0) -> (bool, SimpleNamespace):
    recv = sendAndReceive(LoMRequest(gvars.TypeLoMReq.TypeRecvServerRequest, clientName, timeout, {}))
    return (recv.ResultCode == 0, recv)


# Send response from plugin/action as request (gvars.TypeLoMReq.TypeSendServerResponse) to engine
#
def write_action_response(res: ActionResponseData) -> bool:
    recv = sendAndReceive(LoMRequest(gvars.TypeLoMReq.TypeSendServerResponse, clientName, 0, 
            MsgSendServerResponse(gvars.TypeServerReq.TypeServerRequestAction, res)))
    return recv.ResultCode == 0


def main():
    # To test - update engine_test.py to sleep after initServer. Make sure terminate
    # is given same value as timeout.
    # This starts engine and keep it listening
    # Now run the code below
    set_log_level(syslog.LOG_DEBUG)
    log_debug("Calling register_client for Foo") 
    ret = register_client("Foo")
    log_debug("register_client for Foo ret={}".format(ret))
 

if __name__ == "__main__":
    main()

