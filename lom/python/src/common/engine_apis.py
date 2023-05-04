#! /usr/bin/env python3


from ctypes import *
from common import *
from enum import IntEnum, auto
import os

DLL_NAME = "cmn_c_lib.so"
DLL_PATH = os.getenv("LOM_LIB_PATH")
if not DLL_PATH:
    DLL_PATH = "/usr/share/lom/lib"

class LOM_LIB_FN_INDICES(IntEnum):
    LOM_LIB_FN_INIT         = auto()
    LOM_LIB_FN_RUN_MODE     = auto()
    LOM_LIB_FN_CFG_STR      = auto()
    LOM_LIB_FN_CFG_INT      = auto()
    LOM_LIB_FN_CFG_SEQ      = auto()
    LOM_LIB_FN_CFG_ACTION   = auto()
    LOM_LIB_FN_LIST_ACTIONS = auto()
    LOM_LIB_FN_CFG_PROC     = auto()
    LOM_LIB_FN_ENGINE_START = auto()
    LOM_LIB_FN_ENGINE_STOP  = auto()
    LOM_LIB_FN_REG_CLIENT   = auto()
    LOM_LIB_FN_DEREG_CLIENT = auto()
    LOM_LIB_FN_REG_ACTION   = auto()
    LOM_LIB_FN_DEREG_ACTION = auto()
    LOM_LIB_FN_RECV_REQ     = auto()
    LOM_LIB_FN_SEND_RES     = auto()
    LOM_LIB_FN_NOTIFY_HB    = auto()
    LOM_LIB_FN_CNT          = auto()

# Loaded lib & functions
lomLib = None

lomLibFunctions = {}  # dict[LOM_LIB_FN_INDICES-int, ctypes.CDLL.__init__.<locals>._FuncPtr]



def lom_lib_init():
    global lomLib

    if lomLib == None:
        try:
            lomLib = CDLL(os.path.join(DLL_PATH, DLL_NAME))
        except OSError as e:
            log_panic("Failed to load error:{}".format(e))


    lomLibFunctionsMetaData = {
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_INIT.value): {
                "fn": lomLib.InitConfigPathForC, "args": [c_char_p], "res": c_int },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_RUN_MODE.value): {
                "fn": lomLib.GetLoMRunModeC, "args": [], "res": c_int },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_STR.value): {
                "fn": lomLib.GetGlobalCfgStrC, "args": [ c_char_p ], "res": c_char_p },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_INT.value): {
                "fn": lomLib.GetGlobalCfgIntC, "args": [ c_char_p ], "res": c_int },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_SEQ.value): {
                "fn": lomLib.GetSequenceAsJsonC, "args": [ c_char_p ], "res":  c_char_p },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_ACTION.value): {
                "fn": lomLib.GetActionConfigAsJsonC, "args": [ c_char_p ], "res":  c_char_p },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_LIST_ACTIONS.value): {
                "fn": lomLib.GetActionsListAsJsonC, "args": [], "res":  c_char_p },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_PROC.value): {
                "fn": lomLib.GetProcsConfigC, "args": [ c_char_p ], "res":  c_char_p },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_ENGINE_START.value): {
                "fn": lomLib.EngineStartC, "args": [ c_char_p ], "res":  c_int },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_ENGINE_STOP.value): {
                "fn": lomLib.EngineStopC, "args": [], "res":  c_int },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_REG_CLIENT.value): {
                "fn": lomLib.RegisterClientC, "args": [ c_char_p ], "res":  c_char_p },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_DEREG_CLIENT.value): {
                "fn": lomLib.DeregisterClientC, "args": [], "res":  c_char_p },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_REG_ACTION.value): {
                "fn": lomLib.RegisterActionC, "args": [ c_char_p ], "res":  c_char_p },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_DEREG_ACTION.value): {
                "fn": lomLib.DeregisterActionC, "args": [ c_char_p ], "res":  c_char_p },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_RECV_REQ.value): {
                "fn": lomLib.RecvServerRequestC, "args": [], "res":  c_char_p },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_SEND_RES.value): {
                "fn": lomLib.SendServerResponseC, "args": [ c_char_p ], "res":  c_char_p },
            str(LOM_LIB_FN_INDICES.LOM_LIB_FN_NOTIFY_HB.value): {
                "fn": lomLib.NotifyHeartbeatC, "args": [ c_char_p, c_longlong ], "res":  c_char_p }
            }


    for k, v in lomLibFunctionsMetaData.items():
        fn = v["fn"]
        fn.argtypes = v["args"]
        fn.restype = v["res"]
        lomLibFunctions[int(k)] = fn


#
# Called once upon first import
#
lom_lib_init()

def call_lom_lib(id: LOM_LIB_FN_INDICES,  *args):
    if id not in lomLibFunctions:
        log_error("Failed to find id={} in mapped fns {}", id, list(lomLibFunctions.keys()))
        return -1

    fn = lomLibFunctions[id]
    argsCnt = len(args)

    if len(fn.argtypes) != argsCnt:
        log_error("fn for id={} require {} args, given {}".format(id, len(fn.argtypes), argsCnt))
        return -1

    updArgs = [None] * argsCnt
    for i in range(argsCnt):
        if fn.argtypes[i] == c_char_p:
            updArgs[i] = args[i].encode("utf-8")
        elif fn.argtypes[i] == c_int:
            updArgs[i] = c_int(args[i])
        elif fn.argtypes[i] == c_longlong:
            updArgs[i] = c_longlong(args[i])
        else:
            log_error("id:{} i:{} Unexpected arg type {}".format(id, i, fn.argtypes[i]))

    if argsCnt == 0:
        res = lomLibFunctions[id]()

    elif argsCnt == 1:
        res = lomLibFunctions[id](updArgs[0])

    elif argsCnt == 2:
        res = lomLibFunctions[id](updArgs[0], updArgs[1])

    else:
        log_error("Internal Error: No fn known to handle {} cnt args. id:{}".format(argsCnt, id))
        return -1

    if fn.restype == c_char_p:
        return res.decode("utf-8")

    if fn.restype == c_int:
        return res

    log_error("Internal Error: {} type not expected in return".format(fn.restype))
    return None



class testCode:
    cfgPath = "/tmp"

    cfgData = {
        "globals.conf.json": '\
{\
    "ENGINE_HB_INTERVAL_SECS" : 3,\
    "Foo": "Bar",\
    "val": 7,\
    "flag": true,\
    "lst": [ 6, 7, 8 ]\
}',
        "actions.conf.json": '{}',
        "bindings.conf.json": '{}',
        "procs.conf.json": '{}'
        }


    def __init__(self):
        for f, v in self.cfgData.items():
            with open(os.path.join(self.cfgPath, f), "w") as s:
                s.write(v)


    def globalCfgTest(self):
        ret = call_lom_lib(LOM_LIB_FN_INDICES.LOM_LIB_FN_INIT, self.cfgPath)
        print("LOM_LIB_FN_INDICES.LOM_LIB_FN_INIT ret={}".format(ret))
    
        ret = call_lom_lib(LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_STR, "Foo")
        print("LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_STR ret={}".format(ret))
    
        ret = call_lom_lib(LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_INT, "val")
        print("LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_INT ret={}".format(ret))

    

def test():
    tc = testCode()
    tc.globalCfgTest()


if __name__ == "__main__":
    test()
