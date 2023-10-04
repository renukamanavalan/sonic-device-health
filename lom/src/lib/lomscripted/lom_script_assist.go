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
 *  Each return value is expressed by result_t struct
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
    Validator validatorFn_t
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

var NIL_ANY = result_t{ANONYMOUS, nil, validateNil}
var NIL_ERROR = NIL_ANY
var NON_NIL_ERROR = result_t{ANONYMOUS, nil, validateNonNil}
var TEST_FOR_TRUE = result_t{ANONYMOUS, true, nil}
var TEST_FOR_FALSE = result_t{ANONYMOUS, false, nil}
var PAUSE1 = ScriptEntry_t{ /* Pause for 1 seconds */
    ApiIDPause,
    []Param_t{Param_t{ANONYMOUS, 1, nil}},
    []result_t{NIL_ERROR},
    "Pause for 1 seconds",
}

var PAUSE2 = ScriptEntry_t{ /* Pause for 2 seconds */
    ApiIDPause,
    []Param_t{Param_t{ANONYMOUS, 2, nil}},
    []result_t{NIL_ERROR},
    "Pause for 2 seconds",
}

var TELE_IDLE_CHECK = ScriptEntry_t{
    ApiIDIsTelemetryIdle,
    []Param_t{},
    []result_t{TEST_FOR_TRUE, NIL_ERROR},
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

func CheckNil(n string, vRet any, expNil bool) bool {
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

var currentCache script.SuiteCache_t
                    
func ResetSuiteCache() { 
    currentCache = script.SuiteCache_t{}
}               
            
func GetSuiteCache() script.SuiteCache_t {
    return currentCache
}

func RunOneScriptSuite(suite *ScriptSuite_t) (err error) {
    /* Caches all variables for reference across script entries */
    ResetSuiteCache()           /* Ensure new */
    defer ResetSuiteCache()     /* Clean up cache */

    for i, entry := range suite.Entries {
        if retVals, ok := script.CallByApiID(entry.api, entry.args, cache); !ok {
            err = cmn.LogError("Failed to find API (%v)", entry.api)
        } else if len(retVals) != len(entry.result) {
            err = cmn.LogError("%s: Return length (%d) != expected (%d)", tid, len(retVals), len(entry.result))
        } else {
            for j, e := range entry.result {
                /*
                 * For each try to Getval.
                 * If non nil validator fn exists, it dictates.
                 * Else compare read value from GetVal with returned value
                 */
                retV := retVals[j]
                expVal := cache.GetVal(e.name, e.valExpect, nil)
                if e.validator != nil {
                    if e.validator(e.name, expVal, retV) == false {
                        err = cmn.LogError("Result validation failed suite-index(%d) res-index(%d) retv(%+v)",
                            i, j, retV)
                        retV = nil
                    }
                } else {
                    switch expVal.(type) {
                    case []tele.JsonString_t:
                        expL := expVal.([]tele.JsonString_t)
                        if retL, ok := retV.([]tele.JsonString_t); !ok {
                            err = cmn.LogError("%s: ExpVal(%T) != RetV(%T)", tid, expVal, retV)
                        } else if len(expL) != len(retL) {
                            err = cmn.LogError("%s: len Mismatch ExpVal (%d) != retVal (%d)",
                                tid, len(expL), len(retL))
                        } else {
                            for i, e := range expL {
                                if e != retL[i] {
                                    err = cmn.LogError("%s: val Mismatch index(%d) (%s) != (%s)",
                                        tid, e, retL[i])
                                }
                            }
                        }
                    default:
                        if expVal != retV {
                            err = cmn.LogError("%s: ExpVal(%v) != RetV(%v)(%T)", tid, expVal, retV, retV)
                        }
                    }
                }
                cache.SetVal(e.name, retV)
            }
        }
        if err != nil {
            return
        }
    }
}
