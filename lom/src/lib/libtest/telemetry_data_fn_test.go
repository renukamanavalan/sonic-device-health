package libtest

import (
    cmn "lom/src/lib/lomcommon"
    script "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

var dataInPlay_1 = []tele.JsonString_t{
    tele.JsonString_t("foo"),
    tele.JsonString_t("bar"),
    tele.JsonString_t("hello"),
    tele.JsonString_t("world"),
    tele.JsonString_t("ok"),
}

func getValdataInPlay_1(name string, val any) (any, error) {
    return func(index int, cache script.SuiteCache_t) (*script.StreamingDataEntity_t, error) {
        if index >= len(dataInPlay_1) {
            return nil, cmn.LogError("index(%d) len(%d) out-of-range", index, len(dataInPlay_1))
        }
        /* Set in cache all returned values including this index */
        cache.SetVal(name, dataInPlay_1[:index+1])
        return &script.StreamingDataEntity_t{dataInPlay_1[index : index+1], index < len(dataInPlay_1)-1}, nil
    }, nil
}

func putValdataInPlay_1(name string, val any) (any, error) {
    return func(index int, data tele.JsonString_t, cache script.SuiteCache_t) (
        bool, error) {
        if index >= len(dataInPlay_1) {
            return false, cmn.LogError("index(%d) len(%d) out-of-range", index, len(dataInPlay_1))
        }
        if data != dataInPlay_1[index] {
            return false, cmn.LogError("index(%d) data mismatch (%s) != (%s)", index, data, dataInPlay_1[index])
        }
        return index < len(dataInPlay_1)-1, nil
    }, nil
}

var pubSubFnSuite = testSuite_t{
    id:          "pubSubFnSuite",
    description: "Test pub sub for events - Good run",
    tests: []testEntry_t{
        testEntry_t{
            script.ApiIDRunPubSubProxy,
            []script.Param_t{script.Param_t{"chType_1", tele.CHANNEL_TYPE_EVENTS, nil}},
            []result_t{
                result_t{"chPrxyClose-0", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR, /*Expect nil error */
            },
            "Rub pubsub proxy, required to bind publishers & subscribers",
        },
        testEntry_t{
            script.ApiIDGetSubChannel,
            []script.Param_t{
                script.Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                script.Param_t{"prod_0", tele.CHANNEL_PRODUCER_EMPTY, nil},
                EMPTY_STRING,
            },
            []result_t{
                result_t{"chRead-0", nil, validateNonNil},     /* Save in cache */
                result_t{"chSubClose-0", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get sub channel for same type as proxy above",
        },
        testEntry_t{
            script.ApiIDGetPubChannel,
            []script.Param_t{
                script.Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                script.Param_t{"prod_1", tele.CHANNEL_PRODUCER_ENGINE, nil},
                EMPTY_STRING,
            },
            []result_t{
                result_t{"chWrite-0", nil, validateNonNil}, /* Save in cache */
                result_t{script.ANONYMOUS, nil, validateNil},
            },
            "Get pub channel for same type as proxy above",
        },
        testEntry_t{
            script.ApiIDWriteJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"chWrite-0", nil, nil},            /* Use chan from cache */
                script.Param_t{"pub_0", nil, getValdataInPlay_1}, /* Save written data in cache */
                script.Param_t{script.ANONYMOUS, 1, nil},         /* timeout = 1 second */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil},
            }, /*Expect nil error */
            "Write into pub channel created above",
        },
        testEntry_t{
            script.ApiIDReadJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"chRead-0", nil, nil},     /* Get chRead_0 from cache */
                script.Param_t{script.ANONYMOUS, 5, nil}, /* read cnt = 5 */
                script.Param_t{script.ANONYMOUS, 1, nil}, /* timeout = 1 second */
            },
            []result_t{
                result_t{"pub_0", nil, nil}, /* Validate against cache val for pub_0 */
                result_t{script.ANONYMOUS, nil, validateNil},
            },
            "read from sub channel created above",
        },
        testEntry_t{
            script.ApiIDWriteJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"chWrite-0", nil, nil},               /* Use chan from cache */
                script.Param_t{script.ANONYMOUS, dataInPlay_1, nil}, /* Save written data in cache */
                script.Param_t{script.ANONYMOUS, 1, nil},            /* timeout = 1 second */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil},
            }, /*Expect nil error */
            "Write into pub channel created above",
        },
        testEntry_t{
            script.ApiIDReadJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"chRead-0", nil, nil},                      /* Get chRead_0 from cache */
                script.Param_t{script.ANONYMOUS, nil, putValdataInPlay_1}, /* read into fn*/
                script.Param_t{script.ANONYMOUS, 1, nil},                  /* timeout = 1 second */
            },
            []result_t{
                result_t{script.ANONYMOUS, []tele.JsonString_t{}, nil}, /* Validate against cache val for pub_0 */
                result_t{script.ANONYMOUS, nil, validateNil},
            },
            "read from sub channel created above",
        },
        testEntry_t{
            script.ApiIDCloseChannel,
            []script.Param_t{
                script.Param_t{"chWrite-0", nil, nil}, /* Get chWrite_0 from cache */
            },
            []result_t{NIL_ERROR},
            "Close pub chennel",
        },
        testEntry_t{
            script.ApiIDCloseChannel,
            []script.Param_t{
                script.Param_t{"chSubClose-0", nil, nil}, /* Get from cache */
            },
            []result_t{NIL_ERROR},
            "Close sub chennel",
        },
        testEntry_t{
            script.ApiIDCloseChannel,
            []script.Param_t{
                script.Param_t{"chPrxyClose-0", nil, nil}, /* Get from cache */
            },
            []result_t{NIL_ERROR},
            "Close proxy chennel",
        },
        PAUSE2,
        TELE_IDLE_CHECK,
    },
}
