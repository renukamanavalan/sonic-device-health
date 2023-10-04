package lomscripted

import (
    "errors"
    "fmt"
    "time"

    cmn "lom/src/lib/lomcommon"
    tele "lom/src/lib/lomtelemetry"
)

func callGetPubChannel(args []any, cache SuiteCache_t) []any {
    var err error
    var ch any
    if len(args) != 3 {
        err = cmn.LogError("GetPubChannel expects 3 args. Given=%d", len(args))
    } else if chType, ok := args[0].(tele.ChannelType_t); !ok {
        err = cmn.LogError("Expect tele.ChannelType_t != type(%T)", args[0])
    } else if producer, ok := args[1].(tele.ChannelProducer_t); !ok {
        err = cmn.LogError("Expect tele.ChannelProducer_t != type(%T)", args[1])
    } else if pluginName, ok := args[2].(string); !ok {
        err = cmn.LogError("Expect string != type(%T)", args[2])
    } else {
        ch, err = tele.GetPubChannel(chType, producer, pluginName)
    }
    return []any{ch, err}
}

func callGetSubChannel(args []any, cache SuiteCache_t) []any {
    var err error
    var ch, chClose any
    if len(args) != 3 {
        err = cmn.LogError("GetSubChannel expects 3 args. Given=%d", len(args))
    } else if chType, ok := args[0].(tele.ChannelType_t); !ok {
        err = cmn.LogError("Expect tele.ChannelType_t != type(%T)", args[0])
    } else if producer, ok := args[1].(tele.ChannelProducer_t); !ok {
        err = cmn.LogError("Expect tele.ChannelProducer_t != type(%T)", args[1])
    } else if pluginName, ok := args[2].(string); !ok {
        err = cmn.LogError("Expect string != type(%T)", args[2])
    } else {
        ch, chClose, err = tele.GetSubChannel(chType, producer, pluginName)
    }
    return []any{ch, chClose, err}
}

func callRunPubSubProxy(args []any, cache SuiteCache_t) []any {
    var err error
    var chClose any
    if len(args) != 1 {
        err = cmn.LogError("RunPubSubProxy expects 1 args. Given=%d", len(args))
    } else if chType, ok := args[0].(tele.ChannelType_t); !ok {
        err = cmn.LogError("Expect tele.ChannelType_t != type(%T)", args[0])
    } else {
        chClose, err = tele.RunPubSubProxy(chType)
    }
    return []any{chClose, err}
}

func callSendClientRequest(args []any, cache SuiteCache_t) []any {
    var err error
    var ch any
    if len(args) != 2 {
        err = cmn.LogError("SendClientRequest expects 2 args. Given=%d", len(args))
    } else if chType, ok := args[0].(tele.ChannelType_t); !ok {
        err = cmn.LogError("Expect tele.ChannelType_t != type(%T)", args[0])
    } else if req, ok := args[1].(tele.ClientReq_t); !ok {
        err = cmn.LogError("Expect ClientReq_t != type(%T)", args[1])
    } else {
        ch, err = tele.SendClientRequest(chType, req)
    }
    return []any{ch, err}
}

func callReadClientResponse(args []any, cache SuiteCache_t) []any {
    var err error
    var val tele.ServerRes_t
    if len(args) != 2 {
        err = cmn.LogError("SendClientRequest expects 2 args. Given=%d", len(args))
    } else if ch, ok := args[0].(<-chan *tele.ClientRes_t); !ok || ch == nil {
        err = cmn.LogError("Expect non nil <-chan *tele.ClientRes_t != type(%T)", args[0])
    } else if tout, ok := args[1].(int); !ok {
        err = cmn.LogError("Expect int for timeout != type(%T)", args[1])
    } else {
        select {
        case v, ok := <-ch:
            if !ok {
                err = errors.New("CLOSED")
            } else {
                err = v.Err
                val = v.Res
            }
        case <-time.After(time.Duration(tout) * time.Second):
            err = errors.New("TIMEOUT")
        }
    }
    return []any{val, err}
}

func callReadClientRequest(args []any, cache SuiteCache_t) []any {
    var err error
    var val tele.ClientReq_t
    if len(args) != 2 {
        err = cmn.LogError("ReadJsonStringsChannel need 3 args, chan, read-count/fn & timeout ")
    } else if ch, ok := args[0].(<-chan tele.ClientReq_t); !ok || ch == nil {
        err = cmn.LogError("Expect non-nil tele.ClientReq_t <-chan != type(%T)", args[0])
    } else if tout, ok := args[1].(int); !ok {
        err = cmn.LogError("Expect int for timeout != type(%T)", args[1])
    } else {
        select {
        case val, ok = <-ch:
            if !ok {
                err = cmn.LogError("Read chan CLOSED")
            }
        case <-time.After(time.Duration(tout) * time.Second):
            err = cmn.LogError("Read chan TIMEOUT %d secs", tout)
        }
    }
    return []any{val, err}
}

func callSendClientResponse(args []any, cache SuiteCache_t) []any {
    var err error
    if len(args) != 3 {
        err = cmn.LogError("SendClientResponse need 3 args, as chan, data & timeout")
    } else if ch, ok := args[0].(chan<- tele.ServerRes_t); !ok || ch == nil {
        err = cmn.LogError("Expect non-nil chan<- tele.ServerRes_t != type(%T)", args[0])
    } else if res, ok := args[1].(tele.ServerRes_t); !ok {
        err = cmn.LogError("Expect ServerRes_t != type(%T)", args[1])
    } else if tout, ok := args[2].(int); !ok {
        err = cmn.LogError("Expect int for timeout != type(%T)", args[2])
    } else {
        select {
        case ch <- res:

        case <-time.After(time.Duration(tout) * time.Second):
            err = errors.New(fmt.Sprintf("SendClientResponse TIMEOUT (%d) secs", tout))
        }
    }
    return []any{err}
}

func callRegisterServerReqHandler(args []any, cache SuiteCache_t) []any {
    var err error
    var chReq, chRes any
    if len(args) != 1 {
        err = cmn.LogError("RegisterServerReqHandler expects 1 args. Given=%d", len(args))
    } else if chType, ok := args[0].(tele.ChannelType_t); !ok {
        err = cmn.LogError("Expect tele.ChannelType_t != type(%T)", args[0])
    } else {
        chReq, chRes, err = tele.RegisterServerReqHandler(chType)
    }
    return []any{chReq, chRes, err}
}

func writeJsonStringsData(ch chan<- tele.JsonString_t, d []tele.JsonString_t, tout int) (err error) {
    for i, val := range d {
        select {
        case ch <- val:

        case <-time.After(time.Duration(tout) * time.Second):
            err = errors.New(fmt.Sprintf("Write chan timeout on index(%d/%d) after (%d) seconds",
                i, len(d), tout))
            return
        }
    }
    return
}

func writeJsonStringStreaming(ch chan<- tele.JsonString_t, rdFn GetValStreamingFn_t, tout int, cache SuiteCache_t) (err error) {
    var dp *StreamingDataEntity_t
    for i := 0; err == nil; i++ {
        if dp, err = rdFn(i, cache); err == nil {
            err = writeJsonStringsData(ch, dp.Data, tout)
            if !dp.More {
                /* On no more data break & return */
                break
            }
        }
    }
    return
}

func callWriteJsonStringsChannel(args []any, cache SuiteCache_t) []any {
    var err error
    if len(args) != 3 {
        err = cmn.LogError("WriteJsonStringsChannel need 3 args, as chan, data & timeout")
    } else if ch, ok := args[0].(chan<- tele.JsonString_t); !ok || ch == nil {
        err = cmn.LogError("Expect tele.JsonString_t chan<- != type(%T)", args[0])
    } else if tout, ok := args[2].(int); !ok {
        err = cmn.LogError("Expect int for timeout != type(%T)", args[2])
    } else {
        if val, ok := args[1].([]tele.JsonString_t); ok {
            err = writeJsonStringsData(ch, val, tout)
        } else if val, ok := args[1].(GetValStreamingFn_t); ok {
            err = writeJsonStringStreaming(ch, val, tout, cache)
        } else {
            err = cmn.LogError("Unknown data type (%T)", args[1])
        }
    }
    return []any{err}
}

func readJsonStrings(ch <-chan tele.JsonString_t, tout int, cnt int) (retVal []tele.JsonString_t, err error) {
    vals := []tele.JsonString_t{}

Loop:
    for i := 0; (i < cnt) && (err == nil); i++ {
        select {
        case val, ok := <-ch:
            if !ok {
                err = errors.New("CLOSED")
                break Loop
            } else {
                vals = append(vals, val)
            }

        case <-time.After(time.Duration(tout) * time.Second):
            err = errors.New("TIMEOUT")
            break Loop
        }
    }
    if err == nil {
        retVal = vals
    }
    return
}

func readJsonStringStreaming(ch <-chan tele.JsonString_t, tout int, wrFn PutValStreamingFn_t,
    cache SuiteCache_t) (err error) {
    more := true
    for i := 0; more && (err == nil); i++ {
        if vals, err := readJsonStrings(ch, tout, 1); err == nil {
            more, err = wrFn(i, vals[0], cache)
        }
    }
    return
}

func callReadJsonStringsChannel(args []any, cache SuiteCache_t) []any {
    var err error
    var readVal []tele.JsonString_t
    if len(args) != 3 {
        err = cmn.LogError("ReadJsonStringsChannel need 3 args, chan, read-count/fn & timeout ")
    } else if ch, ok := args[0].(<-chan tele.JsonString_t); !ok {
        err = cmn.LogError("Expect tele.JsonString_t <-chan != type(%T)", args[0])
    } else if tout, ok := args[2].(int); !ok {
        err = cmn.LogError("Expect int for timeout != type(%T)", args[2])
    } else {
        if val, ok := args[1].(int); ok {
            readVal, err = readJsonStrings(ch, tout, val)
        } else if val, ok := args[1].(func(int, tele.JsonString_t, SuiteCache_t) (bool, error)); ok {
            err = readJsonStringStreaming(ch, tout, val, cache)
        } else {
            err = cmn.LogError("Expect cnt to read or func to write. Got (%T)", args[1])
        }
    }
    return []any{readVal, err}
}

func callCloseRequestChannel(args []any, cache SuiteCache_t) []any {
    var err error
    if len(args) != 1 {
        err = cmn.LogError("CloseRequestChannel expects 1 args. Given=%d", len(args))
    } else if chType, ok := args[0].(tele.ChannelType_t); !ok {
        err = cmn.LogError("Expect tele.ChannelType_t != type(%T)", args[0])
    } else {
        err = tele.CloseClientRequest(chType)
    }
    return []any{err}
}

func callCloseChannel(args []any, cache SuiteCache_t) []any {
    var err error
Loop:
    for i, chAny := range args {
        switch chAny.(type) {
        case chan<- tele.JsonString_t:
            close(chAny.(chan<- tele.JsonString_t))
        case chan<- tele.ServerRes_t:
            close(chAny.(chan<- tele.ServerRes_t))
        case chan<- int:
            close(chAny.(chan<- int))
        default:
            err = cmn.LogError("%d: Unknown type for close (%T)", i, chAny)
            break Loop
        }
    }
    return []any{err}
}

func callPause(args []any, cache SuiteCache_t) []any {
    var err error
    if len(args) != 1 {
        err = cmn.LogError("Pause need time in seconds")
    } else if tout, ok := args[0].(int); !ok {
        err = cmn.LogError("Expect pause time int != type(%T)", args[0])
    } else {
        cmn.LogInfo("Pause sleeps for %d seconds", tout)
        time.Sleep(time.Duration(tout) * time.Second)
    }
    return []any{err}
}

func callIsTelemetryIdle(args []any, cache SuiteCache_t) []any {
    var err error
    ret := false
    if len(args) != 0 {
        err = cmn.LogError("Chec for idle need no args")
    } else {
        ret = tele.IsTelemetryIdle()
    }
    return []any{ret, err}
}

func callDoSysShutdown(args []any, cache SuiteCache_t) []any {
    var err error
    if len(args) != 1 {
        err = cmn.LogError("shutdown need timeout in seconds")
    } else if tout, ok := args[0].(int); !ok {
        err = cmn.LogError("Expect timeout as int != type(%T)", args[0])
    } else {
        cmn.DoSysShutdown(tout)
    }
    return []any{err}
}

func callInitSysShutdown(args []any, cache SuiteCache_t) []any {
    var err error
    if len(args) != 0 {
        err = cmn.LogError("init shutdown takes no args")
    } else {
        cmn.InitSysShutdown()
    }
    return []any{err}
}

func CallByApiID(api ApiId_t, args []Param_t, cache SuiteCache_t) (retVals []any, ok bool) {
    var fn ApiFn_t

    if fn, ok = LomAPIByIds[api]; ok {
        argvals := []any{}
        for _, v := range args {
            argvals = append(argvals, cache.GetVal(v.Name, v.Val, v.GetFn))
        }
        retVals = fn(argvals, cache)
    }
    return
}
