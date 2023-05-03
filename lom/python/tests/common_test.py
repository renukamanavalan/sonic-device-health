import os

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


    def __init__(self):
        for f, v in self.cfgData.items():
            with open(os.path.join(self.cfgPath, f), "w") as s:
                s.write(v)



cfg = cfgInit()

