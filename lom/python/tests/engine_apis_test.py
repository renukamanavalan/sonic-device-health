import sys
from unittest.mock import MagicMock, patch

import pytest

from . import common_test

sys.path.append("src/common")
import engine_apis


class TestCfg(object):
    def testGlobal(self):
        # Create testmode file


        ret = common_test.InitCfg(True)
        assert ret == 0, f"lomLib.InitConfigPathForC failed ret={ret}"

        ret = engine_apis.call_lom_lib(
                engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_RUN_MODE)
        assert ret == 1, f"lomLib.lomLib.GetLoMRunModeC ret{ret} != 1"

        ret = engine_apis.call_lom_lib(
                engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_STR, "Foo_Bar")
        assert ret == "Bar", f"lomLib.lomLib.GetGlobalCfgStrC key=Foo_Bar ret{ret} != Bar"

        ret = engine_apis.call_lom_lib(
                engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_INT, "val_7")
        assert ret == 7, f"lomLib.lomLib.GetGlobalCfgIntC key=val_7 ret{ret} != 7"

        ret = engine_apis.call_lom_lib(
                engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_STR, "Non-existing")
        assert ret == "<nil>", f"lomLib.lomLib.GetGlobalCfgStrC key=Non-existing ret{ret} != <nil>"

        ret = engine_apis.call_lom_lib(
                engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_INT, "val_None")
        assert ret == 0, f"lomLib.lomLib.GetGlobalCfgIntC key=val_None ret{ret} != 0"


