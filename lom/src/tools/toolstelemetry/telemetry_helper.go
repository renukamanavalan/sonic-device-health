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


func getFromReader(name string, val any) (ret any, err error) {
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
        } else if name != script.ANONYMOUS  {
            cache.SetVal(name, append(getSlice(name, cache), teleTxt))
        }
        return &script.StreamingDataEntity_t{[]tele.JsonString_t{teleTxt}, more}, err
    }
    return
}

func putIntoWriter(name string, val any) (ret any, err error) {
    writer, ok := val.(*bufio.Writer)
    if !ok || (writer == nil) {
        err = cmn.LogError("Val incorrect type (%T) != *bufio.Writer", val)
        return
    }

    ret = func(_ int, data tele.JsonString_t, cache script.SuiteCache_t) (
        more bool, err error) {
        if _, err = writer.WriteString(string(data)+"\n"); err == nil {
            err = writer.Flush()
        }
        if err == nil {
            more = true
            if name != script.ANONYMOUS  {
                cache.SetVal(name, append(getSlice(name, cache), data))
            }
        }
        return
    }
    return
}

func GetPubSuite(chType tele.ChannelType_t, chProd tele.ChannelProducer_t,
                    reader *bufio.Reader) (ret *script.ScriptSuite_t, err error) {

    if reader == nil {
        err = cmn.LogError("Expect non nil *bufio.Reader")
        return
    }
    ret = &script.ScriptSuite_t {
        Id          : "pubFromStdin",
        Description : "Read a line from stdin & publish until EOF",
        Entries     : []script.ScriptEntry_t {
            script.ScriptEntry_t {
                script.ApiIDGetPubChannel,
                []script.Param_t{
                    script.Param_t{"chType", chType, nil },
                    script.Param_t{"chProd", chProd, nil },
                    script.Param_t{script.ANONYMOUS, "CliTool", nil},
                },
                []script.Result_t {
                    script.Result_t{"chPub-0", nil, script.ValidateNonNil}, /* Save in cache */
                    script.Result_t{script.ANONYMOUS, nil, script.ValidateNil},
                },
                "Get pub channel for same type as proxy above",
            },
            script.ScriptEntry_t {
                script.ApiIDWriteJsonStringsChannel,
                []script.Param_t{
                    script.Param_t{"chPub-0", nil, nil },   /* From cache */
                    script.Param_t{script.ANONYMOUS, reader, getFromReader },
                    script.Param_t{script.ANONYMOUS, 1, nil}, /* timeout = 1 second */
                },
                []script.Result_t { script.NIL_ERROR },
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


func GetSubSuite(chType tele.ChannelType_t, chProd tele.ChannelProducer_t,
                    writer *bufio.Writer) (ret *script.ScriptSuite_t, err error) {

    if writer == nil {
        err = cmn.LogError("Expect non nil *bufio.Writer")
        return
    }
    ret = &script.ScriptSuite_t {
        Id          : "subIntoStdout",
        Description : "Write data fron sub channel to stdout",
        Entries     : []script.ScriptEntry_t {
            script.ScriptEntry_t {
                script.ApiIDGetSubChannel,
                []script.Param_t{
                    script.Param_t{"chType", chType, nil },
                    script.Param_t{"chProd", chProd, nil },
                    script.Param_t{script.ANONYMOUS, "CliTool", nil},
                },
                []script.Result_t {
                    script.Result_t{"chSub-0", nil, script.ValidateNonNil}, /* Save in cache */
                    script.Result_t{"chClose-0", nil, script.ValidateNonNil}, /* Save in cache */
                    script.Result_t{script.ANONYMOUS, nil, script.ValidateNil},
                },
                "Get Sub channel for same type as proxy above",
            },
            script.ScriptEntry_t {
                script.ApiIDReadJsonStringsChannel,
                []script.Param_t{
                    script.Param_t{"chSub-0", nil, nil },   /* From cache */
                    script.Param_t{script.ANONYMOUS, writer, putIntoWriter },
                    script.Param_t{script.ANONYMOUS, 100, nil}, /* timeout = 100 second */
                },
                []script.Result_t { script.NIL_ANY, script.NIL_ERROR },
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

    ret = &script.ScriptSuite_t {
        Id          : "RunProxy",
        Description : "Run Proxy for given timeout",
        Entries     : []script.ScriptEntry_t {
            script.ScriptEntry_t {
                script.ApiIDRunPubSubProxy,
                []script.Param_t{ script.Param_t{script.ANONYMOUS, chType, nil } },
                []script.Result_t {
                    script.Result_t{"chPrxyClose-0", nil, script.ValidateNonNil}, /* Save in cache */
                    script.NIL_ERROR, /*Expect nil error */
                },
                "Get pubsub proxy, required to bind publishers & subscribers",
            },
            script.ScriptEntry_t {
                script.ApiIDPause,
                []script.Param_t{ script.Param_t{script.ANONYMOUS, tout, nil}},
                []script.Result_t { script.NIL_ERROR },
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


