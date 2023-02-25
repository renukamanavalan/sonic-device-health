package main

import (
    . "lomcommon"
    . "lomipc"
    "strconv"
)

type TestClientData struct {
    ReqType     ReqDataType  /* Req type to call */
    Args        []string            /* Args needed for the call */
    Failed      bool                /* Expect to fail or succeed */
}

type TestServerData struct {
    Req     LoMRequest
    Res     LoMResponse             /* LoMResponse to send back */
}

type TestData struct {
    TestClientData
    TestServerData
}

var testData = [1]TestData {
            {   TestClientData { TypeRegClient, []string{"Foo", "2"},  false },
                TestServerData { LoMRequest { TypeRegClient, "foo", 2, MsgRegClient {} },
                        LoMResponse { 0, "Succeeded", nil } } }}



func testClient(chRes chan interface{}) {
    txClient := &ClientTx{nil, ""}

    for i := 0; i < len(testData); i++ {
        tdata := &testData[i]

        switch tdata.ReqType {
        case TypeRegClient:
            if len(tdata.Args) != 2 {
                LogPanic("client: tid:%d: Expect 2 args for register client", i, len(tdata.Args[1]))
            }
            if tout, err := strconv.Atoi(tdata.Args[1]); err == nil {
                err := txClient.RegisterClient(tdata.Args[0], tout)
                if (err != nil) == tdata.Failed {
                    LogPanic("client: tid:%d err=%v failed=%v", i, err, tdata.Failed)
                }
            } else {
                LogPanic("client: tid:%d Expect int val (%s) for timeout", tdata.Args[1], i)
            }
        default:
            LogPanic("TODO - Not yet implemented (%d)", tdata.ReqType)
        }
    }
    chRes <- struct {}{}
}

const readTimeoutSeconds = 2

func main() {
    tx, err := ServerInit()
    if err != nil {
        LogPanic("Failed to init server")
    }
    chResult := make(chan interface{})

    go testClient(chResult)

    for i := 0; i < len(testData); i++ {
        tdata := &testData[i]

        p := tx.ReadClientRequest(readTimeoutSeconds)
        if p == nil {
            LogPanic("Server: tid:%d ReadClientRequest returned nil", i)
        }
        if (*p.Req != tdata.Req) {
            LogPanic("Server: tid:%d: Type(%d) Failed to match msg(%v) != exp(%v)",
                                i, tdata.ReqType, p.Req, tdata.Req)
        }
        p.ChResponse <- tdata.Res
    }

    LogDebug("SUCCEEDED")
}

