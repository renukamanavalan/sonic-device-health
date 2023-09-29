package libtest

import (
    "fmt"
    cmn "lom/src/lib/lomcommon"
    script "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

/*
 * Data driven test FW.
 *
 * A test entry {
 *  Identifies API by API ID
 *  Each arg is represented by param_t struct
 *  Each return value is expressed by result_t struct
 *
 * Named param or result entity is saved in cache.
 * Subseqent param/result could refer value from the cache.
 * A cache is per test suite
 *
 * A test suite is a collection of tests.
 *
 */

/* Test Data for telemetry */
type validatorFn_t func(name string, ValExpect, valRet any) bool

type result_t struct {
    /*
     * if val_expect != nil or name could fetch a non-nil value, it
     * is expected to match the returned result from the call.
     * if non nil validator it is invoked additionally to validate.
     * Upon successful/no validation, if name is non-empty, the returned value
     * is set as new value.
     */
    name      string /* Assign name to this var. Can be script.ANONYMOUS. */
    valExpect any    /* Expected Value of the var. */
    validator validatorFn_t
}

type testEntry_t struct {
    api     script.ApiId_t
    args    []script.Param_t
    result  []result_t
    message string
}

type testSuite_t struct {
    id          string /* Keep it cryptic as it appears in error messages */
    description string /* Give full details for any human reader */
    tests       []testEntry_t
}

var currentCache script.SuiteCache_t

func resetTestCache() {
    currentCache = nil
}

func setTestCache(cache script.SuiteCache_t) {
    currentCache = cache
}

func getTestCache() script.SuiteCache_t {
    return currentCache
}

/* Commonly used entities are pre declared for ease of use */
var EMPTY_STRING = script.Param_t{script.ANONYMOUS, "", nil}

var NIL_ANY = result_t{script.ANONYMOUS, nil, validateNil}
var NIL_ERROR = NIL_ANY
var NON_NIL_ERROR = result_t{script.ANONYMOUS, nil, validateNonNil}
var TEST_FOR_TRUE = result_t{script.ANONYMOUS, true, nil}
var TEST_FOR_FALSE = result_t{script.ANONYMOUS, false, nil}
var PAUSE1 = testEntry_t{ /* Pause for 1 seconds */
    script.ApiIDPause,
    []script.Param_t{script.Param_t{script.ANONYMOUS, 1, nil}},
    []result_t{NIL_ERROR},
    "Pause for 1 seconds",
}

var PAUSE2 = testEntry_t{ /* Pause for 2 seconds */
    script.ApiIDPause,
    []script.Param_t{script.Param_t{script.ANONYMOUS, 2, nil}},
    []result_t{NIL_ERROR},
    "Pause for 2 seconds",
}

var TELE_IDLE_CHECK = testEntry_t{
    script.ApiIDIsTelemetryIdle,
    []script.Param_t{},
    []result_t{TEST_FOR_TRUE, NIL_ERROR},
    "Test if no telemetry channels are open",
}

/* String returned by last validation function */
var testLastErr = ""

func validateNonNilError(n string, vExp, vRet any) bool {
    switch vRet.(type) {
    case error:
        /* Non nil error as expected. Hence clear any last error */
        testLastErr = ""
        return true
    default:
        testLastErr = fmt.Sprintf("name(%s) expect Non nil error. type(%T)", n, vRet)
        return false
    }
}

var emptyVals = map[string]bool{
    "<nil>": true,
    "{}":    true,
    "[]":    true,
    "":      true,
}

func checkNil(n string, vRet any, expNil bool) bool {
    if _, ok := emptyVals[fmt.Sprintf("%v", vRet)]; ok == expNil {
        cmn.LogDebug("validate for nil(%v) succeeded n(%s) vRet(%v)(%T)", expNil, n, vRet, vRet)
        return true
    }
    cmn.LogError("validate for nil(%v) failed n(%s) vRet(%v)(%T)", expNil, n, vRet, vRet)
    return false
}

func validateNil(n string, vExp, vRet any) bool {
    return checkNil(n, vRet, true)
}

func validateNonNil(n string, vExp, vRet any) bool {
    return checkNil(n, vRet, false)
}

var pubSubSuite = testSuite_t{
    id:          "pubSubSuite",
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
                script.Param_t{"chWrite-0", nil, nil}, /* Use chan from cache */
                script.Param_t{"pub_0", []tele.JsonString_t{
                    tele.JsonString_t("Hello World!")}, nil}, /* Save written data in cache */
                script.Param_t{script.ANONYMOUS, 1, nil}, /* timeout = 1 second */
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
                script.Param_t{script.ANONYMOUS, 1, nil}, /* read cnt = 1 */
                script.Param_t{script.ANONYMOUS, 1, nil}, /* timeout = 1 second */
            },
            []result_t{
                result_t{"pub_0", nil, nil}, /* Validate against cache val for pub_0 */
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

var pubSubMultiSuite = testSuite_t{
    id:          "pubSubMultiSuite",
    description: "Test multi pub sub for events - Good run",
    tests: []testEntry_t{
        testEntry_t{
            script.ApiIDRunPubSubProxy,
            []script.Param_t{script.Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}},
            []result_t{
                result_t{"chPrxyClose-C", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR, /*Expect nil error */
            },
            "Rub pubsub proxy, required to bind publishers & subscribers",
        },
        testEntry_t{
            script.ApiIDRunPubSubProxy,
            []script.Param_t{script.Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil}},
            []result_t{
                result_t{"chPrxyClose-E", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR, /*Expect nil error */
            },
            "Rub pubsub proxy, required to bind publishers & subscribers",
        },
        testEntry_t{ /* Get sub channel for events from engine only. */
            script.ApiIDGetSubChannel,
            []script.Param_t{
                script.Param_t{"chType_E", nil, nil}, /* Fetch chType_1 from cache */
                script.Param_t{"prod_E", tele.CHANNEL_PRODUCER_ENGINE, nil},
                EMPTY_STRING,
            },
            []result_t{
                result_t{"chRead-E", nil, validateNonNil},     /* Save in cache */
                result_t{"chSubClose-E", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get sub channel for events from Engine",
        },
        testEntry_t{ /* Get sub channel for counters from a plugin-mgr instance */
            script.ApiIDGetSubChannel,
            []script.Param_t{
                script.Param_t{"chType_C", nil, nil}, /* Fetch from cache */
                script.Param_t{"prod_PM", tele.CHANNEL_PRODUCER_PLMGR, nil},
                script.Param_t{"PMgr-1", "inst-1", nil},
            },
            []result_t{
                result_t{"chRead-C", nil, validateNonNil},     /* Save in cache */
                result_t{"chSubClose-C", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get sub channel for events from Engine",
        },
        testEntry_t{
            script.ApiIDGetPubChannel, /* Simulate publish from plugin-mgr instance */
            []script.Param_t{
                script.Param_t{"chType_C", nil, nil}, /* pub for counters */
                script.Param_t{"prod_PM", nil, nil},  /* from Plugin Mgr */
                script.Param_t{"PMgr-1", nil, nil},   /* instance-1 */
            },
            []result_t{
                result_t{"chWrite-C", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get pub channel for counters as if from Plugin Mgr",
        },
        testEntry_t{
            script.ApiIDGetPubChannel, /* Simulate publish from engine */
            []script.Param_t{
                script.Param_t{"chType_E", nil, nil}, /* pub for events */
                script.Param_t{"prod_E", nil, nil},   /* from engine */
                EMPTY_STRING,
            },
            []result_t{
                result_t{"chWrite-E", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get pub channel for counters as if from Plugin Mgr",
        },
        testEntry_t{
            script.ApiIDWriteJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"chWrite-E", nil, nil}, /* Use chan from cache */
                script.Param_t{"pub_E", []tele.JsonString_t{
                    tele.JsonString_t("Hello World!")}, nil}, /* Save written data in cache */
                script.Param_t{script.ANONYMOUS, 1, nil}, /* timeout = 1 second */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil},
            }, /*Expect nil error */
            "Write into pub channel created above",
        },
        testEntry_t{
            script.ApiIDWriteJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"chWrite-C", nil, nil}, /* Use chan from cache */
                script.Param_t{"pub_C", []tele.JsonString_t{
                    tele.JsonString_t("Some counters")}, nil}, /* Save written data in cache */
                script.Param_t{script.ANONYMOUS, 1, nil}, /* timeout = 1 second */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil},
            }, /*Expect nil error */
            "Write into pub channel created above",
        },
        testEntry_t{
            script.ApiIDReadJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"chRead-C", nil, nil},     /* read counters */
                script.Param_t{script.ANONYMOUS, 1, nil}, /* read cnt = 1 */
                script.Param_t{script.ANONYMOUS, 1, nil}, /* timeout = 1 second */
            },
            []result_t{
                result_t{"pub_C", nil, nil}, /* Validate against cache val for pub_C */
                result_t{script.ANONYMOUS, nil, validateNil},
            },
            "read from sub channel created above",
        },
        testEntry_t{
            script.ApiIDReadJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"chRead-E", nil, nil},     /* read counters */
                script.Param_t{script.ANONYMOUS, 1, nil}, /* read cnt = 1 */
                script.Param_t{script.ANONYMOUS, 1, nil}, /* timeout = 1 second */
            },
            []result_t{
                result_t{"pub_E", nil, nil}, /* Validate against cache val for pub_E */
                result_t{script.ANONYMOUS, nil, validateNil},
            },
            "read from sub channel created above",
        },
        testEntry_t{
            script.ApiIDCloseChannel,
            []script.Param_t{
                script.Param_t{"chPrxyClose-C", nil, nil},
                script.Param_t{"chPrxyClose-E", nil, nil},
                script.Param_t{"chSubClose-E", nil, nil},
                script.Param_t{"chSubClose-C", nil, nil},
                script.Param_t{"chWrite-C", nil, nil},
                script.Param_t{"chWrite-E", nil, nil},
            },
            []result_t{NIL_ERROR},
            "Close pub chennel",
        },
        PAUSE2,
        TELE_IDLE_CHECK,
    },
}

var testTelemetrySuites = []*testSuite_t{
    &pubSubSuite,
    &pubSubMultiSuite,
    &pubSubFnSuite,
    &pubSubReqRepSuite,
    &pubSubFailSuite,
    &scriptAPIValidate,
    &scriptAPIValidate_2,
    &pubSubBindFail,
    &pubSubShutdownSuite, /* KEEP this as last suite as it invokes irreversible shutdown */
}
