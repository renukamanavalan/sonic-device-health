package libTest

import (
    "errors"
    "fmt"
    "testing"
    "time"

    cmn "lom/src/lib/lomcommon"
    tele "lom/src/lib/lomtelemetry"
)

func Test_PubSub(t *testing.T) {
    ch, err := tele.GetPubChannel(tele.CHANNEL_TYPE_EVENTS, tele.CHANNEL_PRODUCER_ENGINE, "")
    /* ch close indirectly closes corresponding PUB channel too */
    defer close(ch)

    if err != nil {
        t.Fatalf("Failed to get sub channel (%v)", err)
    }
    t.Logf("Test Complete")
}

var suiteCache = suiteCache_t{}

func callGetPubChannel(t *testing.T, args []any) []any {
    if len(args) != 3 {
        t.Fatalf("GetPubChannel expects 3 args. Given=%d", len(args))
    }
    if chType, ok := args[0].(tele.ChannelType_t); !ok {
        t.Fatalf("Expect tele.ChannelType_t != type(%T)", args[0])
    } else if producer, ok := args[1].(tele.ChannelProducer_t); !ok {
        t.Fatalf("Expect tele.ChannelProducer_t != type(%T)", args[1])
    } else if pluginName, ok := args[2].(string); !ok {
        t.Fatalf("Expect string != type(%T)", args[2])
    } else {
        ch, err := tele.GetPubChannel(chType, producer, pluginName)
        return []any{ch, err}
    }
    return []any{}
}

func callGetSubChannel(t *testing.T, args []any) []any {
    if len(args) != 3 {
        t.Fatalf("GetSubChannel expects 3 args. Given=%d", len(args))
    }
    if chType, ok := args[0].(tele.ChannelType_t); !ok {
        t.Fatalf("Expect tele.ChannelType_t != type(%T)", args[0])
    } else if producer, ok := args[1].(tele.ChannelProducer_t); !ok {
        t.Fatalf("Expect tele.ChannelProducer_t != type(%T)", args[1])
    } else if pluginName, ok := args[2].(string); !ok {
        t.Fatalf("Expect string != type(%T)", args[2])
    } else {
        ch, err := tele.GetSubChannel(chType, producer, pluginName)
        return []any{ch, err}
    }
    return []any{}
}

func callRunPubSubProxy(t *testing.T, args []any) []any {
    if len(args) != 1 {
        t.Fatalf("RunPubSubProxy expects 1 args. Given=%d", len(args))
    } else if chType, ok := args[0].(tele.ChannelType_t); !ok {
        t.Fatalf("Expect tele.ChannelType_t != type(%T)", args[0])
    } else {
        err := tele.RunPubSubProxy(chType)
        return []any{err}
    }
    return []any{}
}

func callSendClientRequest(t *testing.T, args []any) []any {
    if len(args) != 2 {
        t.Fatalf("SendClientRequest expects 2 args. Given=%d", len(args))
    } else if chType, ok := args[0].(tele.ChannelType_t); !ok {
        t.Fatalf("Expect tele.ChannelType_t != type(%T)", args[0])
    } else if req, ok := args[1].(tele.ClientReq_t); !ok {
        t.Fatalf("Expect ClientReq_t != type(%T)", args[1])
    } else {
        ch, err := tele.SendClientRequest(chType, req)
        return []any{ch, err}
    }
    return []any{}
}

func callRegisterServerReqHandler(t *testing.T, args []any) []any {
    if len(args) != 1 {
        t.Fatalf("RegisterServerReqHandler expects 1 args. Given=%d", len(args))
    } else if chType, ok := args[0].(tele.ChannelType_t); !ok {
        t.Fatalf("Expect tele.ChannelType_t != type(%T)", args[0])
    } else {
        chReq, chRes, err := tele.RegisterServerReqHandler(chType)
        return []any{chReq, chRes, err}
    }
    return []any{}
}

func callDoSysShutdown(t *testing.T, args []any) []any {
    if len(args) != 1 {
        t.Fatalf("DoSysShutdown need timeout")
    } else if tout, ok := args[0].(int); !ok {
        t.Fatalf("Expect int for timeout != type(%T)", args[0])
    } else {
        cmn.DoSysShutdown(tout)
    }
    return []any{}
}

func callWriteChannel(t *testing.T, args []any) []any {
    var err error
    if len(args) != 3 {
        t.Fatalf("WriteChannel need data to write")
    } else if ch, ok := args[0].(chan<- tele.JsonString_t); !ok {
        t.Fatalf("Expect tele.JsonString_t chan<- != type(%T)", args[0])
    } else if d, ok := args[1].(tele.JsonString_t); !ok {
        t.Fatalf("Expect string for data != type(%T)", args[1])
    } else if tout, ok := args[2].(int); !ok {
        t.Fatalf("Expect int for timeout != type(%T)", args[2])
    } else {
        select {
        case ch <- d:

        case <-time.After(time.Duration(tout) * time.Second):
            err = errors.New(fmt.Sprintf("Write chan timeout after (%d) seconds", tout))
        }
    }
    return []any{err}
}

func callReadChannel(t *testing.T, args []any) []any {
    var err error
    var readVal tele.JsonString_t = ""
    if len(args) != 2 {
        t.Fatalf("ReadChannel need data to write")
    } else if ch, ok := args[0].(<-chan tele.JsonString_t); !ok {
        t.Fatalf("Expect tele.JsonString_t <-chan != type(%T)", args[0])
    } else if tout, ok := args[1].(int); !ok {
        t.Fatalf("Expect int for timeout != type(%T)", args[1])
    } else {
        select {
        case val, ok := <-ch:
            if !ok {
                err = errors.New("CLOSED")
            } else {
                readVal = val
            }

        case <-time.After(time.Duration(tout) * time.Second):
            err = errors.New("TIMEOUT")
        }
    }
    return []any{readVal, err}
}

func callCloseChannel(t *testing.T, args []any) []any {
    if len(args) != 1 {
        t.Fatalf("WriteChannel need data to write")
    } else if ch, ok := args[0].(chan<- tele.JsonString_t); !ok {
        t.Fatalf("Expect tele.JsonString_t chan<- != type(%T)", args[0])
    } else {
        close(ch)
    }
    return []any{}
}

func callPause(t *testing.T, args []any) []any {
    if len(args) != 1 {
        t.Fatalf("WriteChannel need data to write")
    } else if tout, ok := args[0].(int); !ok {
        t.Fatalf(fmt.Sprintf("Expect pause time int != type(%T)", args[0]))
    } else {
        time.Sleep(time.Duration(tout) * time.Second)
    }
    return []any{}
}

func testRunOneTeleSuite(t *testing.T, suite *testSuite_t) {
    /* Caches all variables for reference across test entries */
    suiteCache = map[string]any{}

    t.Logf("Starting test suite - {%s} ....", suite.id)

    defer func() { t.Logf("Ended test suite - {%s} ....", suite.id) }()

    for i, entry := range suite.tests {
        t.Logf("Starting test[%d] - {%v} ....", i, entry.api)
        argvals := []any{}
        for _, v := range entry.args {
            argvals = append(argvals, suiteCache.getVal(v.name, v.val))
        }
        retVals := []any{}
        switch entry.api {
        case ApiIDGetPubChannel:
            retVals = callGetPubChannel(t, argvals)
        case ApiIDGetSubChannel:
            retVals = callGetSubChannel(t, argvals)
        case ApiIDRunPubSubProxy:
            retVals = callRunPubSubProxy(t, argvals)
        case ApiIDSendClientRequest:
            retVals = callSendClientRequest(t, argvals)
        case ApiIDDoSysShutdown:
            retVals = callDoSysShutdown(t, argvals)
        case ApiIDWriteChannel:
            retVals = callWriteChannel(t, argvals)
        case ApiIDReadChannel:
            retVals = callReadChannel(t, argvals)
        case ApiIDCloseChannel:
            retVals = callCloseChannel(t, argvals)
        case ApiIDPause:
            retVals = callPause(t, argvals)
        default:
            t.Fatalf("Unknown API ID %v", entry.api)
        }
        if len(retVals) != len(entry.result) {
            t.Fatalf(fmt.Sprintf("Return length (%d) != expected (%d)", len(retVals), len(entry.result)))
        }
        for j, e := range entry.result {
            retV := retVals[j]
            expVal := suiteCache.getVal(e.name, e.valExpect)
            if expVal != retV {
                t.Errorf("ExpVal(%v) != RetV(%v)", expVal, retV)
            }
            if e.validator != nil {
                if e.validator(e.name, expVal, retV) == false {
                    t.Errorf("Result validation failed (%+v) retv(%+v)", entry, retV)
                    retV = nil
                }
            }
            if e.name != ANONYMOUS {
                suiteCache.setVal(e.name, retV)
            }
        }
        t.Logf("Ended test - {%v} ....", entry.api)
    }
}

func setUTGlobals() {
    tele.SUB_CHANNEL_TIMEOUT = time.Duration(1) * time.Second
}

func TestRunTeleSuites(t *testing.T) {
    setUTGlobals()

    for _, suite := range testTelemetrySuites {
        testRunOneTeleSuite(t, suite)
    }
}
