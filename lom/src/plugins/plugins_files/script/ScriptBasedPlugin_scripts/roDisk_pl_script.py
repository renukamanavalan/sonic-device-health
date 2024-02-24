#! /usr/bin/env python3

# Inability to touch any file in /tmp implies disk is in RO state
# Inability to touch any file under /home implies SONiC patch has not mounted home
# into tmpfs/in-memory to make it RW.
# If /home is RO, no remote user can log in as it needs dir creation under home.
#
# This script just publishes current state.
#
import json
import os
import sys
import syslog

LogPrefix = "RoDiskChek"
TEST_FILE = "LoM_Test"
# real one: RO_CHECK_DIR = "/tmp"
# real one: BANDAID_CHECK_DIR = "/home"

# Use mock dirs so as to easily simulate.
RO_CHECK_DIR = "/usr/share/device_health/tmp"
BANDAID_CHECK_DIR = "/usr/share/device_health/home"

def isReadOnly(path:str) -> bool:
    if os.path.exists(path):
        fl = os.path.join(path, TEST_FILE)
        try:
            open(fl, "w+")
            return True
        except (FileNotFoundError, PermissionError, OSError) as e:
            syslog.syslog(syslog.LOG_INFO, "{LogPrefix}: Failed to open {fl} err:{e}")
    return False


def diskCheck():
    ret = {}
    ret["action"] = "DiskCheck"
    ret["ReadOnly"] = not isReadOnly(RO_CHECK_DIR)
    ret["MountedAsRW"] = isReadOnly(BANDAID_CHECK_DIR)
    sys.stdout.write(json.dumps(ret))


if __name__ == "__main__":
    diskCheck()

