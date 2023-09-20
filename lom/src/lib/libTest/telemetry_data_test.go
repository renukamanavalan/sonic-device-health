package libTest

import (
    "fmt"
    tele "lom/src/lib/lomtelemetry"
)

/* Test Data for telemetry */

type apiId_t string

const (
    ApiIDGetPubChannel            apiId_t = "GetPubChannel"
    ApiIDGetSubChannel                    = "GetSubChannel"
    ApiIDRunPubSubProxy                   = "RunPubSubProxy"
    ApiIDSendClientRequest                = "SendClientRequest"
    ApiIDRegisterServerReqHandler         = "RegisterServerReqHandler"
    ApiIDDoSysShutdown                    = "DoSysShutdown"
    ApiIDWriteChannel                     = "WriteChannel"
    ApiIDReadChannel                      = "ReadChannel"
    ApiIDCloseChannel                     = "CloseChannel"
    ApiIDPause                            = "pause"
)

/* Caches named variable among tests in a single suite */
type suiteCache_t map[string]any

const ANONYMOUS = ""

func (s suiteCache_t) getVal(name string, val any) (any, bool) {
    if name == ANONYMOUS {
        return val, true /* Anonymous */
    } else if val != nil {
        s.setVal(name, val)          /* overwrite */
        return val, true
    } else if ct, ok := s[name]; !ok {
        return nil, false
    } else {
        return ct, true
    }
}

func (s suiteCache_t) setVal(name string, val any) {
    if name != ANONYMOUS {
        if val != nil {
            s[name] = val /* Set it */
        } else if _, ok := s[name]; ok {
            delete(s, name)
        }
    }
}

type param_t struct {
    name string /* Assign name to this var */
    val  any    /* Val of this var */
    /* If nil expect this var to pre-exist in cache. */
}

type result_t struct {
    /*
     * if val_expect != nil or name could fetch a non-nil value, it
     * is expected to match the returned result from the call.
     * if non nil validator it is invoked additionally to validate.
     * Upon successful/no validation, if name is non-empty, the returned value
     * is set as new value.
     */
    name      string /* Assign name to this var. Can be anonymous. */
    valExpect any    /* Expected Value of the var. */
    validator func(name string, ValExpect, valRet any) bool
}

type testEntry_t struct {
    api     apiId_t
    args    []param_t
    result  []result_t
    message string
}

type testSuite_t struct {
    id    string
    tests []testEntry_t
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
    id: "Test pub sub for events - Good run",
    tests: []testEntry_t{
        testEntry_t{
            ApiIDRunPubSubProxy,
            []param_t{param_t{"chType_1", tele.CHANNEL_TYPE_EVENTS}},
            []result_t{result_t{ANONYMOUS, nil, validateNil}}, /*Expect nil error */
            "Failed to run sub proxy",
        },
        testEntry_t{
            ApiIDGetSubChannel,
            []param_t{
                param_t{"chType_1", nil}, /* Fetch chType_1 from cache */
                param_t{"prod_0", tele.CHANNEL_PRODUCER_EMPTY},
                param_t{ANONYMOUS, ""},
            },
            []result_t{
                result_t{"chRead-0", nil, validateNonNil}, /* Save in cache */
                result_t{ANONYMOUS, nil, validateNil},
            },
            "Failed to Get sub channel",
        },
        testEntry_t {
            ApiIDGetPubChannel, 
            []param_t {
                param_t { "chType_1", nil },        /* Fetch chType_1 from cache */
                param_t { "prod_1", tele.CHANNEL_PRODUCER_ENGINE },
                param_t { ANONYMOUS, "" },
            },
            []result_t {
                result_t { "chWrite-0", nil, validateNonNil}, /* Save in cache */
                result_t { ANONYMOUS, nil, validateNil },
            },
            "Failed to Get pub channel",
        },
        testEntry_t {
            ApiIDWriteChannel, 
            []param_t {
                param_t { "chWrite-0", nil },       /* Use chan from cache */
                param_t { "pub_0", tele.JsonString_t("Hello World!") },/* Save written data in cache */
                param_t { ANONYMOUS, 1 },           /* timeout = 1 second */
            },
            []result_t {
                result_t {ANONYMOUS, nil, validateNil },
            }, /*Expect nil error */
            "Failed to write pub channel",
        },
        testEntry_t {
            ApiIDReadChannel, 
            []param_t {
                param_t { "chRead-0", nil },        /* Get chRead_0 from cache */
                param_t { ANONYMOUS, 1 },           /* timeout = 1 second */
            },
            []result_t {
                result_t { "pub_0", nil, nil },     /* Validate against cache val for pub_0 */
                result_t {ANONYMOUS, nil, validateNil },
            },
            "Failed to read sub channel",
        },
    },
}

var testTelemetrySuites = []*testSuite_t{
    &pubSubSuite,
}
