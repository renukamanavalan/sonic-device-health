package lomscripted

import (
    "fmt"
    cmn "lom/src/lib/lomcommon"
    tele "lom/src/lib/lomtelemetry"
)

/*
 * Helper to create a suite of entries and run.
 *
 * Validation may be added to abort the suite on a failing entry.
 * A entry {
 *  Identifies API by API ID
 *  Each arg is represented by param_t struct
 *  Each return value is expressed by Result_t struct
 *
 * Named param or result entity is saved in cache.
 * Subseqent param/result could refer value from the cache.
 * A cache is per test suite
 *
 * A suite is a collection of entries.
 *
 */

type ValidatorFn_t func(name string, ValExpect, valRet any) bool

type Result_t struct {
    /*
     * if val_expect != nil or name could fetch a non-nil value, it
     * is expected to match the returned result from the call.
     * if non nil validator it is invoked additionally to validate.
     * Upon successful/no validation, if name is non-empty, the returned value
     * is set as new value.
     */
    Name      string /* Assign name to this var. Can be ANONYMOUS. */
    ValExpect any    /* Expected Value of the var. */
    Validator ValidatorFn_t
}

type ScriptEntry_t struct {
    Api     ApiId_t
    Args    []Param_t
    Result  []Result_t
    Message string
}

type ScriptSuite_t struct {
    Id          string /* Keep it cryptic as it appears in error messages */
    Description string /* Give full details for any human reader */
    Entries     []ScriptEntry_t
}

/* Commonly used entities are pre declared for ease of use */
var EMPTY_STRING = Param_t{ANONYMOUS, "", nil}

var NIL_ANY = Result_t{ANONYMOUS, nil, ValidateNil}
var NIL_ERROR = NIL_ANY
var NON_NIL_ERROR = Result_t{ANONYMOUS, nil, ValidateNonNil}
var TEST_FOR_TRUE = Result_t{ANONYMOUS, true, nil}
var TEST_FOR_FALSE = Result_t{ANONYMOUS, false, nil}
var PAUSE1 = ScriptEntry_t{ /* Pause for 1 seconds */
    ApiIDPause,
    []Param_t{Param_t{ANONYMOUS, 1, nil}},
    []Result_t{NIL_ERROR},
    "Pause for 1 seconds",
}

var PAUSE2 = ScriptEntry_t{ /* Pause for 2 seconds */
    ApiIDPause,
    []Param_t{Param_t{ANONYMOUS, 2, nil}},
    []Result_t{NIL_ERROR},
    "Pause for 2 seconds",
}

var TELE_IDLE_CHECK = ScriptEntry_t{
    ApiIDIsTelemetryIdle,
    []Param_t{},
    []Result_t{TEST_FOR_TRUE, NIL_ERROR},
    "Test if no telemetry channels are open",
}

func ValidateNonNilError(n string, vExp, vRet any) bool {
    switch vRet.(type) {
    case error:
        /* Non nil error as expected. Hence clear any last error */
        return true
    default:
        cmn.LogError("name(%s) expect Non nil error. type(%T)", n, vRet)
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

func ValidateNil(n string, vExp, vRet any) bool {
    return checkNil(n, vRet, true)
}

func ValidateNonNil(n string, vExp, vRet any) bool {
    return checkNil(n, vRet, false)
}

var currentCache SuiteCache_t
                    
func ResetSuiteCache() SuiteCache_t { 
    currentCache = SuiteCache_t{}
    return currentCache
}               
            
func GetSuiteCache() SuiteCache_t {
    return currentCache
}

func RunOneScriptSuite(suite *ScriptSuite_t) (err error) {
    /* Caches all variables for reference across script entries */
    cache := ResetSuiteCache()  /* Ensure new */
    defer ResetSuiteCache()     /* Clean up cache */

    for i, entry := range suite.Entries {
        if retVals, ok := CallByApiID(entry.Api, entry.Args, cache); !ok {
            err = cmn.LogError("Failed to find API (%v)", entry.Api)
        } else if len(retVals) != len(entry.Result) {
            err = cmn.LogError("Return length (%d) != expected (%d)", len(retVals), len(entry.Result))
        } else {
            for j, e := range entry.Result {
                /*
                 * For each try to Getval.
                 * If non nil validator fn exists, it dictates.
                 * Else compare read value from GetVal with returned value
                 */
                retV := retVals[j]
                expVal := cache.GetVal(e.Name, e.ValExpect, nil)
                if e.Validator != nil {
                    if e.Validator(e.Name, expVal, retV) == false {
                        err = cmn.LogError("Result validation failed suite-index(%d) res-index(%d) retv(%+v)",
                            i, j, retV)
                        retV = nil
                    }
                } else {
                    switch expVal.(type) {
                    case []tele.JsonString_t:
                        expL := expVal.([]tele.JsonString_t)
                        if retL, ok := retV.([]tele.JsonString_t); !ok {
                            err = cmn.LogError("ExpVal(%T) != RetV(%T)", expVal, retV)
                        } else if len(expL) != len(retL) {
                            err = cmn.LogError("len Mismatch ExpVal (%d) != retVal (%d)",
                                len(expL), len(retL))
                        } else {
                            for i, e := range expL {
                                if e != retL[i] {
                                    err = cmn.LogError("val Mismatch index(%d) (%s) != (%s)",
                                        e, retL[i])
                                }
                            }
                        }
                    default:
                        if expVal != retV {
                            err = cmn.LogError("ExpVal(%v) != RetV(%v)(%T)", expVal, retV, retV)
                        }
                    }
                }
                cache.SetVal(e.Name, retV)
            }
        }
        if err != nil {
            break
        }
    }
    return
}
