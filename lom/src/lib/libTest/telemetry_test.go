package libTest

import (
    "errors"
    "fmt"
    "testing"
    "time"

    cmn "lom/src/lib/lomcommon"
    tele "lom/src/lib/lomtelemetry"
)

func xTest_PubSub(t *testing.T) {
    ch, err := tele.GetPubChannel(tele.CHANNEL_TYPE_EVENTS, tele.CHANNEL_PRODUCER_ENGINE, "")
    /* ch close indirectly closes corresponding PUB channel too */
    defer close(ch)

    if err != nil {
        t.Fatalf(fatalFmt("Failed to get sub channel (%v)", err))
    }
    t.Logf(logFmt("Test Complete"))
}

var suiteCache = suiteCache_t{}

func callGetPubChannel(t *testing.T, args []any) []any {
    if len(args) != 3 {
        t.Fatalf(fatalFmt("GetPubChannel expects 3 args. Given=%d", len(args)))
    }
    if chType, ok := args[0].(tele.ChannelType_t); !ok {
        t.Fatalf(fatalFmt("Expect tele.ChannelType_t != type(%T)", args[0]))
    } else if producer, ok := args[1].(tele.ChannelProducer_t); !ok {
        t.Fatalf(fatalFmt("Expect tele.ChannelProducer_t != type(%T)", args[1]))
    } else if pluginName, ok := args[2].(string); !ok {
        t.Fatalf(fatalFmt("Expect string != type(%T)", args[2]))
    } else {
        ch, err := tele.GetPubChannel(chType, producer, pluginName)
        return []any{ch, err}
    }
    return []any{}
}

func callGetSubChannel(t *testing.T, args []any) []any {
    if len(args) != 3 {
        t.Fatalf(fatalFmt("GetSubChannel expects 3 args. Given=%d", len(args)))
    }
    if chType, ok := args[0].(tele.ChannelType_t); !ok {
        t.Fatalf(fatalFmt("Expect tele.ChannelType_t != type(%T)", args[0]))
    } else if producer, ok := args[1].(tele.ChannelProducer_t); !ok {
        t.Fatalf(fatalFmt("Expect tele.ChannelProducer_t != type(%T)", args[1]))
    } else if pluginName, ok := args[2].(string); !ok {
        t.Fatalf(fatalFmt("Expect string != type(%T)", args[2]))
    } else {
        ch, err := tele.GetSubChannel(chType, producer, pluginName)
        return []any{ch, err}
    }
    return []any{}
}

func callRunPubSubProxy(t *testing.T, args []any) []any {
    if len(args) != 1 {
        t.Fatalf(fatalFmt("RunPubSubProxy expects 1 args. Given=%d", len(args)))
    } else if chType, ok := args[0].(tele.ChannelType_t); !ok {
        t.Fatalf(fatalFmt("Expect tele.ChannelType_t != type(%T)", args[0]))
    } else {
        err := tele.RunPubSubProxy(chType)
        return []any{err}
    }
    return []any{}
}

func callSendClientRequest(t *testing.T, args []any) []any {
    if len(args) != 2 {
        t.Fatalf(fatalFmt("SendClientRequest expects 2 args. Given=%d", len(args)))
    } else if chType, ok := args[0].(tele.ChannelType_t); !ok {
        t.Fatalf(fatalFmt("Expect tele.ChannelType_t != type(%T)", args[0]))
    } else if req, ok := args[1].(tele.ClientReq_t); !ok {
        t.Fatalf(fatalFmt("Expect ClientReq_t != type(%T)", args[1]))
    } else {
        ch, err := tele.SendClientRequest(chType, req)
        return []any{ch, err}
    }
    return []any{}
}

func callRegisterServerReqHandler(t *testing.T, args []any) []any {
    if len(args) != 1 {
        t.Fatalf(fatalFmt("RegisterServerReqHandler expects 1 args. Given=%d", len(args)))
    } else if chType, ok := args[0].(tele.ChannelType_t); !ok {
        t.Fatalf(fatalFmt("Expect tele.ChannelType_t != type(%T)", args[0]))
    } else {
        chReq, chRes, err := tele.RegisterServerReqHandler(chType)
        return []any{chReq, chRes, err}
    }
    return []any{}
}

func callDoSysShutdown(t *testing.T, args []any) []any {
    if len(args) != 1 {
        t.Fatalf(fatalFmt("DoSysShutdown need timeout"))
    } else if tout, ok := args[0].(int); !ok {
        t.Fatalf(fatalFmt("Expect int for timeout != type(%T)", args[0]))
    } else {
        cmn.DoSysShutdown(tout)
    }
    return []any{}
}

func callWriteChannel(t *testing.T, args []any) []any {
    var err error
    if len(args) != 3 {
        t.Fatalf(fatalFmt("WriteChannel need 3 args"))
    } else if ch, ok := args[0].(chan<- tele.JsonString_t); !ok {
        t.Fatalf(fatalFmt("Expect tele.JsonString_t chan<- != type(%T)", args[0]))
    } else if d, ok := args[1].(tele.JsonString_t); !ok {
        t.Fatalf(fatalFmt("mis type. Expect JsonString_t != type(%T)", args[1]))
    } else if tout, ok := args[2].(int); !ok {
        t.Fatalf(fatalFmt("Expect int for timeout != type(%T)", args[2]))
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
        t.Fatalf(fatalFmt("ReadChannel need 2 args"))
    } else if ch, ok := args[0].(<-chan tele.JsonString_t); !ok {
        t.Fatalf(fatalFmt("Expect tele.JsonString_t <-chan != type(%T)", args[0]))
    } else if tout, ok := args[1].(int); !ok {
        t.Fatalf(fatalFmt("Expect int for timeout != type(%T)", args[1]))
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
        t.Fatalf(fatalFmt("WriteChannel need data to write"))
    } else if ch, ok := args[0].(chan<- tele.JsonString_t); !ok {
        t.Fatalf(fatalFmt("Expect tele.JsonString_t chan<- != type(%T)", args[0]))
    } else {
        close(ch)
    }
    return []any{}
}

func callPause(t *testing.T, args []any) []any {
    if len(args) != 1 {
        t.Fatalf(fatalFmt("WriteChannel need data to write"))
    } else if tout, ok := args[0].(int); !ok {
        t.Fatalf(fatalFmt("Expect pause time int != type(%T)", args[0]))
    } else {
        time.Sleep(time.Duration(tout) * time.Second)
    }
    return []any{}
}

func testRunOneTeleSuite(t *testing.T, suite *testSuite_t) {
    /* Caches all variables for reference across test entries */
    suiteCache = map[string]any{}

    t.Logf(logFmt("Starting test suite - {%s} ....", suite.id))

    defer func() { t.Logf(logFmt("Ended test suite - {%s} ....", suite.id)) }()

    for i, entry := range suite.tests {
        t.Logf(logFmt("Starting test[%d] - {%v} ....", i, entry.api))
        argvals := []any{}
        for ai, v := range entry.args {
            if val, ok := suiteCache.getVal(v.name, v.val); !ok {
                t.Fatalf(fatalFmt("Failed to getVal test(%v) arg{%d: (%s)}",
                        entry.api, ai, v.name))
            } else {
                argvals = append(argvals, val)
            }
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
            t.Fatalf(fatalFmt("Unknown API ID %v", entry.api))
        }
        if len(retVals) != len(entry.result) {
            t.Fatalf(fatalFmt("Return length (%d) != expected (%d)", len(retVals), len(entry.result)))
        }
        for j, e := range entry.result {
            retV := retVals[j]
            var expVal any
            ok := false
            if expVal, ok = suiteCache.getVal(e.name, e.valExpect); ok {
                if expVal != retV {
                    t.Fatalf(fatalFmt("ExpVal(%v) != RetV(%v)(%T)", expVal, retV, retV))
                }
            }
            if e.validator != nil {
                if e.validator(e.name, expVal, retV) == false {
                    t.Errorf(errorFmt("Result validation failed (%+v) retv(%+v)", entry, retV))
                    retV = nil
                }
            }
            suiteCache.setVal(e.name, retV)
        }
        t.Logf(logFmt("Ended test - {%v} ....", entry.api))
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

