import os
import sys

sys.path.append("src/common")
import engine_apis

class cfgInit:
    cfgPath = "/tmp"

    cfgData = {
        "globals.conf.json": '\
{\
    "ENGINE_HB_INTERVAL_SECS" : 3,\
    "Foo_Bar": "Bar",\
    "val_7": 7,\
    "flag_true": true,\
    "flag_false": false,\
    "lst_6_7_8": [ 6, 7, 8 ]\
}',
        "actions.conf.json": '{}',
        "bindings.conf.json": '{}',
        "procs.conf.json": '{}'
        }


    def __init__(self, testMode: bool):
        for f, v in self.cfgData.items():
            with open(os.path.join(self.cfgPath, f), "w") as s:
                s.write(v)

        testModeFl = os.path.join(self.cfgPath, "LoMTestMode")
        if testMode:
            with open(testModeFl, "w") as s:
                s.write("")
        else:
            # Remove file if exists
            pathlib.Path(testModeFl).unlink(missing_ok = True)



cfg = None

def InitCfg(testMode: bool):
    c = cfgInit(testMode)
    ret = engine_apis.call_lom_lib(
            engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_INIT, c.cfgPath)
    if ret == 0:
        cfg = c
        return 0
    else:
        log_error("Failed to load config")
        return -1

