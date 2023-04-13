#! /usr/bin/env python3
import threading
import sys
sys.path.append("../hal/sonic")
sys.path.append("../common")

import DBServer
import ClientDBAccess
from common import *

db = DBServer.DBMainServer()

print("db = {}".format(db))

threads = threading.enumerate()
i=0
for thread in threads:
    print("{}:thread {} is active".format(i, thread.name))
    i += 1

clInst = ClientDBAccess.GetDBClientInstance("test-0", db)

ret = clInst.AddSubs("CONFIG_DB", "YYY")
print("AddSubs ret={}".format(ret))

while True:
    ret = clInst.readSubsData(None)
    print("sub o/p: ret={}".format(ret))
    if "quit" in ret.data:
        break

ret = clInst.GetDBKeys("CONFIG_DB", "YYY")
print("keys(YYY) = ({})".format(ret))

db.GetSubscriber().teriminateRun()
db.GetDbAccess().teriminateRun()

db.GetSubscriber

input("type in enter to end")
