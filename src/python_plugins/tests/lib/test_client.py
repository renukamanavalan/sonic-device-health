#! /usr/bin/env python3

from enum import Enum
from ctypes import *
import os
import select
import time
import threading

import gvars

from common import *

# This module helps mock c-bindings calls to server.
# This enable a test code to act as server/engine
#  
# This mimics every client & sever API
# Binds writes & reads between client & server via internal mocked
# cache service.
#
# Way it works:
#   The test code invokes the plugin_proc.py in a different thread.
#   When there are multiple procs multiple threads will be created as one per proc.
#   Each thread gets its own cache service instance using the trailing number in 
#   its name as index into list of pre-created cache service instances.
#   The cache service mimics the R/W channels between main thread & proc.
#
#   Each cache-service instance has two caches as one for each direction
#   Each cache instance has a pipe with two fds as one for read & write.
#
#   Main thread owns caches from all services for server to client.
#   Proc thread owns cache for client to server.
#
#   Main thread mimics the server.
#   Main thread collects write fds from all cache instances for server to client
#   and collects rd fs from all client to server cache instances
#   These fds are used for signalling between main thread and the thread that owns
#   the cache instance.
#   Cache supports rd/wr index and read / write methods.
#
#   Any write, the main thread writes into appropriate client instance.
#
#   It listens on signals from all collected rd fds and read from signalling
#   cache instances.
#
#   This test code acts mimics engine in the main thread.
#   In non-test scenario, the supervisord from container manages the processes
#   for each proc.
#
# All requests/responses across are saved in a list capped by size.
# Each side adding to list, signals the other via a pipe.
#

# Create this just once and use
# Each thread transparently gets its own copy
#
th_local = threading.local()

CACHE_LIMIT = 100

shutdown = False

# TODO: Mimic error codes defined from clib_bind

test_error_code = 0
test_error_str = ""

def _report_error(code: int, errMsg: str):
    global test_error_code, test_error_str

    test_error_code = code
    test_error_str = errMsg
    if code != 0:
        log_error("ERROR: {}".format(errMsg))
    return code

def _reset_error():
    global test_error_code, test_error_str

    test_error_code = 0
    test_error_str = ""


def _poll(rdfds:[], timeout: int) -> [int]:
    while (not shutdown):
        poll_wait = 2

        if (timeout >= 0):
            if (timeout < poll_wait):
                poll_wait = timeout
            timeout -= poll_wait

        r, _, _ = select.select(rdfds, [], [], poll_wait)
        if r:
            return r

        if timeout == 0:
            return []

    return []


# Caches data in one direction with indices for nxt, cnt to relaize Q full
# state and data buffer.
#
SIGNAL_MSG = b"data"

class CacheData:
    def __init__(self, limit:int, c2s:bool):
        self.limit = limit
        self.rd_index = 0
        self.wr_index = 0
        self.rd_cnt = 0
        self.wr_cnt = 0
        self.data: { int: {} } = {}
        self.c2s = c2s      # True for client to server direction
        self.signal_rd, self.signal_wr = os.pipe()


    def get_signal_rd_fd(self) -> int:
        return self.signal_rd


    def get_signal_wr_fd(self) -> int:
        return self.signal_wr


    def _drain_signal(self):
        # Drain a single signal only
        lst = [ self.signal_rd ]
        r = _poll(lst, 0)
        log_debug("***** call _drain signal: c2s:{} rd:{} r={}".format(self.c2s, self.signal_rd, r))
        if self.signal_rd in r:
            os.read(self.signal_rd, len(SIGNAL_MSG))


    def _raise_signal(self):
        log_debug("***** Raised signal: c2s:{} rd:{}".format(self.c2s, self.signal_rd))
        os.write(self.signal_wr, SIGNAL_MSG)


    def write(self, data: {}) -> bool:
        # Test code never going to rollover. So ignore cnt rollover possibility.
        #
        if (self.wr_cnt - self.rd_cnt) >= self.limit:
            log_error("c2s:{} write overflow. dropped {}/{}".format(
                self.c2s, self.wr_cnt, self.rd_cnt))
            return False

        self.data[self.wr_index] = data
        self.wr_index += 1
        if self.wr_index >= self.limit:
            self.wr_index = 0
        self.wr_cnt += 1
        self._raise_signal()
        return True


    # read with optional timeout.
    # timeout =
    #   0 -- Return immediately with or w/o data
    #  <0 -- Block until data is available for read
    #  >0 -- Wait for these many seconds for data
    # 
    def read(self, timeout=-1) -> (bool, {}):
        r = _poll([self.signal_rd], timeout)
        self._drain_signal()

        if self.rd_cnt < self.wr_cnt:
            # copy as other thread could write, upon rd_cnt incremented.
            #
            ret = { k:v for k, v in self.data[self.rd_index].items()}
            self.rd_index += 1
            if self.rd_index >= self.limit:
                self.rd_index = 0
            self.rd_cnt += 1
            return True, ret
        else:
            msg = "c2s:{} read empty. {}/{} timeout:{}".format(
                self.c2s, self.wr_cnt, self.rd_cnt, timeout)
            if timeout != -1:
                log_info(msg)
            else:
                log_error(msg)

            return False, {}


class cache_service:
    def __init__(self, proc_name: [str], limit:int=CACHE_LIMIT):
        # Get cache for both directions
        self.c2s = CacheData(limit, True)
        self.s2c = CacheData(limit, False)
        self.proc_name = proc_name


    def get_proc_name(self) ->str:
        return self.proc_name

    def write_to_server(self, d: {}) -> bool:
        return self.c2s.write(d)

    def write_to_client(self, d: {}) -> bool:
        return self.s2c.write(d)

    def read_from_server(self, timeout:int = -1) -> (bool, {}):
        return self.s2c.read(timeout)

    def read_from_client(self, timeout:int = -1) -> (bool, {}):
        return self.c2s.read(timeout)

    def get_signal_rd_fd(self, is_c2s:bool) -> int:
        if is_c2s:
            return self.c2s.get_signal_rd_fd()
        else:
            return self.s2c.get_signal_rd_fd()


    def get_signal_wr_fd(self, is_c2s:bool) -> int:
        if is_c2s:
            return self.c2s.get_signal_wr_fd()
        else:
            return self.s2c.get_signal_wr_fd()


lst_cache_services = {}
server_rd_fds = {}  # fd : name


#
# Each Python plugin proc is created in its own thread.
# The server init is given the list of expected procs.
# The server pre-creates cache for each by name.
#
# Each proc takes its instance by name and save it in
# thread local. Rest of the calls from this thread uses this
# instance.
#
def _create_cache_services(lst: [str]):
    global lst_cache_services
    global server_rd_fds

    for i in lst:
        p = cache_service(i)
        server_rd_fds[p.get_signal_rd_fd(True)] = i;
        lst_cache_services[i] = p


#
# Mocker clib server calls
#
def server_init(cl: POINTER(c_char_p), cnt: c_int) -> c_int:
    clients = []
    for i in cl:
        clients.append(i.decode("utf-8"))
    _create_cache_services(clients)
    _reset_error()
    return c_int(0) 


def server_deinit():
    lst_cache_services = {}
    _reset_error()


def write_server_message_c(bmsg: c_char_p) -> c_int:
    _reset_error()
    ret = 0;
    msg = bmsg.decode("utf-8")

    d = json.loads(msg)
    if (len(d) > 1):
        ret = _report_error(-1, "Expect only entry ({})".format(msg))

    elif (list(d)[0] != gvars.REQ_ACTION_REQUEST):
        ret = _report_error(-1, "Server writes action-request only. ({})".format(msg))

    elif gvars.REQ_CLIENT_NAME not in d[gvars.REQ_ACTION_REQUEST]:
        ret = _report_error(-1, "Missing client_name ({})".format(msg))

    else:
        cl_name = d[gvars.REQ_ACTION_REQUEST][gvars.REQ_CLIENT_NAME]
        if cl_name not in lst_cache_services:
            ret = _report_error(-1, "Missing client_name {} cache services".format(cl_name))

        elif not lst_cache_services[cl_name].write_to_client(d):
            ret = _report_error(-1, "Failed to write")

    return c_int(ret)


def read_server_message_c(tout : c_int) -> c_char_p:
    _reset_error()
    ret_data = ""

    lst = list(server_rd_fds)
    r = _poll(lst, tout)
    if not r:
        _report_error(-1, "read timeout")
        return "".encode("utf-8")

    cl_name = server_rd_fds[r[0]]
    ret, d = lst_cache_services[cl_name].read_from_client(0)

    if not ret:
        _report_error(-1, "Failed to read")
    elif len(d) != 1:
        _report_error(-1, "Internal error. Expected one key. ({})".
                format(json.dumps(d)))
    elif list(d)[0] not in [ gvars.REQ_REGISTER_CLIENT, gvars.REQ_DEREGISTER_CLIENT,
            gvars.REQ_REGISTER_ACTION, gvars.REQ_HEARTBEAT, gvars.REQ_ACTION_RESPONSE]:
        _report_error("Internal error. Unexpected request: {}".format(json.dumps(d)))
    else:
        ret_data = json.dumps(d)

    return ret_data.encode("utf-8")


#
# Mocked clib client calls
#
def clib_get_last_error() -> int:
    return test_error_code


def clib_get_last_error_str() -> str:
    return test_error_str


def _is_initialized():
    return getattr(th_local, 'cache_svc', None) is not None


def clib_register_client(bcl_name: bytes, fd: c_int) -> c_int:
    _reset_error()
    ret = 0

    cl_name = bcl_name.decode("utf-8")
    if _is_initialized():
        ret = _report_error(-1, "Duplicate registration {}".format(cl_name))

    elif cl_name not in lst_cache_services:
        # Proc index must run from 0 sequentially as services
        # are created for count of entries in proc's conf.
        #
        ret = _report_error(-1, "client:{} not in cache services created by server".format(cl_name))

    else:
        log_info("Registered:{}".format(cl_name))

        th_local.cache_svc = lst_cache_services[cl_name]
        th_local.cl_name = cl_name
        th_local.actions = []

        fd.value = th_local.cache_svc.get_signal_rd_fd(False)

        rc = th_local.cache_svc.write_to_server({
            gvars.REQ_REGISTER_CLIENT: {
                gvars.REQ_CLIENT_NAME: cl_name }})

        if not rc:
            ret = _report_error(-1, "client failed to write register client {}".
                    format(cl_name))
    return c_int(ret)


def clib_deregister_client(bcl_name: bytes) -> c_int:
    _reset_error()
    ret = 0
    cl_name = bcl_name.decode("utf-8")
    if not _is_initialized():
        ret = _report_error(-1, "deregister_client: client not registered {}".format(cl_name))

    else:
        rc = th_local.cache_svc.write_to_server({
            gvars.REQ_DEREGISTER_CLIENT: {
                gvars.REQ_CLIENT_NAME: cl_name }})
        if not rc:
            ret = _report_error(-1, "client failed to write deregister client {}".
                    format(cl_name))

        # Clean local cache
        th_local.cache_svc = None
        th_local.cl_name = None
        th_local.actions = None

    return c_int(ret)


def clib_register_action(baction_name: bytes) -> c_int:
    _reset_error()
    ret = 0;
    action_name = baction_name.decode("utf-8")
    if not _is_initialized():
        ret = _report_error(-1, "register_action: client not registered {}".format(action_name))

    elif action_name in th_local.actions:
        ret = _report_error(-2, "Duplicate registration {}".format(action_name))

    else:
        th_local.actions.append(action_name)
        rc = th_local.cache_svc.write_to_server({
            gvars.REQ_REGISTER_ACTION: {
                gvars.REQ_ACTION_NAME: action_name,
                gvars.REQ_CLIENT_NAME: th_local.cl_name }})
        if not rc:
            ret = _report_error(-3, "client failed to write register action {}/{}".
                    format(th_local.cl_name, action_name))
    return c_int(ret)


def clib_touch_heartbeat(baction_name:bytes, binstance_id: bytes) -> c_int:
    _reset_error()
    ret = 0

    action_name = baction_name.decode("utf-8")
    instance_id = binstance_id.decode("utf-8")

    if not _is_initialized():
        ret = _report_error(-1, "touch_heartbeat: client not registered {}".format(action_name))

    elif action_name not in th_local.actions:
        ret = _report_error(-2, "Heartbeat from unregistered action {}".format(action_name))

    else:
        rc = th_local.cache_svc.write_to_server({
            gvars.REQ_HEARTBEAT: {
                gvars.REQ_CLIENT_NAME: th_local.cl_name,
                gvars.REQ_ACTION_NAME: action_name,
                gvars.REQ_INSTANCE_ID: instance_id }})
        if not rc:
            ret = _report_error(-3, "client failed to write heartbeat {}/{}/{}".
                    format(th_local.cl_name, action_name, instance_id))

    return c_int(ret)


def clib_read_action_request(timeout:int) -> bytes:
    _reset_error()
    data = ""
    if not _is_initialized():
        _report_error(-1, "read_action_request: client not registered")
        return "".encode("utf-8")

    ret, d = th_local.cache_svc.read_from_server(timeout)
    if not ret:
        if timeout == -1:
            _report_error(-2, "client failed to read action request {}".
                format(th_local.cl_name))
    else:
        data = json.dumps(d)
    return data.encode("utf-8")


def clib_write_action_response(resp: bytes) -> c_int:
    _reset_error()
    ret = 0

    if not _is_initialized():
        ret = _report_error(-1, "write_action_request: client not registered")

    else:
        rc = th_local.cache_svc.write_to_server(json.loads(resp.decode("utf-8")))
        if not rc:
            ret = _report_error(-3, "client failed to write response {}".
                    format(th_local.cl_name))
    return c_int(ret)



# Called by client - here the Plugin Process
#
def clib_poll_for_data(fds: POINTER(c_int), cnt: c_int,
        ready_fds: POINTER(c_int), ready_cnt: POINTER(c_int),
        err_fds: POINTER(c_int), err_cnt: POINTER(c_int),
        timeout: c_int) -> c_int:
    ret = 0
    if not _is_initialized():
        ret = _report_error("poll_for_data: client not registered")

    else:
        lfds = list(fds)
        r = _poll(lfds, timeout.value)

        ret = ready_cnt.value = len(r)
        err_cnt.value = 0

        for i in range(ret):
            ready_fds[i] = r[i]

    return c_int(ret)


