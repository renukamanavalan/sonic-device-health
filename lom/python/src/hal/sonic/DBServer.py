#! /usr/bin/env python3

"""
    Has a singleton instance to handle all subscribe requests
    Has another singleton instance to handle all DB access requests Get/Set/Delete

    These instances are created by Main server and given to requesting clients.
    Each instance runs its main method ("run") in a dedicated thread

    Clients send their requests via Queue of these singleton service instances.
    The dedicated thread reads requests & process and send response back via
    Queue embedded in the request itself.

    As subscriber main thread sleeps forever on select call looking for DB updates
    create a fake table and send update to this table, each time a request is added
    to the Q to alert main thread about the pending req. This fake table alert will 
    make select return and facilitates looking at Q

"""

from collections import defaultdict
from dataclasses import dataclass
from enum import Enum, auto
import queue
from queue import Queue
from swsscommon import swsscommon
from threading import Thread

import sys
sys.path.append("../../common")
from common import *

mainServer = None

DBName_t    = str
TblName_t   = str
Key_t       = str
DBData_t    = {}    # Dict of key & val; it is none for non-exist

Suscriber_t = swsscommon.SubscriberStateTable

caller_id_t = str

# Supported DB operations
class OpStr_t(Enum):
    GET         = "get"
    GET_KEYS    = "getKeys"
    MOD         = "mod"
    DEL         = "del"


# Subscribe request
@dataclass
class DBSubsReq_t:
    cid:    caller_id_t
    db:     DBName_t            # Could be empty when registering caller only.
    tbl:    TblName_t
    drop:   bool = False        # False - Add; True - Remove this subscription
    q:      queue.Queue = None  # Queue to send response DBSubsRes_t. Reqd for first req.


@dataclass
class DBSubsRes_t:
    db:     DBName_t
    tbl:    TblName_t
    key:    str
    data:   DBData_t = None    # None for delete /non-existing
    keys:   [str] = None       # List of keys for get keys

@dataclass
class DBAccessReq_t:
    db:     DBName_t
    tbl:    TblName_t
    key:    str
    op:     OpStr_t
    q:      Queue
    dict:   {} = None   # Dict of key/val. None for delete op

MAX_PEND_SUBS_RES_CNT = 100

# Put response in Q if not full.
#
def putResInQ(q: Queue, res: DBSubsRes_t):
    failed = False
    if not q.full():
        try:
            q.put_nowait(res)
        except queue.Full:
            failed = True
    else:
        failed = True

    if failed:
        log_error("Failed to put response len({})".format(len(q)))


"""
"""

"""
    Caller info
    Every caller is uniquely identified by ID
    All subs responses for a caller goes via same Q. Hence one Q per caller.
    Keeps the list of subscription paths tracked by DB vs set of tables

"""

class Caller_t:
    def __init__(self, cid: str, q: Queue):
        self.cid = cid
        self.q = q
        self.lst = defaultdict(set)  # DB name vs set of tables


    def AddSubs(self, db: DBName_t, tbl: TblName_t):
       self.lst[db].add(tbl)

    def delSubs(self, db: DBName_t, tbl: TblName_t):
       self.lst[db].discard(tbl)
       if len(lst[db]) == 0:
           self.lst.discard(db)


"""
    SWSS common subscriber associated with one or more callers
"""
class SubsInfo_t:
    def __init__(self, sub: Suscriber_t, cid: str):
        self.sub = sub
        self.callers = set([cid])



"""
    DB Subscriber

    A single instance created by DBMainServer to handle all subscriptions
    All DB Subcriptions happens via this instance only.
    Init calls run method in a dedicated thread.
    A fake temp table is created for alerting main thread
    for any additions from caller threads.

    All clients send their subscribe requests to this main thread via queue
    Alert main thread via updating fake table (updSubs)

    Main loop does the subscription, when first request arrives for a
    a table in a DB, it creates  swsscommon.SubscriberStateTable (Subscriber_t),
    creates SubsInfo_t and add this caller's id to its list. On future requests
    for same table, it just adds the id of caller to SubsInfo_t instance.

    Each SubsInfo_t has a subscriber added to swsscommon::selector as selectable

    The dedicated thread of this instance that runs "run" method, waits on select
    and passes received DB update to all callers as recorded in corresponding
    SubsInfo_t. The response is sent via Q in caller's object.
    
"""

class DBSubscriber:

    def __init__(self):
        self.ReqQ = queue.Queue()   # For all subscribe requests

        self.testCallerId = "__Internal__"
        self.testTblName = "LoMInternal"
        self.testDBName = "STATE_DB"

        self.selector = swsscommon.Select()
        self.DBConn = {}

        self.callers = {}               # caller cId vs caller obj

        self.subscribers = {}           # dict [DB;TbleName] = SubsInfo_t

        self.terminate = False

        self.GetSubscriber(self.testDBName, self.testTblName, self.testCallerId)
        self.tid = Thread(target = self.run)
        self.tid.start()


    # A way to terminate the thread. Set flag & alert the thread
    def teriminateRun(self):
        # Set flag and send dummy req
        self.terminate = True
        self.UpdSubs(DBSubsReq_t("","", "", True, None))
        log_debug("Waiting for run thread to join")
        self.tid.join()

    # DBClient intances send request to main thread via Q and alert the thread.
    # Register subscriptions with cid.
    #
    def UpdSubs(self, req: DBSubsReq_t):
        # Write into Q & alert.
        self.ReqQ.put(req)
        swsscommon.Table(self.GetDBConn(self.testDBName), self.testTblName).set("alert", [('fo0', 'bar')])


    # internal key used in tracking subscribers, which is 1 : 1 with (db, tbl) tuple.
    def getSubsDictKey(self, db : DBName_t, tbl: TblName_t):
        return ";".join([db, tbl])


    # Get & cache DB connection
    def GetDBConn(self, db_name: DBName_t):
        try:
            if db_name not in self.DBConn:
                self.DBConn[db_name] = swsscommon.DBConnector(db_name, 0)
            return self.DBConn[db_name]
        except Exception as err:
            log_error("Failed to get DBConn for ({})".format(db_name))
            return None

    # Get & cache subscribers
    # Single subscriber shared among multiple callers.
    #
    def GetSubscriber(self, db: DBName_t, tbl: TblName_t, cid: str) -> int:
        if not cid:
            log_error("require cid")
            return -1
        key = self.getSubsDictKey(db, tbl)
        if key not in self.subscribers:
            conn = self.GetDBConn(db)
            if conn == None:
                return -1
            sub = swsscommon.SubscriberStateTable(conn, tbl)
            self.selector.addSelectable(sub)
            self.subscribers[key] = SubsInfo_t(sub, cid)
        else:
            self.subscribers[key].callers.add(cid)
        return 0


    # Drop the caller
    # If last caller, drop subscriber
    def DropSubscriber(self, db: DBName_t, tbl: TblName_t, cid: str):
        key = self.getSubsDictKey(db, tbl)
        if key not in self.subscribers:
            return

        if not cid:
            return None

        info = self.subscribers[key]
        info.callers.discard(cid)

        if len(info.callers) == 0:
            self.selector.removeSelectable(sub)
            self.subscribers.discard(key)


    # Process AddSubs req
    # First req for caller may just come with caller-id & Q
    # Subsequent reqs may not carry Q and if carry must match the original Q
    # registered for this caller.
    # Based on the state, this may create a new subscriber or tag to existing.
    #
    def AddReq(self, req: DBSubsReq_t) -> int:
        # Validate
        if req.cid not in self.callers:
            if req.q == None:
                log_error("Missing q for new caller {}".format(req))
                return -1
            self.callers[req.cid] = Caller_t(req.cid, req.q)
        elif not req.db or not req.tbl:
            # Subsequent request must have DB & Tbl
            log_error("Missing Db/tbl in req {}".format(req))
            return -1

        caller = self.callers[req.cid]
        if (req.q != None) and (req.q != caller.q):
            log_error("Caller cId switching Q. De-register caller first.{}".format(req))
            return -1

        if not req.db:
            # caller registration only
            return 0

        # get subscriber if not already one
        ret = self.GetSubscriber(req.db, req.tbl, req.cid)

        if ret == 0:
            # Add to caller info
            caller.AddSubs(req.db, req.tbl)

        return ret


    # Drop a subs path (db & tbl non empty) or drop entire caller.
    def DropReq(self, req: DBSubsReq_t):
        if req.cid not in self.callers:
            return
        if req.db and not req.tbl:
            log_error("Drop require both db & tbl or none {}".format(req))
            return

        caller = self.callers[req.cid]
        for db, lst in caller.lst.items():
            if (not req.db) or (req.db == db):
                for tbl in lst:
                    if (not req.tbl) or (req.tbl == tbl):
                        self.DropSubscriber(db, tbl, req.cid)
                    if req.tbl:
                        caller.delSubs(db, tbl)
        if not req.db:
            # Dropping entire caller
            self.callers.discard(req.cid)


    # Process the req Q in no wait mode
    # Per doc, exception could raise even when empty return false.
    #
    def processReq(self):
        while not self.ReqQ.empty():
            try:
                req = self.ReqQ.get_nowait()
                if self.terminate:
                    return

                if not req.drop:
                    self.AddReq(req)
                else:
                    self.DropReq(req)
            except queue.Empty:
                break



    # Run forever until terminate requested
    # In each loop look for out standing requests for add/remove subs
    # Hence add/subs make a dummy update to alert the thread
    #
    def run(self):
        while not self.terminate:
            do_refresh = False

            state, _ = self.selector.select(-1)
            log_debug("DROP: select ...")
            if state == self.selector.ERROR:
                log_error("Subscriber select loop failed. Aborting")
            else:
                for _, subscriber in self.subscribers.items():
                    sub = subscriber.sub
                    key, op, fvs = sub.pop()
                    if not key:
                        continue
                    tbl = sub.getTableName()
                    db = sub.getDbConnector().getDbName()

                    if (db == self.testDBName) and (tbl == self.testTblName):
                        do_refresh = True   # Handle subscriber update requests
                        continue

                    res = DBSubsRes_t(db, tbl, key, dict(fvs))
                    for cid in subscriber.callers:
                        if cid != self.testCallerId:
                            if cid in self.callers:
                                putResInQ(self.callers[cid].q, res)
                            else:
                                log_error("Internal ERROR: Failed to find cid({}) in callers{}".
                                        format(cid, self.callers))

            if do_refresh:
                self.processReq()



# Meant for DB ops only.
# A singleton instance is created at the start.
# clients send their request via Q maintained by this instance.
# Runs in a dedicated thread and watch for requests from Q
# Executes each request and send response back via Q embedded in the request
#
class DBAccess:

    def __init__(self):
        self.ReqQ = queue.Queue()
        self.DBConn = {}
        self.terminate = False
        self.tid = Thread(target = self.run)
        self.tid.start()
        log_debug("DROP: ... DBAccess constructed")


    # A way to terminate the thread. Set flag & alert the thread
    def teriminateRun(self):
        # Set flag and send dummy req
        self.terminate = True
        self.sendReq(DBAccessReq_t("","","","", None))
        log_debug("Waiting for run thread to join")
        self.tid.join()

    def GetDBConn(self, db_name: DBName_t):
        if db_name not in self.DBConn:
            self.DBConn[db_name] = swsscommon.DBConnector(db_name, 0)
        return self.DBConn[db_name]

    # Send request and alert the thread.
    def sendReq(self, req: DBAccessReq_t):
        # Write into Q & alert.
        self.ReqQ.put(req)

    def run(self):
        while not self.terminate:
            log_debug("DROP: read stArt ...")
            req = self.ReqQ.get()
            if self.terminate:
                break
            conn = self.GetDBConn(req.db)
            tbl = swsscommon.Table(conn, req.tbl)

            if req.op == OpStr_t.GET:
                res = DBSubsRes_t(req.db, req.tbl, req.key, dict(tbl.get(req.key)[1]))
                putResInQ(req.q, res)
            elif req.op == OpStr_t.GET_KEYS:
                keys = tbl.getKeys()
                res = DBSubsRes_t(req.db, req.tbl, req.key, None, keys)
                putResInQ(req.q, res)
            elif req.op == OpStr_t.MOD:
                tbl.set(req.key, list(req.data.items()))
                putResInQ(req.q, 0)
            elif req.op == OpStr_t.DEL:
                conn.delete(req.tbl, req.key)
                putResInQ(req.q, 0)
            elif self.terminate:
                break
            else:
                log_error("Unknown op:{} req:{}".format(req.op, req))
                putResInQ(req.q, 0)


class DBMainServer:
    def __init__(self):
        self.dbSubscriber = DBSubscriber()
        self.dbAccess = DBAccess()
        log_debug("DROP: ... DBMainServer constructed")

    def GetSubscriber(self) -> DBSubscriber: 
        return self.dbSubscriber

    def GetDbAccess(self) -> DBSubscriber: 
        return self.dbAccess


