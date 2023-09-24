package libtest

import (
    "fmt"
    script "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

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

var NIL_ERROR = result_t{script.ANONYMOUS, nil, validateNil}

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

func validateNil(n string, vExp, vRet any) bool {
    if vRet == nil {
        testLastErr = ""
        return true
    }
    testLastErr = fmt.Sprintf("name=%s Expect nil. type(%T)(%v)", n, vRet, vRet)
    return false
}

func validateNonNil(n string, vExp, vRet any) bool {
    if vRet != nil {
        testLastErr = ""
        return true
    }
    testLastErr = fmt.Sprintf("name=%s Expect non nil, but nil", n)
    return false
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
                NIL_ERROR,  /*Expect nil error */
            },
            "Rub pubsub proxy, required to bind publishers & subscribers",
        },
        testEntry_t{
            script.ApiIDGetSubChannel,
            []script.Param_t{
                script.Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                script.Param_t{"prod_0", tele.CHANNEL_PRODUCER_EMPTY, nil},
                script.Param_t{script.ANONYMOUS, "", nil},
            },
            []result_t{
                result_t{"chRead-0", nil, validateNonNil}, /* Save in cache */
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
                script.Param_t{script.ANONYMOUS, "", nil},
            },
            []result_t{
                result_t{"chWrite-0", nil, validateNonNil}, /* Save in cache */
                result_t{script.ANONYMOUS, nil, validateNil},
            },
            "Get pub channel for same type as proxy above",
        },
        testEntry_t{
            script.ApiIDWriteChannel,
            []script.Param_t{
                script.Param_t{"chWrite-0", nil, nil},                           /* Use chan from cache */
                script.Param_t{"pub_0", tele.JsonString_t("Hello World!"), nil}, /* Save written data in cache */
                script.Param_t{script.ANONYMOUS, 1, nil},                        /* timeout = 1 second */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil},
            }, /*Expect nil error */
            "Write into pub channel created above",
        },
        testEntry_t{
            script.ApiIDReadChannel,
            []script.Param_t{
                script.Param_t{"chRead-0", nil, nil},     /* Get chRead_0 from cache */
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
                script.Param_t{"chWrite-0", nil, nil},  /* Get chWrite_0 from cache */
            },
            []result_t{ NIL_ERROR },
            "Close pub chennel",
        },
        testEntry_t{
            script.ApiIDCloseChannel,
            []script.Param_t{
                script.Param_t{"chSubClose-0", nil, nil},  /* Get from cache */
            },
            []result_t{ NIL_ERROR },
            "Close pub chennel",
        },
        testEntry_t{
            script.ApiIDCloseChannel,
            []script.Param_t{
                script.Param_t{"chPrxyClose-0", nil, nil},  /* Get from cache */
            },
            []result_t{ NIL_ERROR },
            "Close proxy chennel",
        },
    },
}

var testTelemetrySuites = []*testSuite_t{
    &pubSubSuite,
}
