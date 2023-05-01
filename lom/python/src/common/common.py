#! /usr/bin/env python

import ctypes
import json
import os
import sys
import syslog
import time
from threading import current_thread
from typing import NamedTuple

import gvars

# python_proc overrides this path via args, if provided.
GLOBAL_RC_FILE = "/etc/LoM/global.rc.json"
_CT_PATH = os.path.dirname(os.path.abspath(__file__))

from enum import Enum

# *******************************
# Syspath updates.
# *******************************
#
def syspath_append(path:str):
    if path.endswith("/"):
        path = path[0:-1]

    if path not in sys.path:
        sys.path.append(path)



# *******************************
# Syslog related info
# *******************************
#
def syslog_init(proc_name:str):
    name = os.path.basename(sys.argv[0]) + "_" + proc_name
    syslog.openlog(name, syslog.LOG_PID)

_lvl_to_str = [
        "Emergency",
        "Alert",
        "Critical",
        "Error",
        "Warning",
        "Notice",
        "Informational",
        "Debug"
    ]


ct_log_level = syslog.LOG_ERR

def set_log_level(lvl:int):
    global ct_log_level

    ct_log_level = lvl


def _log_write(lvl: int, msg:str):
    if lvl <= ct_log_level:
        syslog.syslog(lvl, msg)
        print("{}:{}:{}: {}".format(current_thread().name, _lvl_to_str[lvl], 
            time.time(), msg))


def log_panic(msg:str):
    _log_write(syslog.LOG_CRIT, msg+" Exiting ...")
    os.exit(-1)

def log_error(msg:str):
    _log_write(syslog.LOG_ERR, msg)

def log_info(msg:str):
    _log_write(syslog.LOG_INFO, msg)

def log_warning(msg:str):
    _log_write(syslog.LOG_WARNING, msg)

def log_debug(msg:str):
    _log_write(syslog.LOG_DEBUG, msg)


def GetLoMRunMode():
    return GetLoMRunModeC()

def ist_test_mode() -> bool:
    return gvars.TEST_RUN

