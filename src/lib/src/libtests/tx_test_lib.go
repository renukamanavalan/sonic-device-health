package main

import (
    . "lomcommon"
    . "lomipc"
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

const TEST_CL_NAME = "Foo"
const TEST_ACTION_NAME = "Detect-0"

var ClTimeout = 2

var testData = []TestData {
            {   TestClientData { TypeRegClient, []string{TEST_CL_NAME },  false },
                TestServerData { LoMRequest { TypeRegClient, TEST_CL_NAME, ClTimeout, MsgRegClient {} },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },
            {   TestClientData { TypeRegAction, []string{ TEST_ACTION_NAME },  true },
                TestServerData { LoMRequest { TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction { TEST_ACTION_NAME } },
                        LoMResponse { 1, "failed by design", MsgEmptyResp {} } } },
            {   TestClientData { TypeRegAction, []string{ TEST_ACTION_NAME },  false },
                TestServerData { LoMRequest { TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction { TEST_ACTION_NAME } },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },
            {   TestClientData { TypeDeregAction, []string{ TEST_ACTION_NAME },  false },
                TestServerData { LoMRequest { TypeDeregAction, TEST_CL_NAME, ClTimeout, MsgDeregAction { TEST_ACTION_NAME } },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },
            {   TestClientData { TypeDeregClient,  []string{}, false },
                TestServerData { LoMRequest { TypeDeregClient, TEST_CL_NAME, ClTimeout, MsgDeregClient {} },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },
            {   TestClientData { TypeDeregClient,  []string{}, true },
                TestServerData { LoMRequest {}, LoMResponse {} } },
        }


func testClient(chRes chan interface{}, chComplete chan interface{}) {
    txClient := &ClientTx{nil, "", ClTimeout}

    for i := 0; i < len(testData); i++ {
        tdata := &testData[i]

        switch tdata.ReqType {
        case TypeRegClient:
            if len(tdata.Args) != 1 {
                LogPanic("client: tid:%d: Expect 1 args for register client len=%d", i, len(tdata.Args))
            }
            err := txClient.RegisterClient(tdata.Args[0])
            if (err != nil) != tdata.Failed {
                LogPanic("client: tid:%d err=%v failed=%v", i, err, tdata.Failed)
            }
        case TypeDeregClient:
            if len(tdata.Args) != 0 {
                LogPanic("client: tid:%d: Expect No args for register client", i, len(tdata.Args[1]))
            }
            err := txClient.DeregisterClient()
            if (err != nil) != tdata.Failed {
                LogPanic("client: tid:%d err=%v failed=%v", i, err, tdata.Failed)
            }
        case TypeRegAction:
            if len(tdata.Args) != 1 {
                LogPanic("client: tid:%d: Expect 1 args for register action len=%d", i, len(tdata.Args))
            }
            err := txClient.RegisterAction(tdata.Args[0])
            if (err != nil) != tdata.Failed {
                LogPanic("client: tid:%d err=%v failed=%v", i, err, tdata.Failed)
            }
        case TypeDeregAction:
            if len(tdata.Args) != 1 {
                LogPanic("client: tid:%d: Expect 1 args for deregister action len=%d", i, len(tdata.Args))
            }
            err := txClient.DeregisterAction(tdata.Args[0])
            if (err != nil) != tdata.Failed {
                LogPanic("client: tid:%d err=%v failed=%v", i, err, tdata.Failed)
            }
        default:
            LogPanic("client: tid:%d TODO - Not yet implemented (%d)", i, tdata.ReqType)
        }
        LogDebug("client: tid=%d succeeded", i)
        chRes <- struct {}{}
    }
    LogDebug("client: Complete")
    chComplete <- struct {}{}
}

const readTimeoutSeconds = 2

func main() {
    tx, err := ServerInit()
    if err != nil {
        LogPanic("Failed to init server")
    }
    chResult := make(chan interface{})
    chComplete := make(chan interface{})

    go testClient(chResult, chComplete)

    for i := 0; i < len(testData); i++ {
        if len(chComplete) != 0 {
            LogPanic("Server tid:%d But client complete", i)
        }

        tdata := &testData[i]
        LogDebug("Server: Running: tid=%d", i)

        if (tdata.Req != LoMRequest{}) {
            p := tx.ReadClientRequest(readTimeoutSeconds, chComplete)
            if p == nil {
                LogPanic("Server: tid:%d ReadClientRequest returned nil", i)
            }
            if (*p.Req != tdata.Req) {
                LogInfo("Server: tid:%d: Type(%d) Failed to match msg(%v) != exp(%v)",
                                    i, tdata.ReqType, *p.Req, tdata.Req)
            }
            /* Response to remote client -- done via clientTx */
            p.ChResponse <- &tdata.Res
        }
        /* Wait for client to complete */
        <- chResult
            
    }
    LogDebug("Server Complete. Waiting on client to complete...")
    <- chComplete
    LogDebug("SUCCEEDED")
}

