package main

import (
    "bufio"
    "fmt"
    cmn "lom/src/lib/lomcommon"
    script "lom/src/lib/lomscripted"
)

func getFromReader(name string, val any) (ret any, err error) {
    var data []string
    reader, ok := val.(*bufio.Reader)
    if !ok || (reader == nil) {
        err = cmn.LogError("Val incorrect type (%T) != *bufio.Readerr", val)
        return
    }
    more := true

    ret = func(_ int, cache SuiteCache_t) (*StreamingDataEntity_t, error) {
        text, err := reader.ReadString('\n')
        if err == nil {
            text = strings.TrimSpace(text)
        } else {
            text = ""
        }
        if text == "" {
            more = false
        } else if name != script.ANONYMOUS  {
            data = append(data, text)
            cache.SetVal(name, data)
        }
        return &StreamingDataEntity_t{[]tele.JsonString_t{tele.JsonString_t(text)}, more}, err
    }
    return
}

func GetPubSuite(chType ChannelType_t, chProd ChannelProducer_t,
                    reader *bufio.Reader) (ret *ScriptSuite_t, err error) {

    if reader == nil {
        err = cmn.LogError("Expect non nil *bufio.Reader")
        return
    }
    ret = &ScriptSuite_t = {
        id          : "pubFromStdin",
        description : "Read a line from stdin & publish until EOF",
        entries     : []ScriptEntry_t {
            ScriptEntry_t {
                script.ApiIDGetPubChannel,
                []script.Param_t{
                    script.Param_t{"chType", chType, nil },
                    script.Param_t{"chProd", chProd, nil },
                },
                []result_t {
                    result_t{"chPub-0", nil, validateNonNil}, /* Save in cache */
                    result_t{script.ANONYMOUS, nil, validateNil},
                },
                "Get pub channel for same type as proxy above",
            },
            ScriptEntry_t {
                script.ApiIDWriteJsonStringsChannel,
                []script.Param_t{
                    script.Param_t{"chPub-0", nil, nil },   /* From cache */
                    script.Param_t{script.ANONYMOUS, reader, getFromReader },
                },
                []result_t { script.NIL_ERROR },
                "Get pub channel for same type as proxy above",
            },
            ScriptEntry_t{
                script.ApiIDCloseChannel,
                []script.Param_t{
                    script.Param_t{"chPub-0", nil, nil}, /* Get from cache */
                },
                []result_t{NIL_ERROR},
                "Close pub chennel",
            },
        },
    }
    return
}


