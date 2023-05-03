import sys
from unittest.mock import MagicMock, patch

import pytest

from . import common_test

sys.path.append("src/common")
import engine_apis

EXP_SEQ_BIND_0 = '{"SequenceName":"bind-0","Timeout":2,"Priority":0,"Actions":[{"Name":"Detect-0","Mandatory":false,"Timeout":0,"Sequence":0},{"Name":"Safety-chk-0","Mandatory":false,"Timeout":1,"Sequence":1},{"Name":"Mitigate-0","Mandatory":false,"Timeout":6,"Sequence":2}]}'

EXP_ACT_DETECT_0 = '{"Name":"Detect-0","Type":"","Timeout":0,"HeartbeatInt":0,"Disable":false,"Mimic":false,"ActionKnobs":""}'

EXP_ACT_LIST = '{"Detect-0":{"IsAnomaly":true},"Detect-1":{"IsAnomaly":true},"Mitigate-0":{"IsAnomaly":false},"Mitigate-1":{"IsAnomaly":false},"Safety-chk-0":{"IsAnomaly":false},"Safety-chk-1":{"IsAnomaly":false}}'

EXP_PROC_0 = '{"Detect-0":{"name":"Detect-0","version":"00.01.1","path":" /path/"},"Detect-1":{"name":"Detect-1","version":"02.00.1","path":" /path/"},"Safety-chk-0":{"name":"Safety-chk-0","version":"02.00.1","path":" /path/"}}'

testCfgList = [
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_STR, 
            "arg": "Foo_Bar",
            "ret": "Bar",
            "msg": "Get globl cfg Str"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_STR, 
            "arg": "non-existing",
            "ret": "<nil>",
            "msg": "Get globl cfg Str non-exist"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_INT, 
            "arg": "val_7",
            "ret": 7,
            "msg": "Get globl cfg Int"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_INT, 
            "arg": "non-existing",
            "ret": 0,
            "msg": "Get globl cfg Int non-exist"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_SEQ, 
            "arg": "Detect-0",
            "ret": EXP_SEQ_BIND_0,
            "msg": "Get cfg Seq"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_SEQ, 
            "arg": "non-exist",
            "ret": "",
            "msg": "Get cfg Seq non-exist"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_ACTION, 
            "arg": "Detect-0",
            "ret": EXP_ACT_DETECT_0,
            "msg": "Get cfg Action"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_ACTION, 
            "arg": "non-exist",
            "ret": "",
            "msg": "Get cfg Action non-exist"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_LIST_ACTIONS, 
            "arg": None,
            "ret": EXP_ACT_LIST,
            "msg": "Get Actions list"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_PROC, 
            "arg": "proc_0",
            "ret": EXP_PROC_0,
            "msg": "Get proc config"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_PROC, 
            "arg": "non-exist",
            "ret": "",
            "msg": "Get proc config for non-exist"
        }
    ]

class TestCfg(object):
    def testGlobal(self):
        # Create testmode file

        ret = common_test.InitCfg(True)
        assert ret == 0, f"lomLib.InitConfigPathForC failed ret={ret}"

        ret = engine_apis.call_lom_lib(
                engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_RUN_MODE)
        assert ret == 1, f"lomLib.lomLib.GetLoMRunModeC ret{ret} != 1"

        ret = common_test.InitCfg(False)
        assert ret == 0, f"lomLib.InitConfigPathForC failed ret={ret}"

        ret = engine_apis.call_lom_lib(
                engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_RUN_MODE)
        assert ret == 2, f"lomLib.lomLib.GetLoMRunModeC ret{ret} != 2"

        id = 0
        for tc in testCfgList:
            if tc["arg"] != None:
                ret = engine_apis.call_lom_lib(tc["id"], tc["arg"])
            else:
                ret = engine_apis.call_lom_lib(tc["id"])

            assert ret == tc["ret"], f'id:{id} {tc["msg"]}'
            id += 1



