from dataclasses import dataclass
import sys
from unittest.mock import MagicMock, patch
from typing import Tuple

sys.path.append("src/common")
import common
sys.path.append("src/hal/sonic")
import ClientDBAccess
import DBServer
import time

def DB_TBL_Key(db: str, tbl: str):
    return f"{db}_{tbl}"


@dataclass
class testSubsData_t:
    DB:     str
    tbl:    str
    Data:   Tuple[str, str, Tuple]


testDBSubs = [
        testSubsData_t("DB_0", "TBL_0", ( "Key_0", "SET", (("val", "0"),))),
        testSubsData_t("DB_0", "TBL_0", ( "Key_0", "SET", (("val", "1"),))),
        testSubsData_t("DB_0", "TBL_0", ( "Key_1", "SET", (("val", "0"),))),
        testSubsData_t("DB_0", "TBL_0", ( "Key_1", "SET", (("val", "1"),))),
        testSubsData_t("DB_0", "TBL_1", ( "Key_0", "SET", (("val", "0"),))),
        testSubsData_t("DB_0", "TBL_1", ( "Key_0", "SET", (("val", "1"),))),
        testSubsData_t("DB_0", "TBL_1", ( "Key_1", "SET", (("val", "0"),))),
        testSubsData_t("DB_0", "TBL_1", ( "Key_1", "SET", (("val", "1"),))),
        testSubsData_t("DB_1", "TBL_0", ( "Key_0", "SET", (("val", "0"),))),
        testSubsData_t("DB_1", "TBL_0", ( "Key_0", "SET", (("val", "1"),))),
        testSubsData_t("DB_1", "TBL_0", ( "Key_1", "SET", (("val", "0"),))),
        testSubsData_t("DB_1", "TBL_0", ( "Key_1", "SET", (("val", "1"),))),
        testSubsData_t("DB_1", "TBL_1", ( "Key_0", "SET", (("val", "0"),))),
        testSubsData_t("DB_1", "TBL_1", ( "Key_0", "SET", (("val", "1"),))),
        testSubsData_t("DB_1", "TBL_1", ( "Key_1", "SET", (("val", "0"),))),
        testSubsData_t("DB_1", "TBL_1", ( "Key_1", "SET", (("val", "1"),)))
    ]


# A test run. Caches all info related to this test run
# Easy to clean/ reset before starting next
#
testRun = None


class TestRun:
    def __init__(self):
        self.testDBSubsIndex = 0
        self.db_conns = {}
        self.subscribers = set()
        self.selector = None
        self.tables = {}
        self.selectQ = queue.Queue()
        self.subsQ = queue.Queue()
        self.subsQ = queue.Queue()


class DBConnector:
    def __init__(self, db): 
        self.db = db

    def getDbName(self):
        return self.db

db_conns = {}
def DBConnector_side_effect(db, k):
    global subs_conns

    assert k == 0, f"Expect second param {k} == 0"
    assert db not in testRun.db_conns, f"Request again for db {db}"
    conn = DBConnector(db)
    testRun.db_conns[db] = conn
    return conn


class mock_sub:
    def __init__(self, conn, tbl):
        self.conn = conn
        self.db = conn.getDbName()
        self.tbl = tbl
        return


    def key(self):
        return DB_TBL_Keyself.db, self.tbl)


    def pop(self):
        empty = ("", "", "")
        if testDBSubsIndex >= len(testDBSubs):
            return empty

        e = testDBSubs[testDBSubsIndex]
        if (self.db != e["DB"]) or (self.tbl != e["Tbl"]):
            return empty

        testRun.subsQ.get()     # Wait for signal to release data from main loop
        testDBSubsIndex += 1
        return e["Data"] 


    def getDbConnector(self):
        return self.conn


def SubscriberStateTable_side_effect(conn, tbl):
    key = DB_TBL_Key(conn.getDbName(), tbl)
    assert key not in testRun.subscribers, f"Request again for sub {key}"
    s = mock_sub(conn, tbl)
    testRun.subscribers.add(s.key())
    return s


class mock_selector:
    ERROR = -1

    def __init__(self):
        self.selectables = set()
        # print("Mock Selector constructed")


    def addSelectable(self, sub):       # mock_sub
        assert sub.key not in self.selectables, f"Duplicate selectable {sub.key}"
        self.selectables.add(sub.key)
        return 0

    def removeSelectable(self, sub):       # mock_sub
        assert sub.key in self.selectables, f"Absent selectable {sub.key}"
        self.selectables.remove(sub.key)
        return 0


    def select(self, timeout):
        self.selectQ.get()          # Wait for signal from main loop
        return (0, None)


def select_side_effect():
    assert not testRun.selector f"Expect only one call for Select"
    testRun.selector = mock_selector()
    return testRun.selector


class Table:
    def __init__(self, dbConn, tbl):
        self.dbConn = dbConn
        self.tbl = tbl
        self.data = {}


    def get(self, key):
        if key in self.data:
            return (True, self.data[key])
        return (True, {})


    def set(self, key, items):
        if key not in self.data:
            self.data[key] = {}
        d = self.data[key]
        for (k, v) in items:
            d[k] = v

    
    def getKeys(self):
        return list(self.data.keys())


def table_side_effect(dbConn, tbl):
    key = DB_TBL_Key(dbConn.getDbName(), tbl)
    if key in testRun.tables:
        return testRun.tables[key]
    tbl = Table(dbConn, tbl)
    testRun.tables[key] = tbl
    return tbl


@dataclass
class subData_t:
    DB:     str
    tbl:    str


@dataclass
class clientData_t:
    cid:    str
    SubOp: []   # List of tuple (Add/Del, data)
    ReadSubs: [ subData_t ]

clientList = [
        clientData("cid-0",         # (0,0) & (0, 1) 
            [
                (True, subData_t("DB_1", "TBL_1")),     # Add
                (True, subData_t("DB_0", "TBL_0")),     # Add
                (False, subData_t("DB_1", "TBL_1")),    # Remove added
                (False, subData_t("DB_0", "TBL_1")),    # Remove not added
                (True, subData_t("DB_0", "TBL_1"))      # added
            ],
            [ subData_t("DB_0", "TBL_0"), subData_t("DB_0", "TBL_1") ]
        ),
        clientData("cid-1",        # (0, 1), (1,0)
            [
                (True, subData_t("DB_0", "TBL_1")),     # Add
                (True, subData_t("DB_1", "TBL_0")),     # Add
                (False, subData_t("DB_1", "TBL_1"))     # Remove not added
            ],
            [ subData_t("DB_0", "TBL_1"), subData_t("DB_1", "TBL_0") ]
        ),
        clientData("cid-2",     # (1,0) & (1,1)
            [
                (True, subData_t("DB_1", "TBL_0")),     # Add
                (True, subData_t("DB_1", "TBL_1")),     # Add
                (True, subData_t("DB_0", "TBL_1")),     # Add
                (False, subData_t("DB_0", "TBL_1"))     # Remove added
            ],
            [ subData_t("DB_1", "TBL_0"), subData_t("DB_1", "TBL_1") ]
        ),
        clientData("cid-3",     # all
            [
                (True, subData_t("DB_1", "TBL_1")),     # Add
                (True, subData_t("DB_0", "TBL_0")),     # Add
                (True, subData_t("DB_0", "TBL_1")),     # Add
                (True, subData_t("DB_0", "TBL_0")),     # Add
            ],
            [
                subData_t("DB_0", "TBL_0"),
                subData_t("DB_0", "TBL_1")
                subData_t("DB_1", "TBL_0"),
                subData_t("DB_1", "TBL_1")
            ]
        )
    ]


class ClientInst:
    def __init__(self, clientListIndex: int, server: DBMainServer):
        self.clData = clientList[clientListIndex]
        self.inst = GetDBClientInstance(clName, server)
        self.opIndex = 0
        self.keys = set()
        for k in self.clData.ReadSubs:
            self.keys.add(DB_TBL_Key(k.DB, k.tbl))


    def verifyInst(self):
        p = GetDBClientInstance(self.name, None)
        assert self.inst == p, f"exist inst{self.inst.cid()} != {p.cid()}"


    def doOp(self) -> bool:
        if self.opIndex >= len(self.clData.SubOp):
            return False
        op = self.clData.SubOp[self.opIndex]
        self.opIndex += 1

        if op[0]:
            # Add
            self.inst.AddSubs(op[1].DB, op[1].tbl)
        else:
            self.inst.DelSubs(op[1].DB, op[1].tbl)
        return True
        

    def verifyRead(self):
        assert testRun.testDBSubsIndex < len(testDBSubs), f"Test Error index > len"

        dbData = testDBSubs[testRun.testDBSubsIndex]
        match = DB_TBL_Key(dbData.DB, DB.tbl) in self.keys
        
        res = self.inst.readSubsData(0)
        assert match != (res != None), f"match={match} res={res}"

        if res != None:
            assert res.db == dbData.DB, f"SubsData: DB mismatch"
            assert res.tbl == dbData.tbl, f"SubsData: tbl mismatch"
            assert res.key == dbData.Data[0], f"SubsData: key mismatch"
            assert res.data == dict(dbData.items()), f"SubsData: {res.data} != {dbData}"
        
    


class TestDBClient(object):

    @patch("DBServer.swsscommon.DBConnector")
    @patch("DBServer.swsscommon.SubscriberStateTable")
    @patch("DBServer.swsscommon.Table")
    @patch("DBServer.swsscommon.Select")
    def test_client(self, mock_select, mock_tbl, mock_subs, mock_conn):
        mock_select.side_effect = select_side_effect
        mock_tbl.side_effect = table_side_effect
        mock_subs.side_effect = SubscriberStateTable_side_effect
        mock_conn.side_effect = DBConnector_side_effect

        testRun = TestRun()
        common.set_log_level(7)
        clients = []

        server = DBServer.DBMainServer()

        for c in range(len(clientList)):
            cl = ClientInst(c, server)
            clients.append(cl)

        for cl in clients:
            cl.verifyInst()

        ret = True
        while ret:
            for cl in clients:
                ret |= cl.doOp()

        while testRun.testDBSubsIndex < len(testDBSubs):
            self.selectQ.put(0)
            self.subsQ.put(0)
            time.sleep(3)   # Allow DB server thread to pop & push to client instances
            for cl in clients:
                cl.verifyRead()

        server.GetSubscriber().teriminateRun()

