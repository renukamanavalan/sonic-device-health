package txlib_test

/*
 * Info
 *
 * To run test
 * localadmin@remanava-dev-1:~/source/fork/Device-Health/go-main/src/lib$ clear; GOPATH=$(pwd) go test -coverprofile=coverprofile.out  -coverpkg lomipc,lomcommon -covermode=atomic txlib_test
 *
 * To create HTML page
 * localadmin@remanava-dev-1:~/source/fork/Device-Health/go-main/src/lib$ GOPATH=$(pwd) go tool cover -html=coverprofile.out -o /tmp/coverage.html
 *
 * Edge shows uncovered lines by Red color
 *
 * Current
 *    ok      txlib_test      1.017s  coverage: 98.5% of statements in lomipc, lomcommon
 *
 * ./build.sh v <-- to run tests
 */

import (
    "errors"
    "io"
    "log/syslog"
    "net/rpc"
    . "lomcommon"
    . "lomipc"
    "strconv"
    "testing"
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
var ActReqData = ServerRequestData { TypeServerRequestAction,
        ActionRequestData { "Bar", "inst_1", "an_inst_0", "an_key",
            []ActionResponseData {
                    { TEST_ACTION_NAME, "an_inst_0", "an_inst_0", "an_key", "res_anomaly", 0, ""},
                    { "Foo-safety", "inst_0", "an_inst_0", "an_key", "res_foo_check", 2, "some failure"},
        } } }

var ActResData = ServerResponseData { TypeServerRequestAction, ActionResponseData {
                "Foo", "Inst-0", "AN-Inst-0", "an-key", "some resp", 9, "Failure Data" } }

var ShutReqData = ServerRequestData { TypeServerRequestShutdown, ShutdownRequestData{} }

var ClTimeout = 2

var testData = []TestData {
            // Reg Client
            {   TestClientData { TypeRegClient, []string{TEST_CL_NAME }, nil, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeRegClient, TEST_CL_NAME, ClTimeout, MsgRegClient {} },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },

            // Reg Action - test failure
            {   TestClientData { TypeRegAction, []string{ TEST_ACTION_NAME }, nil, true, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction { TEST_ACTION_NAME } },
                        LoMResponse { 1, "failed by design", MsgEmptyResp {} } } },

            // Reg Action - test failure
            {   TestClientData { TypeRegAction, []string{ TEST_ACTION_NAME }, nil, true, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction { TEST_ACTION_NAME } },
                        LoMResponse { 1, "SKIP", MsgEmptyResp {} } } },

            // Register action
            {   TestClientData { TypeRegAction, []string{ TEST_ACTION_NAME }, nil, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction { TEST_ACTION_NAME } },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },

            // Request for request and server sends Action request
            {   TestClientData { TypeRecvServerRequest, []string{}, nil, false, ActReqData },
                TestServerData { LoMRequest { TypeRecvServerRequest, TEST_CL_NAME, ClTimeout, MsgRecvServerRequest{} },
                        LoMResponse { 0, "Succeeded", ActReqData } } },

            // Send Action response to server
            {   TestClientData { TypeSendServerResponse, []string{}, ActResData, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeSendServerResponse, TEST_CL_NAME, ClTimeout, ActResData },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp{} } } },

            // Send Action heartbeat to server
            {   TestClientData { TypeNotifyActionHeartbeat, []string{ TEST_ACTION_NAME, "100" }, nil, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeNotifyActionHeartbeat, TEST_CL_NAME, ClTimeout,
                                            MsgNotifyHeartbeat { TEST_ACTION_NAME, 100 } },
                        LoMResponse { 0, "Good", MsgEmptyResp {} } } },

            // Request for request and server sends shutdown request
            {   TestClientData { TypeRecvServerRequest, []string{}, nil, false, ShutReqData },
                TestServerData { LoMRequest { TypeRecvServerRequest, TEST_CL_NAME, ClTimeout, MsgRecvServerRequest{} },
                        LoMResponse { 0, "Succeeded", ShutReqData } } },

            // Send Dereg action
            {   TestClientData { TypeDeregAction, []string{ TEST_ACTION_NAME }, nil, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeDeregAction, TEST_CL_NAME, ClTimeout, MsgDeregAction { TEST_ACTION_NAME } },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },

            // Send Dereg client
            {   TestClientData { TypeDeregClient,  []string{}, nil, false, MsgEmptyResp{} },
                TestServerData { LoMRequest { TypeDeregClient, TEST_CL_NAME, ClTimeout, MsgDeregClient {} },
                        LoMResponse { 0, "Succeeded", MsgEmptyResp {} } } },

            // Send duplicate Dereg client which is expected to fail
            {   TestClientData { TypeDeregClient,  []string{}, nil, true, MsgEmptyResp{} },
                TestServerData { LoMRequest {}, LoMResponse {} } },
        }

var testCount = len(testData)

func testClient(chRes chan interface{}, chComplete chan interface{}) {
    txClient := &ClientTx{nil, "", ClTimeout}

    for i := 0; i < testCount; i++ {
        tdata := &testData[i]
        var err error
        var reqData *ServerRequestData = nil

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
            res, ok := p.(ServerResponseData)
            if (!ok) {
                LogPanic("client: tid:%d: Expect ServerResponseData as DataArgs (%T)/(%v)", i, p, p)
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
            if expData, ok := p.(ServerRequestData); ok {
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

func TestMain(t *testing.T) {
    tx, err := ServerInit()
    if err != nil {
        t.Errorf("Failed to init server")
    }
    chResult := make(chan interface{})
    chComplete := make(chan interface{})

    go testClient(chResult, chComplete)

    for i := 0; i < testCount; i++ {
        if len(chComplete) != 0 {
            t.Errorf("Server tid:%d But client complete", i)
        }

        tdata := &testData[i]
        LogDebug("Server: Running: tid=%d", i)

        if (tdata.Req != LoMRequest{}) {
            p, _ := tx.ReadClientRequest(readTimeoutSeconds, chComplete)
            if p == nil {
                t.Errorf("Server: tid:%d ReadClientRequest returned nil", i)
            }
            if (*p.Req != tdata.Req) {
                t.Errorf("Server: tid:%d: Type(%d) Failed to match msg(%v) != exp(%v)",
                                    i, tdata.ReqType, *p.Req, tdata.Req)
            }
            /* Response to remote client -- done via clientTx */
            if tdata.Res.ResultStr == "SKIP" {
                p.ChResponse <- struct{}{}
            } else {
                p.ChResponse <- &tdata.Res
            }
        }
        /* Wait for client to complete */
        <- chResult
            
    }
    LogDebug("Server Complete. Waiting on client to complete...")
    <- chComplete
    LogDebug("SUCCEEDED")
}


func TestClientFail(t *testing.T) {
    txClient := &ClientTx{nil, "", ClTimeout}
    {
        retE := errors.New("rerer")
        retC := errors.New("irerrwe")
        resCode := -1
        
        /* Save & override */
        dial := RPCDialHttp
        RPCDialHttp = func(s1 string, s2 string) (*rpc.Client, error) {
            return nil, retE
        }

        clCall := ClientCall
        ClientCall = func(tx *ClientTx, serviceMethod string, args any, reply any) error {
            if retC != nil {
                return retC
            }
            x, ok := reply.(*LoMResponse)
            if !ok {
                t.Errorf("Cient call reply not map to LomResponse (%T)", x)
            }
            x.ResultCode = resCode
            x.RespData = struct{}{}
            return nil
        }

        {
            err := txClient.RegisterClient("")
            if (err != retE) {
                t.Errorf("Failed to fail HTTP call")
            }

        }

        /* Don't fail HTTP */
        retE = nil
        {
            if err := txClient.RegisterClient(""); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
            if err := txClient.DeregisterClient(); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
            if err := txClient.RegisterAction(""); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
            if err := txClient.DeregisterAction(""); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
            if _, err := txClient.RecvServerRequest(); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
            d := ServerResponseData{}
            if err := txClient.SendServerResponse(&d); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
            if err := txClient.NotifyHeartbeat("", 0); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
        }
        
        /* Don't fail call, but return non zero result */
        retC = nil
        {
            if err := txClient.RegisterClient(""); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
            if err := txClient.DeregisterClient(); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
            if err := txClient.RegisterAction(""); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
            if err := txClient.DeregisterAction(""); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
            if _, err := txClient.RecvServerRequest(); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
            d := ServerResponseData{}
            if err := txClient.SendServerResponse(&d); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
            if err := txClient.NotifyHeartbeat("", 0); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
        }

        /* Fail in respData */
        resCode = 0
        {
            if err := txClient.RegisterClient(""); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
            if err := txClient.DeregisterClient(); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
            if err := txClient.RegisterAction(""); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
            if err := txClient.DeregisterAction(""); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
            if _, err := txClient.RecvServerRequest(); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
            d := ServerResponseData{}
            if err := txClient.SendServerResponse(&d); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
            if err := txClient.NotifyHeartbeat("", 0); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
        }

        /* Restore overrides */
        RPCDialHttp = dial
        ClientCall = clCall
    }
}

func TestServerFail(t *testing.T) {
    {
        p1 := []ActionResponseData {{}, {} }
        p2 := []ActionResponseData {{} }

        if false != SlicesComp(p1, p2) {
            t.Errorf("SlicesComp Failed to fail")
        }

        p2 = []ActionResponseData{{}, {}}
        p2[0].Action = "foo"
        if false != SlicesComp(p1, p2) {
            t.Errorf("SlicesComp same len Failed to fail")
        }
    }
    {
        s1 := &ServerRequestData { TypeServerRequestAction, struct{}{} }
        s2 := &ServerRequestData { TypeServerRequestShutdown, 
                    ActionRequestData {"foo", "", "", "", []ActionResponseData{}} }
        if false != s1.Equal(s2) {
            t.Errorf("Failed to find mismatched req type")
        }

        s2.ReqType = TypeServerRequestAction
        if false != s1.Equal(s2) {
            t.Errorf("Failed to find mismatched reqData type")
        }

        s1.ReqData = ActionRequestData{"bar", "", "", "", []ActionResponseData{} }
        if false != s1.Equal(s2) {
            t.Errorf("Failed to find mismatched reqData value")
        }

        s1.ReqData = struct{}{}
        s2.ReqData = struct{}{}
        if false != s1.Equal(s2) {
            t.Errorf("Failed to find Unexpected ReqData type")
        }
    }

    {
        tx := LoMTransport{make(chan interface{}, 1)}
        chAbort := make(chan interface{}, 1)

        /* Send incorrect data type */
        {
            {
            t := &struct{}{}
            tx.ServerCh <- t
            }
            if p, e := tx.ReadClientRequest(1, chAbort); e == nil || p != nil {
                t.Errorf("Failed to fail for incorrect Req data type to server")
            }
        }

        /* Timeout */
        if p, e := tx.ReadClientRequest(1, chAbort); e == nil || p != nil {
            t.Errorf("Failed to fail for timeout")
        }

        /* explicit Abort */
        chAbort <- "Abort"
        if p, e := tx.ReadClientRequest(1, chAbort); e == nil || p != nil {
            t.Errorf("Failed to fail for abort")
        }
    } 
}

func TestHelper(t *testing.T) {
    {
        /* Test logger helper */
        FmtFprintfCnt := 0
        
        v := FmtFprintf
        FmtFprintf = func(w io.Writer, s string, a ...any) (int, error) {
            FmtFprintfCnt++
            return 0, nil
        }

        LogWarning("LoM: Lib Test WARNING messsage")
        if FmtFprintfCnt != 1 {
            t.Errorf("FmtFprintf not called")
        }

        lvl := GetLogLevel()
        SetLogLevel(syslog.LOG_ERR)
        if syslog.LOG_ERR != GetLogLevel() {
            t.Errorf("Failed tp set/get log level")
        }

        LogWarning("LoM: Lib Test WARNING messsage")
        if FmtFprintfCnt != 1 {
            t.Errorf("FmtFprintf is called when not expected")
        }

        SetLogLevel(syslog.LOG_DEBUG)

        FmtFprintf = v
        LogWarning("LoM: Lib Test WARNING messsage")
        if FmtFprintfCnt != 1 {
            t.Errorf("FmtFprintf is called when not expected")
        }
        SetLogLevel(lvl)
    }

    {
        /* Test log_panic to exit */
        ExitCnt := 0
        e := OSExit
        OSExit = func(v int) {
            ExitCnt++
        }
        LogPanic("Hitting Panic")
        if ExitCnt != 1 {
            t.Errorf("Panic test failed")
        }
        OSExit = e
    }

}

