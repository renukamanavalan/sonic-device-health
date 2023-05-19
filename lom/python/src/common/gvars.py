#! /usr/bin/env python3

from enum import auto, IntEnum

# Shared global definitions are held here
# This could be updated by any module
# To ensure to get the final value, import module 
# don't do from module import *
# refer as <module name>.<attr name>
#

REQ_CONTEXT             = "context"
REQ_ACTION              = "Action"
REQ_ANOMALY_INSTANCE_ID = "AnomalyInstanceId"
REQ_ANOMALY_KEY         = "AnomalyKey"
REQ_INSTANCE_ID         = "InstanceId"
REQ_RESP_DATA           = "RespData"
REQ_RESPONSE            = "Response"
REQ_RESULT_CODE         = "ResultCode"
REQ_RESULT_STR          = "ResultStr"
REQ_TIMESTAMP           = "Timestamp"


class TypeLoMReq(IntEnum):
    TypeRegClient = auto()
    TypeDeregClient = auto()
    TypeRegAction = auto()
    TypeDeregAction = auto()
    TypeRecvServerRequest = auto()
    TypeSendServerResponse = auto()
    TypeNotifyActionHeartbeat = auto()

class TypeServerReq(IntEnum):
    TypeServerRequestNone = auto()
    TypeServerRequestAction = auto()
    TypeServerRequestShutdown = auto()


# requests
# These are between clib client & server, hence mocked here.
REQ_REGISTER_CLIENT = "register_client"
REQ_DEREGISTER_CLIENT = "deregister_client"
REQ_REGISTER_ACTION = "register_action"
REQ_HEARTBEAT = "heartbeat"
REQ_ACTION_REQUEST = "action_request"

# Expected attribute names from CDLL for Action req/resp
# These can be refreshed from loaded DLL
# e.g. _get_str_globals("REQ_ACTION_TYPE")
#
REQ_ACTION_TYPE = "request_type"
REQ_ACTION_TYPE_ACTION = "action"
REQ_ACTION_TYPE_SHUTDOWN = "shutdown"

REQ_CLIENT_NAME = "client_name"
REQ_ACTION_NAME = "action_name"
REQ_HEARTBEAT_INTERVAL = "heartbeat_interval"
REQ_PAUSE = "action_pause"

REQ_ACTION_DATA = "action_data"

REQ_MITIGATION_STATE = "state" 
REQ_MITIGATION_STATE_INIT = "init"
REQ_MITIGATION_STATE_PROG = "in-progress"
REQ_MITIGATION_STATE_TIMEOUT = "timeout"
REQ_MITIGATION_STATE_DONE = "complete"

# run type
TEST_RUN = False
