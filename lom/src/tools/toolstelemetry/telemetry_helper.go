package toolstelemetry

import (
    "bufio"
    "fmt"
    "strings"

    cmn "lom/src/lib/lomcommon"
    script "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

func getSlice(name string, cache script.SuiteCache_t) []tele.JsonString_t {
    data := []tele.JsonString_t{}
    if dataV := cache.GetVal(name, nil, nil); dataV != nil {
        if sl, ok := dataV.([]tele.JsonString_t); ok {
            data = sl
        }
    }
    return data
}

func getJsonStringsFromReader(name string, val any) (ret any, err error) {
    reader, ok := val.(*bufio.Reader)
    if !ok || (reader == nil) {
        err = cmn.LogError("Val incorrect type (%T) != *bufio.Readerr", val)
        return
    }
    more := true

    ret = func(_ int, cache script.SuiteCache_t) (*script.StreamingDataEntity_t, error) {
        fmt.Printf("Enter string to publish\n")
        text, err := reader.ReadString('\n')
        if err == nil {
            text = strings.TrimSpace(text)
        } else {
            text = ""
        }
        teleTxt := tele.JsonString_t(text)
        if text == "" {
            more = false
        } else if name != script.ANONYMOUS {
            cache.SetVal(name, append(getSlice(name, cache), teleTxt))
        }
        return &script.StreamingDataEntity_t{[]tele.JsonString_t{teleTxt}, more}, err
    }
    return
}

func putJsonStringsIntoWriter(name string, val any) (ret any, err error) {
    writer, ok := val.(*bufio.Writer)
    if !ok || (writer == nil) {
        err = cmn.LogError("Val incorrect type (%T) != *bufio.Writer", val)
        return
    }

    ret = func(_ int, data tele.JsonString_t, cache script.SuiteCache_t) (
        more bool, err error) {
        if _, err = writer.WriteString(string(data) + "\n"); err == nil {
            err = writer.Flush()
        }
        if err == nil {
            more = true
            if name != script.ANONYMOUS {
                cache.SetVal(name, append(getSlice(name, cache), data))
            }
        }
        return
    }
    return
}

func GetPubSuite(chType tele.ChannelType_t, chProd tele.ChannelProducer_t, suffix string,
    reader *bufio.Reader) (ret *script.ScriptSuite_t, err error) {

    if reader == nil {
        err = cmn.LogError("Expect non nil *bufio.Reader")
        return
    }
    ret = &script.ScriptSuite_t{
        Id:          "pubFromStdin",
        Description: "Read a line from stdin & publish until EOF",
        Entries: []script.ScriptEntry_t{
            script.ScriptEntry_t{
                script.ApiIDGetPubChannel,
                []script.Param_t{
                    script.Param_t{script.ANONYMOUS, chType, nil},
                    script.Param_t{script.ANONYMOUS, chProd, nil},
                    script.Param_t{script.ANONYMOUS, suffix, nil},
                },
                []script.Result_t{
                    script.Result_t{"chPub-0", nil, script.ValidateNonNil}, /* Save in cache */
                    script.Result_t{script.ANONYMOUS, nil, script.ValidateNil},
                },
                "Get pub channel for same type as proxy above",
            },
            script.ScriptEntry_t{
                script.ApiIDWriteJsonStringsChannel,
                []script.Param_t{
                    script.Param_t{"chPub-0", nil, nil}, /* From cache */
                    script.Param_t{script.ANONYMOUS, reader, getJsonStringsFromReader},
                    script.Param_t{script.ANONYMOUS, 1, nil}, /* timeout = 1 second */
                },
                []script.Result_t{script.NIL_ERROR},
                "Get pub channel for same type as proxy above",
            },
            script.ScriptEntry_t{
                script.ApiIDCloseChannel,
                []script.Param_t{
                    script.Param_t{"chPub-0", nil, nil}, /* Get from cache */
                },
                []script.Result_t{script.NIL_ERROR},
                "Close pub chennel",
            },
        },
    }
    return
}

func GetSubSuite(chType tele.ChannelType_t, chProd tele.ChannelProducer_t, suffix string,
    writer *bufio.Writer) (ret *script.ScriptSuite_t, err error) {

    if writer == nil {
        err = cmn.LogError("Expect non nil *bufio.Writer")
        return
    }
    ret = &script.ScriptSuite_t{
        Id:          "subIntoStdout",
        Description: "Write data fron sub channel to stdout",
        Entries: []script.ScriptEntry_t{
            script.ScriptEntry_t{
                script.ApiIDGetSubChannel,
                []script.Param_t{
                    script.Param_t{script.ANONYMOUS, chType, nil},
                    script.Param_t{script.ANONYMOUS, chProd, nil},
                    script.Param_t{script.ANONYMOUS, suffix, nil},
                },
                []script.Result_t{
                    script.Result_t{"chSub-0", nil, script.ValidateNonNil},   /* Save in cache */
                    script.Result_t{"chClose-0", nil, script.ValidateNonNil}, /* Save in cache */
                    script.Result_t{script.ANONYMOUS, nil, script.ValidateNil},
                },
                "Get Sub channel for same type as proxy above",
            },
            script.ScriptEntry_t{
                script.ApiIDReadJsonStringsChannel,
                []script.Param_t{
                    script.Param_t{"chSub-0", nil, nil}, /* From cache */
                    script.Param_t{script.ANONYMOUS, writer, putJsonStringsIntoWriter},
                    script.Param_t{script.ANONYMOUS, 100, nil}, /* timeout = 100 second */
                },
                []script.Result_t{script.NIL_ANY, script.NIL_ERROR},
                "Get Sub channel for same type as proxy above",
            },
            script.ScriptEntry_t{
                script.ApiIDCloseChannel,
                []script.Param_t{
                    script.Param_t{"chClose-0", nil, nil}, /* Get from cache */
                },
                []script.Result_t{script.NIL_ERROR},
                "Close Sub chennel",
            },
        },
    }
    return
}

func GetProxySuite(chType tele.ChannelType_t, tout int) (ret *script.ScriptSuite_t, err error) {

    ret = &script.ScriptSuite_t{
        Id:          "RunProxy",
        Description: "Run Proxy for given timeout",
        Entries: []script.ScriptEntry_t{
            script.ScriptEntry_t{
                script.ApiIDRunPubSubProxy,
                []script.Param_t{script.Param_t{script.ANONYMOUS, chType, nil}},
                []script.Result_t{
                    script.Result_t{"chPrxyClose-0", nil, script.ValidateNonNil}, /* Save in cache */
                    script.NIL_ERROR, /*Expect nil error */
                },
                "Get pubsub proxy, required to bind publishers & subscribers",
            },
            script.ScriptEntry_t{
                script.ApiIDPause,
                []script.Param_t{script.Param_t{script.ANONYMOUS, tout, nil}},
                []script.Result_t{script.NIL_ERROR},
                "Pause for tout seconds",
            },
            script.ScriptEntry_t{
                script.ApiIDCloseChannel,
                []script.Param_t{
                    script.Param_t{"chPrxyClose-0", nil, nil}, /* Get from cache */
                },
                []script.Result_t{script.NIL_ERROR},
                "Close proxy channel",
            },
        },
    }
    return
}

func getStrFromReader(name string, val any) (ret any, err error) {
    isClient := false
    data := ""

    fmt.Printf("Enter request to send\n")
    if lst, ok := val.([]any); !ok || len(lst) != 2 {
        err = cmn.LogError("Expect slice of any of 2")
    } else if isClient, ok = lst[0].(bool); !ok {
        err = cmn.LogError("Expect slice first entry bool != (%T)", lst[0])
    } else if reader, ok := lst[1].(*bufio.Reader); !ok || (reader == nil) {
        err = cmn.LogError("Expect slice second entry *bufio.Reader != (%T)", lst[1])
    } else if data, err = reader.ReadString('\n'); err != nil {
    } else if isClient {
        ret = tele.ClientReq_t(data)
    } else {
        ret = tele.ServerRes_t(data)
    }
    return
}

func putStrToWriter(name string, valExpect, valRet any) bool {
    var err error
    isClient := false

    if lst, ok := valExpect.([]any); !ok || len(lst) != 2 {
        err = cmn.LogError("Expect slice of any of 2")
    } else if isClient, ok = lst[0].(bool); !ok {
        err = cmn.LogError("Expect slice first entry bool != (%T)", lst[0])
    } else if writer, ok := lst[1].(*bufio.Writer); !ok || (writer == nil) {
        err = cmn.LogError("Expect slice second entry *bufio.Writer != (%T)", lst[1])
    } else {
        data := ""
        switch valRet.(type) {
        case tele.ServerRes_t:
            if !isClient {
                err = cmn.LogError("Only client expects tele.ServerRes_t != (%T)", valRet)
            } else {
                data = string(valRet.(tele.ServerRes_t))
            }
        case tele.ClientReq_t:
            if isClient {
                err = cmn.LogError("Only client expects tele.ClientReq_t != (%T)", valRet)
            } else {
                data = string(valRet.(tele.ClientReq_t))
            }
        default:
            if !isClient {
                err = cmn.LogError("Only client expects tele.ServerRes_t != (%T)", valRet)
            } else {
                err = cmn.LogError("Only client expects tele.ClientReq_t != (%T)", valRet)
            }
        }
        if err == nil {
            if _, err = writer.WriteString(data + "\n"); err == nil {
                err = writer.Flush()
            }
        }
    }
    return err == nil
}

func GetReqSuite(chType tele.ChannelType_t, cnt int, writer *bufio.Writer,
    reader *bufio.Reader, tout int) (ret *script.ScriptSuite_t, err error) {

    ret = &script.ScriptSuite_t{
        Id:          "ClientReqLoop",
        Description: "Run a loop for request",
        Entries: []script.ScriptEntry_t{
            script.ScriptEntry_t{
                script.ApiIDSendClientRequest,
                []script.Param_t{
                    script.Param_t{script.ANONYMOUS, chType, nil},
                    script.Param_t{script.ANONYMOUS, []any{true, reader}, getStrFromReader},
                },
                []script.Result_t{
                    script.Result_t{"Reqchres", nil, script.ValidateNonNil}, /* chan to read resp */
                    script.NIL_ERROR, /*Expect nil error */
                },
                "Send a client request",
            },
            script.ScriptEntry_t{
                script.ApiIDReadClientResponse,
                []script.Param_t{
                    script.Param_t{"Reqchres", nil, nil},        /* Get from cache */
                    script.Param_t{script.ANONYMOUS, tout, nil}, /* timeout */
                },
                []script.Result_t{
                    script.Result_t{script.ANONYMOUS, []any{true, writer}, putStrToWriter}, /* send to writer */
                    script.NIL_ERROR, /*Expect nil error */
                },
                "Read client response",
            },
            script.ScriptEntry_t{
                script.ApiIDAny,
                []script.Param_t{
                    script.Param_t{"ReqLoopI", []int{0, cnt, 0}, script.LoopFn}},
                []script.Result_t{script.NIL_ERROR},
                "Loop cnt times",
            },
            script.ScriptEntry_t{
                script.ApiIDCloseRequestChannel,
                []script.Param_t{
                    script.Param_t{script.ANONYMOUS, chType, nil},
                },
                []script.Result_t{script.NIL_ERROR},
                "Close client req channel",
            },
            script.ScriptEntry_t{
                script.ApiIDDoSysShutdown,
                []script.Param_t{script.Param_t{script.ANONYMOUS, 2, nil}}, /* timeout = 2 secs */
                []script.Result_t{script.NIL_ERROR},
                "DoShutdown",
            },
        },
    }
    return
}

func GetResSuite(chType tele.ChannelType_t, cnt int, writer *bufio.Writer,
    reader *bufio.Reader, tout int) (ret *script.ScriptSuite_t, err error) {

    ret = &script.ScriptSuite_t{
        Id:          "ServerResponseLoop",
        Description: "Run a loop for server responder",
        Entries: []script.ScriptEntry_t{
            script.ScriptEntry_t{
                script.ApiIDRegisterServerReqHandler,
                []script.Param_t{
                    script.Param_t{script.ANONYMOUS, chType, nil},
                },
                []script.Result_t{
                    script.Result_t{"Reschreq", nil, script.ValidateNonNil}, /* chan to read resp */
                    script.Result_t{"Reschres", nil, script.ValidateNonNil}, /* chan to read resp */
                    script.NIL_ERROR, /*Expect nil error */
                },
                "Register a handler",
            },
            script.ScriptEntry_t{
                script.ApiIDReadClientRequest,
                []script.Param_t{
                    script.Param_t{"Reschreq", nil, nil},        /* Get from cache */
                    script.Param_t{script.ANONYMOUS, tout, nil}, /* timeout */
                },
                []script.Result_t{
                    script.Result_t{script.ANONYMOUS, []any{false, writer}, putStrToWriter}, /* send to writer */
                    script.NIL_ERROR, /*Expect nil error */
                },
                "Read client response",
            },
            script.ScriptEntry_t{
                script.ApiIDSendClientResponse,
                []script.Param_t{
                    script.Param_t{"Reschres", nil, nil}, /* Get from cache */
                    script.Param_t{script.ANONYMOUS, []any{false, reader}, getStrFromReader},
                    script.Param_t{script.ANONYMOUS, tout, nil}, /* timeout */
                },
                []script.Result_t{
                    script.NIL_ERROR, /*Expect nil error */
                },
                "Read client response",
            },
            script.ScriptEntry_t{
                script.ApiIDAny,
                []script.Param_t{
                    script.Param_t{"ResLoopI", []int{0, cnt, -2}, script.LoopFn}},
                []script.Result_t{script.NIL_ERROR},
                "Loop for cnt times previous 2 entries",
            },
            script.ScriptEntry_t{
                script.ApiIDCloseChannel,
                []script.Param_t{
                    script.Param_t{"Reschres", nil, nil},
                },
                []script.Result_t{script.NIL_ERROR},
                "Close server",
            },
            script.ScriptEntry_t{
                script.ApiIDDoSysShutdown,
                []script.Param_t{script.Param_t{script.ANONYMOUS, 2, nil}}, /* timeout = 2 secs */
                []script.Result_t{script.NIL_ERROR},
                "DoShutdown",
            },
        },
    }
    return
}
