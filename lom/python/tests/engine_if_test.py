#! /usr/bin/env python3

import json
import socket
import threading

import sys
syspath_append("../src/plugin_manager")
import engine_if

"""
Sample strings:

    RegClient: ({"ReqType":1,"Client":"Foo","TimeoutSecs":0,"ReqData":{}})

    DeregClient: ({"ReqType":2,"Client":"Foo","TimeoutSecs":0,"ReqData":{}})

    RegAction: ({"ReqType":3,"Client":"Foo","TimeoutSecs":0,"ReqData":{"Action":"act-0"}})

    DeregAction: ({"ReqType":4,"Client":"Foo","TimeoutSecs":0,"ReqData":{"Action":"act-0"}})

    NotifyHB: ({"ReqType":7,"Client":"Foo","TimeoutSecs":0,"ReqData":{"Action":"act-0","Timestamp":789}})

    RecvReq: ({"ReqType":5,"Client":"Foo","TimeoutSecs":0,"ReqData":{}})

    SendResp: ({"ReqType":6,"Client":"Foo","TimeoutSecs":0,"ReqData":{"ReqType":1,"ResData":{"Action":"act-0","InstanceId":"Inst-id","AnomalyInstanceId":"an-id","AnomalyKey":"an-key","Response":"res..","ResultCode":0,"ResultStr":"good"}}})

    Empty Res: ({"ResultCode":0,"ResultStr":"all good","RespData":{}})

    Non-empty Res: ({"ResultCode":0,"ResultStr":"all good","RespData":{"ReqType":1,"ReqData":{"Action":"act-2","InstanceId":"Inst-02","AnomalyInstanceId":"an-02","AnomalyKey":"an-k02","Timeout":987,"Context":[{"Action":"act-0","InstanceId":"Inst-id","AnomalyInstanceId":"an-id","AnomalyKey":"an-key","Response":"res..","ResultCode":0,"ResultStr":"good"},{"Action":"act-1","InstanceId":"Inst-01","AnomalyInstanceId":"an-01","AnomalyKey":"an-k01","Response":"res01","ResultCode":0,"ResultStr":"go01"}]}}})
"""

test_cases = {}
test_run_id = None
tests_done = False

HOST = "127.0.0.1"  # Standard loopback interface address (localhost)
PORT = 65432  # Port to listen on (non-privileged ports are > 1023)
BUFSZ = 4096

def run_server():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind((HOST, PORT))
        s.listen()
        conn, addr = s.accept()
        with conn:
            print(f"Connected by {addr}")
            while not tests_done:
                te = test_cases[test_run_id]
                data = conn.recv(BUFSZ)
                if data != te["server_read"]:

                                                                                            break
                                                                                                    conn.sendall(data)


