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
    }
    if chType, ok := args[0].(tele.ChannelType_t); !ok {
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
    }
    if chType, ok := args[0].(tele.ChannelType_t); !ok {
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

func writeData(ch chan<- tele.JsonString_t, d []tele.JsonString_t, tout int) (err error) {
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

func writeDataStreaming(ch chan<- tele.JsonString_t, rdFn GetValStreamingFn_t, tout int, cache SuiteCache_t) (err error) {
    var dp *streamingDataEntity_t
    for i := 0; err == nil; i++ {
        if dp, err = rdFn(i, cache); err == nil {
            err = writeData(ch, dp.Data, tout)
            if !dp.More {
                /* On no more data break & return */
                break
            }
        }
    }
    return 
}

func callWriteChannel(args []any, cache SuiteCache_t) []any {
    var err error
    if len(args) != 3 {
        err = cmn.LogError("WriteChannel need min 3 args, as chan, data & timeout")
    } else if ch, ok := args[0].(chan<- tele.JsonString_t); !ok {
        err = cmn.LogError("Expect tele.JsonString_t chan<- != type(%T)", args[0])
    } else if tout, ok := args[2].(int); !ok {
        err = cmn.LogError("Expect int for timeout != type(%T)", args[2])
    } else {
        switch args[1].(type) {
        case []tele.JsonString_t:
            err = writeData(ch, args[1].([]tele.JsonString_t), tout)
        case GetValStreamingFn_t:
            err = writeDataStreaming(ch, args[1].(GetValStreamingFn_t), tout, cache)
        default:
            err = cmn.LogError("Unknown data type (%T)", args[1])
        }
    }
    return []any{err}
}

func readData(ch <-chan tele.JsonString_t, tout int, cnt int) (vals []tele.JsonString_t, err error) {
    vals = []tele.JsonString_t{}

    for i := 0; (i < cnt) && (err == nil); i++ {
        select {
        case val, ok := <-ch:
            if !ok {
                err = errors.New("CLOSED")
            } else {
                vals = append(vals, val)
            }

        case <-time.After(time.Duration(tout) * time.Second):
            err = errors.New("TIMEOUT")
        }
    }
    return
}

func readDataStreaming(ch <-chan tele.JsonString_t, tout int, wrFn PutValStreamingFn_t,
        cache SuiteCache_t) (err error) {
    more := true
    for i := 0; more && (err == nil); i++ {
        if vals, err := readData(ch, tout, 1); err == nil {
            more, err = wrFn(i, vals[0], cache)
        }
    }
    return
}

func callReadChannel(args []any, cache SuiteCache_t) []any {
    var err error
    readVal := []tele.JsonString_t{}
    if len(args) != 3 {
        err = cmn.LogError("ReadChannel need 3 args, chan, read-count/fn & timeout ")
    } else if ch, ok := args[0].(<-chan tele.JsonString_t); !ok {
        err = cmn.LogError("Expect tele.JsonString_t <-chan != type(%T)", args[0])
    } else if tout, ok := args[2].(int); !ok {
        err = cmn.LogError("Expect int for timeout != type(%T)", args[1])
    } else {
        switch args[1].(type) {
        case int:
            readVal, err = readData(ch, tout, args[1].(int))
        case PutValStreamingFn_t:
            err = readDataStreaming(ch, tout, args[1].(PutValStreamingFn_t), cache)
        }
    }
    return []any{readVal, err}
}

func callCloseChannel(args []any, cache SuiteCache_t) []any {
    var err error
    for i, chAny := range args {
        if ch, ok := chAny.(chan<- tele.JsonString_t); ok {
            close(ch)
        } else if ch, ok := chAny.(chan<- int); ok {
            close(ch)
        } else {
            /* Last error gets returned. */
            err = cmn.LogError("%d: Expect chan<- int/JsonString_t != type(%T)", i, chAny)
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

