package main

import (
    . "lomcommon"
    . "lomipc"
)

type TestClientData struct {
    ReqType     ReqDataType  /* Req type to call */
    Args        []string            /* Args needed for the call */
    Failed      bool                /* Expect to fail or succeed */
    ExpResp     interface{}         /* Differs per request */
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
var ActReqData = ActionRequestData { "Bar", "inst_1", "an_inst_0", "an_key",
        []ActionResponseData {
                { TEST_ACTION_NAME, "an_inst_0", "an_inst_0", "an_key", "res_anomaly", 0, ""},
                { "Foo-safety", "inst_0", "an_inst_0", "an_key", "res_foo_check", 2, "some failure"},
        } }

var ClTimeout = 2

var testData = []TestData {
            {   TestClientData { TypeRegClient, []string{TEST_CL_NAME },  false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeRegClient, TEST_CL_NAME, ClTimeout, MsgRegClient {} },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },
            {   TestClientData { TypeRegAction, []string{ TEST_ACTION_NAME },  true, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction { TEST_ACTION_NAME } },
                        LoMResponse { 1, "failed by design", MsgEmptyResp {} } } },
            {   TestClientData { TypeRegAction, []string{ TEST_ACTION_NAME },  false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction { TEST_ACTION_NAME } },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },
            {   TestClientData { TypeRecvActionRequest, []string{},  false, ActReqData },
                TestServerData { LoMRequest { TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRecvActionRequest{} },
                        LoMResponse { 0, "Succeeded", ActReqData } } },
            {   TestClientData { TypeDeregAction, []string{ TEST_ACTION_NAME },  false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeDeregAction, TEST_CL_NAME, ClTimeout, MsgDeregAction { TEST_ACTION_NAME } },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },
            {   TestClientData { TypeDeregClient,  []string{}, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeDeregClient, TEST_CL_NAME, ClTimeout, MsgDeregClient {} },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },
            {   TestClientData { TypeDeregClient,  []string{}, true, MsgEmptyResp{} },
                TestServerData { LoMRequest {}, LoMResponse {} } },
        }


func testClient(chRes chan interface{}, chComplete chan interface{}) {
    txClient := &ClientTx{nil, "", ClTimeout}

    for i := 0; i < len(testData); i++ {
        tdata := &testData[i]
        var err error
        var reqData *ActionRequestData = nil

        switch tdata.ReqType {
        case TypeRegClient:
            if len(tdata.Args) != 1 {
                LogPanic("client: tid:%d: Expect 1 args for register client len=%d", i, len(tdata.Args))
            }
            err = txClient.RegisterClient(tdata.Args[0])
        case TypeDeregClient:
            if len(tdata.Args) != 0 {
                LogPanic("client: tid:%d: Expect No args for register client len=%d", i, len(tdata.Args[1]))
            }
            err = txClient.DeregisterClient()
        case TypeRegAction:
            if len(tdata.Args) != 1 {
                LogPanic("client: tid:%d: Expect 1 args for register action len=%d", i, len(tdata.Args))
            }
            err = txClient.RegisterAction(tdata.Args[0])
        case TypeDeregAction:
            if len(tdata.Args) != 1 {
                LogPanic("client: tid:%d: Expect 1 args for deregister action len=%d", i, len(tdata.Args))
            }
            err = txClient.DeregisterAction(tdata.Args[0])
        case TypeRecvActionRequest:
            if len(tdata.Args) != 0 {
                 LogPanic("client: tid:%d: Expect No args for RecvActionRequest  len=%d", i, len(tdata.Args))
            }
            reqData, err = txClient.RecvActionRequest()
        default:
            LogPanic("client: tid:%d TODO - Not yet implemented (%d)", i, tdata.ReqType)
        }
        if (err != nil) != tdata.Failed {
            LogPanic("client: tid:%d type(%d/%s) err=%v failed=%v", i, tdata.ReqType,
                    ReqTypeToStr[tdata.ReqType], err, tdata.Failed)
        }

        p := tdata.ExpResp
        if reqData != nil {
            if expData, ok := p.(ActionRequestData); ok {
                if !reqData.Equal(&expData) {
                    LogPanic("Client: tid:%d ReqData (%v) != expData(%v)", i, *reqData, expData)
                }
            } else {
                LogPanic("Client: tid:%d Type mismatch Rcvd:(%T) exp(%T)",i, reqData, p)
            }
        } else if x, ok := p.(MsgEmptyResp); !ok {
            LogPanic("Client: tid:%d Received None. Exp:(%T)", i, x)
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

