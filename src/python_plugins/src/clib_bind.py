#! /usr/bin/env python3

from common import *
import gvars
from ctypes import *

# *******************************
# c-bindings related info
# *******************************
#

_CT_DIR = os.path.dirname(os.path.abspath(__file__))

# TODO: Get list of error codes defined for various errors

# Set Clib .so file path here
CLIB_DLL_FILE = os.path.join(_CT_DIR, "../../lom_lib.so")

_clib_dll = None
_clib_set_test_mode = None
_clib_is_test_mode = None

_clib_set_log_level = None
_clib_get_log_level = None

_clib_log_write = None

_clib_set_thread_name = None

_clib_get_last_error = None
_clib_get_last_error_str = None
_clib_register_client = None
_clib_deregister_client = None
_clib_register_action = None
_clib_touch_heartbeat = None
_clib_read_action_request = None
_clib_write_action_response = None
_clib_poll_for_data = None

# Load c-APIs for server for Python test code

_clib_server_init = None
_clib_server_deinit = None
_clib_write_server_message_c = None
_clib_read_server_message_c = None


def c_lib_init() -> bool:
    global _clib_dll
    global _clib_get_last_error, _clib_get_last_error_str, _clib_register_client
    global _clib_deregister_client, _clib_register_action, _clib_touch_heartbeat
    global _clib_read_action_request, _clib_write_action_response, _clib_poll_for_data
    global _clib_server_init, _clib_server_deinit
    global _clib_write_server_message_c, _clib_read_server_message_c
    global _clib_set_test_mode, _clib_is_test_mode, _clib_set_thread_name
    global _clib_set_log_level, _clib_get_log_level, _clib_log_write

    if _clib_dll:
        return True

    if not gvars.MOCK_LIB:
        try:
            _clib_dll = CDLL(CLIB_DLL_FILE)
        except OSError as e:
            log_error("Failed to load CDLL {} err: {}".format(CLIB_DLL_FILE, str(e)))
            return False

        try:
            _clib_register_client = _clib_dll.register_client
            _clib_register_client.argtypes = [ c_char_p, POINTER(c_int) ]
            _clib_register_client.restype = c_int

            _clib_register_action = _clib_dll.register_action
            _clib_register_action.argtypes = [ c_char_p ]
            _clib_register_action.restype = c_int

            _clib_deregister_client = _clib_dll.deregister_client
            _clib_deregister_client.argtypes = [ c_char_p ]
            _clib_deregister_client.restype = c_int

            _clib_touch_heartbeat = _clib_dll.touch_heartbeat
            _clib_touch_heartbeat.argtypes = [ c_char_p, c_char_p ]
            _clib_touch_heartbeat.restype = c_int

            _clib_read_action_request = _clib_dll.read_action_request
            _clib_read_action_request.argtypes = [c_int]
            _clib_read_action_request.restype = c_char_p

            _clib_write_action_response = _clib_dll.write_action_response
            _clib_write_action_response.argtypes = [ c_char_p ]
            _clib_write_action_response.restype = c_int

            _clib_poll_for_data = _clib_dll.poll_for_data
            _clib_poll_for_data.argtypes = [ POINTER(c_int), c_int, POINTER(c_int), POINTER(c_int),
                    POINTER(c_int), POINTER(c_int) , c_int ]
            _clib_poll_for_data.restype = c_int

            _clib_server_init = _clib_dll.server_init_c
            _clib_server_init.argtypes = [ POINTER(c_char_p), c_int ]
            _clib_server_init.restype = c_int

            _clib_server_deinit = _clib_dll.server_deinit
            _clib_server_deinit.argtypes = []
            _clib_server_deinit.restype = None

            _clib_write_server_message_c = _clib_dll.write_server_message_c
            _clib_write_server_message_c.argtypes = [ c_char_p ]
            _clib_write_server_message_c.restype = c_int

            _clib_read_server_message_c = _clib_dll.read_server_message_c
            _clib_read_server_message_c.argtypes = [ c_int ]
            _clib_read_server_message_c.restype = c_char_p

            _clib_get_last_error = _clib_dll.lom_get_last_error
            _clib_get_last_error.argtypes = []
            _clib_get_last_error.restype = c_int

            _clib_get_last_error_str = _clib_dll.lom_get_last_error_msg
            _clib_get_last_error_str.argtypes = []
            _clib_get_last_error_str.restype = c_char_p

            _clib_set_test_mode = _clib_dll.set_test_mode
            _clib_set_test_mode.argtypes = []
            _clib_set_test_mode.restype = None

            _clib_is_test_mode = _clib_dll.is_test_mode
            _clib_is_test_mode.argtypes = []
            _clib_is_test_mode.restype = c_bool

            _clib_set_log_level = _clib_dll.set_log_level
            _clib_set_log_level.argtypes = [ c_int ]
            _clib_set_log_level.restype = None

            _clib_get_log_level = _clib_dll.get_log_level
            _clib_get_log_level.argtypes = []
            _clib_get_log_level.restype = c_int

            _clib_log_write = _clib_dll.log_write
            _clib_log_write.argtypes = [ c_int, c_char_p , c_char_p ]
            _clib_log_write.restype = None

            _clib_set_thread_name = _clib_dll.set_thread_name
            _clib_set_thread_name.argtypes = [ c_char_p ]
            _clib_set_thread_name.restype = None

            # Update values in gvars.py
            _update_globals()

        except Exception as e:
            log_error("Failed to load functions from CDLL {} err: {}".
                    format(CLIB_DLL_FILE, str(e)))
            _clib_dll = None
            return False
    else:
        import test_client
        log_debug("clib in test mode")

        _clib_get_last_error = test_client.clib_get_last_error
        _clib_get_last_error_str = test_client.clib_get_last_error_str
        _clib_register_client = test_client.clib_register_client
        _clib_deregister_client = test_client.clib_deregister_client
        _clib_register_action = test_client.clib_register_action
        _clib_touch_heartbeat = test_client.clib_touch_heartbeat
        _clib_read_action_request = test_client.clib_read_action_request
        _clib_write_action_response = test_client.clib_write_action_response
        _clib_poll_for_data = test_client.clib_poll_for_data
        _clib_server_init = test_client.clib_server_init
        _clib_server_deinit = test_client.clib_server_deinit
        _clib_write_server_message_c = test_client.clib_write_server_message_c
        _clib_read_server_message_c = test_client.clib_read_server_message_c

        _clib_set_test_mode = test_client.clib_set_test_mode
        _clib_is_test_mode = test_client.clib_is_test_mode

        _clib_set_log_level = test_client.clib_set_log_level
        _clib_get_log_level = test_client.clib_get_log_level
        _clib_log_write = test_client.clib_log_write
        _clib_set_thread_name = test_client.clib_set_thread_name

        _clib_dll = "Test mode"
        
    return True


def validate_dll():
    if not _clib_dll:
        log_error("CLib is not loaded. Failed.")
        return False
    return True


def get_last_error() -> (int, str):
    return _clib_get_last_error(), _clib_get_last_error_str().decode("utf-8")


def _print_clib_error(m:str, ret:int):
    err, estr = get_last_error()
    log_error("{}: ret:{} last_error:{} ({})".format(m, ret, err, estr))


def register_client(proc_id: str) -> (bool, int):
    if not validate_dll():
        return False, {}

    cfd  = c_int(0)
    ret = _clib_register_client(proc_id.encode("utf-8"), cfd)
    if ret != 0:
        log_error("register_client failed for {}".format(proc_id))
        return False, -1
    return True, cfd.value


def register_action(action: str) -> bool:
    if not validate_dll():
        return False, {}

    ret = _clib_register_action(action.encode("utf-8"))
    if ret != 0:
        log_error("register_action failed {}".format(action))
        return False

    log_info("clib_bind: register_action {}".format(action))
    return True


def deregister_client(proc_id: str):
    if not validate_dll():
        return False, {}

    _clib_deregister_client(proc_id.encode("utf-8"))


def touch_heartbeat(action: str, instance_id: str) -> bool:
    if not validate_dll():
        return False, {}

    ret = _clib_touch_heartbeat(action.encode("utf-8"), instance_id.encode("utf-8"))
    if ret != 0:
        log_error("touch_heartbeat failed action:{} id:{}".format(action, instance_id))
        return False
    return True


# CLIB globals
def _get_str_clib_globals(name:str) -> str:
    return (c_char_p.in_dll(_clib_dll, name)).value.decode("utf-8")



def _update_globals():
    gvars.REQ_ACTION_TYPE = _get_str_clib_globals("REQ_ACTION_TYPE")
    gvars.REQ_ACTION_TYPE_ACTION = _get_str_clib_globals("REQ_ACTION_TYPE_ACTION")
    gvars.REQ_ACTION_TYPE_SHUTDOWN = _get_str_clib_globals("REQ_ACTION_TYPE_SHUTDOWN")

    gvars.REQ_CLIENT_NAME = _get_str_clib_globals("REQ_CLIENT_NAME")
    gvars.REQ_ACTION_NAME = _get_str_clib_globals("REQ_ACTION_NAME")
    gvars.REQ_INSTANCE_ID = _get_str_clib_globals("REQ_INSTANCE_ID")
    gvars.REQ_ANOMALY_INSTANCE_ID = _get_str_clib_globals("REQ_ANOMALY_INSTANCE_ID")
    gvars.REQ_ANOMALY_KEY = _get_str_clib_globals("REQ_ANOMALY_KEY")
    gvars.REQ_CONTEXT = _get_str_clib_globals("REQ_CONTEXT")
    gvars.REQ_TIMEOUT = _get_str_clib_globals("REQ_TIMEOUT")
    gvars.REQ_ACTION_DATA = _get_str_clib_globals("REQ_ACTION_DATA")
    gvars.REQ_RESULT_CODE = _get_str_clib_globals("REQ_RESULT_CODE")
    gvars.REQ_RESULT_STR  = _get_str_clib_globals("REQ_RESULT_STR")
    gvars.REQ_REGISTER_CLIENT = _get_str_clib_globals("REQ_REGISTER_CLIENT")
    gvars.REQ_DEREGISTER_CLIENT = _get_str_clib_globals("REQ_DEREGISTER_CLIENT")
    gvars.REQ_REGISTER_ACTION = _get_str_clib_globals("REQ_REGISTER_ACTION")
    gvars.REQ_HEARTBEAT = _get_str_clib_globals("REQ_HEARTBEAT")
    gvars.REQ_ACTION_REQUEST = _get_str_clib_globals("REQ_ACTION_REQUEST")
    gvars.REQ_ACTION_RESPONSE = _get_str_clib_globals("REQ_ACTION_RESPONSE")
    gvars.REQ_HEARTBEAT_INTERVAL = _get_str_clib_globals("REQ_HEARTBEAT_INTERVAL")
    gvars.REQ_PAUSE = _get_str_clib_globals("REQ_PAUSE")
    gvars.REQ_MITIGATION_STATE = _get_str_clib_globals("REQ_MITIGATION_STATE" )
    gvars.REQ_MITIGATION_STATE_INIT = _get_str_clib_globals("REQ_MITIGATION_STATE_INIT")
    gvars.REQ_MITIGATION_STATE_PROG = _get_str_clib_globals("REQ_MITIGATION_STATE_PROG")
    gvars.REQ_MITIGATION_STATE_TIMEOUT = _get_str_clib_globals("REQ_MITIGATION_STATE_TIMEOUT")
    gvars.REQ_MITIGATION_STATE_DONE = _get_str_clib_globals("REQ_MITIGATION_STATE_DONE")


class ActionRequest:
    def __init__(self, sdata: str):
        self.str_data = sdata
        data = json.loads(sdata)[gvars.REQ_ACTION_REQUEST]
        self.type = data[gvars.REQ_ACTION_TYPE]
        if self.type == gvars.REQ_ACTION_TYPE_ACTION:
            self.client_name = data[gvars.REQ_CLIENT_NAME]
            self.action_name = data[gvars.REQ_ACTION_NAME]
            self.instance_id = data[gvars.REQ_INSTANCE_ID]
            self.anomaly_instance_id = data[gvars.REQ_ANOMALY_INSTANCE_ID]
            self.anomaly_key = data[gvars.REQ_ANOMALY_KEY]
            self.context = data[gvars.REQ_CONTEXT]
            self.timeout = data[gvars.REQ_TIMEOUT]

    def __repr__(self):
        return self.str_data

    def is_shutdown(self) -> bool:
        return self.type == gvars.REQ_ACTION_TYPE_SHUTDOWN


def read_action_request(timeout:int = -1) -> (bool, ActionRequest):
    if not validate_dll():
        return False, {}

    req = _clib_read_action_request(timeout).decode("utf-8")

    if not req:
        e, estr = get_last_error()
        if e and (timeout == -1):
            log_error("read_action_request failed")
        else:
            log_info("read_action_request timedout. timeout={}".format(timeout))
        return False, None

    return True, ActionRequest(req)



class ActionResponse:
    def __init__(self, client_name:str, action_name:str, instance_id:str,
            anomaly_instance_id:str, anomaly_key:str, action_data: str,
            result_code:int, result_str:str) :
        self.data = json.dumps({gvars.REQ_ACTION_RESPONSE: {
                gvars.REQ_CLIENT_NAME: client_name,
                gvars.REQ_ACTION_NAME: action_name,
                gvars.REQ_ACTION_TYPE: gvars.REQ_ACTION_TYPE_ACTION,
                gvars.REQ_INSTANCE_ID: instance_id,
                gvars.REQ_ANOMALY_INSTANCE_ID: anomaly_instance_id,
                gvars.REQ_ANOMALY_KEY: anomaly_key,
                gvars.REQ_ACTION_DATA: action_data,
                gvars.REQ_RESULT_CODE: str(result_code),
                gvars.REQ_RESULT_STR : result_str }})

                
    def __repr__(self) -> str:
        return self.data 

    def value(self) -> str:
        return self.data 


def write_action_response(res: ActionResponse) -> bool:
    if not validate_dll():
        return False

    ret = _clib_write_action_response(
            res.value().encode("utf-8"))

    if ret != 0:
        log_error("write_action_response failed")
        return False

    return True


def poll_for_data(lst_fds: [int], timeout:int,
        ready_fds: [int], err_fds: [int]) -> int:
    if not validate_dll():
        return False

    lcnt = len(lst_fds)
    elst = [-1] * lcnt

    clst_fds = (c_int * lcnt)(*lst_fds)
    clcnt = c_int(lcnt)
    crfd = (c_int * lcnt)(*elst)
    cefd = (c_int * lcnt)(*elst)
    crcnt  = c_int(0)
    cecnt  = c_int(0)

    DROP_TEST("fds=({})".format(lst_fds))
    ret = _clib_poll_for_data(clst_fds, clcnt, crfd, crcnt, cefd, cecnt, c_int(timeout))

    lrfd = list(crfd)
    lefd = list(cefd)

    for i in range(crcnt.value):
        ready_fds.append(lrfd[i])

    for i in range(cecnt.value):
        err_fds.append(lefd[i])

    DROP_TEST("ret={} crcnt={} readyfds=({})".format(ret, crcnt.value, ready_fds))
    return ret


def server_init(slst: [str]) -> int:
    lst = []
    for i in slst:
        lst.append(i.encode("utf-8"))
    clients = (c_char_p * len(lst))(*lst)

    return _clib_server_init(clients, len(lst))


def server_deinit():
    _clib_server_deinit()


def write_server_message(msg: str) -> int:
    ret = _clib_write_server_message_c(msg.encode("utf-8"))
    return ret


def read_server_message(tout: int) -> str:
    return _clib_read_server_message_c(tout).decode("utf-8")


def set_test_mode():
    _clib_set_test_mode()


def is_test_mode() -> bool:
    return _clib_is_test_mode()


def set_log_level(lvl: int):
    _clib_set_log_level(c_int(lvl))


def get_log_level() -> int:
    return _clib_get_log_level()

def log_write(lvl: int, caller: str, msg: str):
    _clib_log_write(c_int(lvl), caller.encode("utf-8"), msg.encode("utf-8"))


def set_thread_name(name: str):
    _clib_set_thread_name(name.encode("utf-8"))

