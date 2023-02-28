package main

import (
    . "lomcommon"
    . "lomipc"
    "strconv"
)

type TestClientData struct {
    ReqType     ReqDataType  /* Req type to call */
    Args        []string            /* Args needed for the call */
    DataArgs    interface{}
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

var ActResData = ActionResponseData { "Foo", "Inst-0", "AN-Inst-0", "an-key", "some resp", 9, "Failure Data" }

var ClTimeout = 2

var testData = []TestData {
            {   TestClientData { TypeRegClient, []string{TEST_CL_NAME }, nil, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeRegClient, TEST_CL_NAME, ClTimeout, MsgRegClient {} },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },
            {   TestClientData { TypeRegAction, []string{ TEST_ACTION_NAME }, nil, true, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction { TEST_ACTION_NAME } },
                        LoMResponse { 1, "failed by design", MsgEmptyResp {} } } },
            {   TestClientData { TypeRegAction, []string{ TEST_ACTION_NAME }, nil, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction { TEST_ACTION_NAME } },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },
            {   TestClientData { TypeRecvServerRequest, []string{}, nil, false, ActReqData },
                TestServerData { LoMRequest { TypeRecvServerRequest, TEST_CL_NAME, ClTimeout, MsgRecvServerRequest{} },
                        LoMResponse { 0, "Succeeded", ActReqData } } },
            {   TestClientData { TypeSendServerResponse, []string{}, ActResData, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeSendServerResponse, TEST_CL_NAME, ClTimeout, ActResData },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp{} } } },
                        {   TestClientData { TypeNotifyActionHeartbeat, []string{ TEST_ACTION_NAME, "100" }, nil, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeNotifyActionHeartbeat, TEST_CL_NAME, ClTimeout,
                                            MsgNotifyHeartbeat { TEST_ACTION_NAME, 100 } },
                        LoMResponse { 0, "Good", MsgEmptyResp {} } } },
            {   TestClientData { TypeDeregAction, []string{ TEST_ACTION_NAME }, nil, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeDeregAction, TEST_CL_NAME, ClTimeout, MsgDeregAction { TEST_ACTION_NAME } },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },
            {   TestClientData { TypeDeregClient,  []string{}, nil, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeDeregClient, TEST_CL_NAME, ClTimeout, MsgDeregClient {} },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },
            {   TestClientData { TypeDeregClient,  []string{}, nil, true, MsgEmptyResp{} },
                TestServerData { LoMRequest {}, LoMResponse {} } },
        }

var testCount = len(testData)

func testClient(chRes chan interface{}, chComplete chan interface{}) {
    txClient := &ClientTx{nil, "", ClTimeout}

    for i := 0; i < testCount; i++ {
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
        case TypeRecvServerRequest:
            if len(tdata.Args) != 0 {
                 LogPanic("client: tid:%d: Expect No args for RecvServerRequest len=%d", i, len(tdata.Args))
            }
            reqData, err = txClient.RecvServerRequest()
        case TypeSendServerResponse:
            if len(tdata.Args) != 0 {
                 LogPanic("client: tid:%d: Expect No args for SendServerResponse len=%d", i, len(tdata.Args))
            }
            p := tdata.DataArgs
            res, ok := p.(ActionResponseData)
            if (!ok) {
                LogPanic("client: tid:%d: Expect ActionResponseData as DataArgs (%T)/(%v)", i, p, p)
            }
            err = txClient.SendServerResponse(&res)
        case TypeNotifyActionHeartbeat:
            if len(tdata.Args) != 2 {
                LogPanic("client: tid:%d: Expect 2 args for register action len=%d", i, len(tdata.Args))
            }
            t, e := strconv.ParseInt(tdata.Args[1], 10, 64)
            if e != nil {
                LogPanic("client: tid:%d: Expect int64 val as second arg (%v)", i, tdata.Args[1])
            }
            err = txClient.NotifyHeartbeat(tdata.Args[0], EpochSecs(t))
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

    for i := 0; i < testCount; i++ {
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
                LogPanic("Server: tid:%d: Type(%d) Failed to match msg(%v) != exp(%v)",
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

