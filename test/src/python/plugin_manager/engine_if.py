#! /usr/bin/env python3

from dataclasses import dataclass
import json
import socket
import sys
from types import SimpleNamespace

syspath_append("../common")
from common import *
import gvars

RPC_JSON_PORT = 1235
MAX_RECV_BUFSZ = 4096

@dataclass
class LoMRequest:
    ReqType:        int
    Client:         str
    TimeoutSecs:    int
    ReqData:        {}


@dataclass
class MsgRegAction:
    Action:         str


@dataclass
class MsgSendServerResponse:
    ReqType:        int
    ResData:        {}


@dataclass
class MsgNotifyHeartbeat:
    Action:     str
    Timestamp:  int


@dataclass
class ActionResponseData:
    Action:             str
    InstanceId:         str
    AnomalyInstanceId:  str
    AnomalyKey:         str
    Response:           str
    ResultCode:         int
    ResultStr:          str

@dataclass
class LoMResponse:
    ResultCode:         int
    ResultStr:          str
    RespData:           {}


clientName = ""
clientConn = None

def connectServer():
    # Register client *always* connect. Close if any existing.
    if clientConn != none:
        clientConn.close()

    clientConn = socket.create_connection(("localhost", LOM_RPC_JSON_PORT))


def sendAndReceive(req: LoMRequest) -> {}:
    if clientConn == None:
        return LoMResponse(-1, "No existing conn", {})

    msg = json.dumps(req, default=vars)
    clientConn.sendall(msg.encode())
    recv = clientConn.recv(MAX_RECV_BUFSZ).decode()
    ret = json.loads(recv, object_hook=lambda d: SimpleNamespace(**d))
    if recv.ResultCode != 0
        LogError("req({}) failed with ({})".format(req, recv))
    return ret


def register_client(clName: str) -> bool:
    global clientName

    connectServer()
    clientName = clName
    recv = sendAndReceive(LoMRequest(TypeRegClient, clName, 0, {}))
    return recv.ResultCode == 0


def deregister_client():
    sendAndReceive(LoMRequest(TypeDeregClient, clientName, 0, {}))


def register_action(act: str) -> bool:
    recv = sendAndReceive(LoMRequest(TypeRegAction, clientName, 0, MsgRegAction(act)))
    return recv.ResultCode == 0


def deregister_action(act: str) -> bool:
    recv = sendAndReceive(LoMRequest(TypeDeregAction, clientName, 0, MsgDeregAction(act)))
    return recv.ResultCode == 0


def regNotifyHB(act: str, tout: int):
    recv = sendAndReceive(LoMRequest(TypeRegAction, clientName, 0, MsgNotifyHeartbeat(act, tout)))
    return recv.ResultCode == 0


def read_action_request(timeout:int = -1) -> (bool, SimpleNamespace):
    recv = sendAndReceive(TypeRecvServerRequest, clientName, timeout, {})
    return (recv.ResultCode == 0, recv)


def write_action_response(res ActionResponseData) -> bool:
    recv = sendAndReceive(LoMRequest(TypeSendServerResponse, clientName, 0, 
            MsgSendServerResponse(TypeServerRequestAction, res)))
    return recv.ResultCode == 0



