package lomscripted

import (
    "errors"
    "fmt"
    "time"

    cmn "lom/src/lib/lomcommon"
    tele "lom/src/lib/lomtelemetry"
)

func callGetPubChannel(args []any) []any {
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

func callGetSubChannel(args []any) []any {
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

func callRunPubSubProxy(args []any) []any {
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

func callSendClientRequest(args []any) []any {
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

func callRegisterServerReqHandler(args []any) []any {
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

func callWriteChannel(args []any) []any {
    var err error
    if len(args) != 3 {
        err = cmn.LogError("WriteChannel need 3 args")
    } else if ch, ok := args[0].(chan<- tele.JsonString_t); !ok {
        err = cmn.LogError("Expect tele.JsonString_t chan<- != type(%T)", args[0])
    } else if d, ok := args[1].(tele.JsonString_t); !ok {
        err = cmn.LogError("mis type. Expect JsonString_t != type(%T)", args[1])
    } else if tout, ok := args[2].(int); !ok {
        err = cmn.LogError("Expect int for timeout != type(%T)", args[2])
    } else {
        select {
        case ch <- d:

        case <-time.After(time.Duration(tout) * time.Second):
            err = errors.New(fmt.Sprintf("Write chan timeout after (%d) seconds", tout))
        }
    }
    return []any{err}
}

func callReadChannel(args []any) []any {
    var err error
    var readVal tele.JsonString_t = ""
    if len(args) != 2 {
        err = cmn.LogError("ReadChannel need 2 args")
    } else if ch, ok := args[0].(<-chan tele.JsonString_t); !ok {
        err = cmn.LogError("Expect tele.JsonString_t <-chan != type(%T)", args[0])
    } else if tout, ok := args[1].(int); !ok {
        err = cmn.LogError("Expect int for timeout != type(%T)", args[1])
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

func callCloseChannel(args []any) []any {
    var err error
    if len(args) != 1 {
        err = cmn.LogError("WriteChannel need data to write")
    } else if ch, ok := args[0].(chan<- tele.JsonString_t); ok {
        close(ch)
    } else if ch, ok := args[0].(chan<- int); ok {
        close(ch)
    } else {
        err = cmn.LogError("Expect chan<- int/JsonString_t != type(%T)", args[0])
    }
    return []any{err}
}

func callPause(args []any) []any {
    var err error
    if len(args) != 1 {
        err = cmn.LogError("WriteChannel need data to write")
    } else if tout, ok := args[0].(int); !ok {
        err = cmn.LogError("Expect pause time int != type(%T)", args[0])
    } else {
        time.Sleep(time.Duration(tout) * time.Second)
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
        retVals = fn(argvals)
    }
    return
}
