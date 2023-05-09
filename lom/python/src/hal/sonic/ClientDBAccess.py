#! /usr/bin/env python3

"""
    Each client creates a client instance and uses for all its DB requests.

    This client calls into DBMainServer which runs dedicated singletons as one
    for subscription and other for any DB access in dedicated threads.

    All client requests are routed to these singletons via Q maintained by these.
    Any response is given back via Q maintained by this caller.

"""

from dataclasses import dataclass
from enum import Enum, auto
from queue import Queue
import queue
import common
import DBServer

DBMainServer_t = DBServer.DBMainServer
DBName_t    = DBServer.DBName_t
TblName_t   = DBServer.TblName_t
Key_t       = DBServer.Key_t
DBData_t    = DBServer.DBData_t

DBSubsRes_t = DBServer.DBSubsRes_t
DBSubsReq_t = DBServer.DBSubsReq_t
DBAccessReq_t = DBServer.DBAccessReq_t


clients = {}     # Record of all active instances

class DBClientInstance:
    def __init__(self, cid: str, server: DBMainServer_t):
        self.cid = cid
        self.server = server
        self.qSubs = Queue(10)      # 10 to buffer outstanding responses.
                                    # Any Q overflow results in drop.
                                    # So client should drain this Q as fast as possible.
        self.qAccess = Queue(1)     # Queue for sending DBAccess response.
                                    # As DBAccess response is 1:1 with request no additional
                                    # buffer needed.

        # register as caller for future subscriptions
        self.server.GetSubscriber().UpdSubs(DBSubsReq_t(cid, "", "", False, self.qSubs))


    def __del__(self):
        # De register caller
        self.deregister()


    def isActive(self) -> bool:
        return bool(self.cid)


    def deregister(self):
        if self.isActive():
            self.server.GetSubscriber().UpdSubs(DBSubsReq_t(self.cid, "", "", True))
            clients.pop(self.cid)
            self.cid = ""
            self.server = None


    def AddSubs(self, db: DBName_t, tbl: TblName_t):
        if self.isActive():
            self.server.GetSubscriber().UpdSubs(DBSubsReq_t(self.cid, db, tbl))

    def DelSubs(self, db: DBName_t, tbl: TblName_t):
        if self.isActive():
            self.server.GetSubscriber().UpdSubs(DBSubsReq_t(self.cid, db, tbl, True))

    
    def readSubsData(self, timeout: int) -> DBSubsRes_t:
        if self.isActive():
            try:
                return self.qSubs.get(block=True, timeout=timeout)
            except queue.Empty:
                pass
        return None


    def GetDBEntry(self, db: DBName_t, tbl: TblName_t, key: str) -> DBSubsRes_t:
        if self.isActive:
            self.server.GetDbAccess().sendReq(
                    DBAccessReq_t(db, tbl, key, DBServer.OpStr_t.GET, self.qAccess))
            return self.qAccess.get()
        return None


    def GetDBKeys(self, db: DBName_t, tbl: TblName_t) -> DBSubsRes_t:
        if self.isActive:
            self.server.GetDbAccess().sendReq(
                    DBAccessReq_t(db, tbl, "", DBServer.OpStr_t.GET_KEYS, self.qAccess))
            return self.qAccess.get()
        return None


    def ModDBEntry(self, db: DBName_t, tbl: TblName_t, key: str, data: DBData_t):
        if self.isActive:
            self.server.GetDbAccess().sendReq(
                    DBAccessReq_t(db, tbl, key, DBServer.OpStr_t.MOD, self.qAccess, data))
            self.qAccess.get()      # Wait for completion


    def DelDBEntry(self, db: DBName_t, tbl: TblName_t, key: str):
        if self.isActive:
            self.server.GetDbAccess().sendReq(
                    DBAccessReq_t(db, tbl, key, DBServer.OpStr_t.DEL, self.qAccess))
            self.qAccess.get()      # Wait for completion




def GetDBClientInstance(cid: str, server: DBMainServer_t):
    if cid not in clients:
        clients[cid] = DBClientInstance(cid, server)
    return clients[cid]


def DropDBClientInstance(cid: str):
    if cid in clients:
        inst = clients[cid]
        # de-register explicitly as destructor may not get called if there is
        # reference leak. With de-register, the instance is invalidated.
        inst.deregister()
        clients.discard(cid)


