package libtest

import (
    "fmt"
    "testing"
    "time"

    script "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

func testRunOneTeleSuite(t *testing.T, suite *testSuite_t) {
    /* Caches all variables for reference across test entries */
    cache := script.SuiteCache_t{}
    setTestCache(cache)
    defer resetTestCache()

    t.Logf(logFmt("Starting test suite - {%s} ....", suite.id))

    defer func() { t.Logf(logFmt("Ended test suite - {%s} ....", suite.id)) }()

    for i, entry := range suite.tests {
        t.Logf(logFmt("Starting test[%d] - {%v} {%s}....", i, entry.api, entry.message))
        tid := fmt.Sprintf("%s:%d:%s", suite.id, i, entry.api)

        retVals, ok := script.CallByApiID(entry.api, entry.args, cache)

        if !ok {
            t.Fatalf(fatalFmt("%s: Failed to find API (%v)", tid, entry.api))
        }
        if len(retVals) != len(entry.result) {
            t.Fatalf(fatalFmt("%s: Return length (%d) != expected (%d)", tid, len(retVals), len(entry.result)))
        }
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
                    t.Fatalf(errorFmt("Result validation failed testID(%d) res-index(%d) retv(%+v)",
                                i, j, retV))
                    retV = nil
                }
            } else {
                switch expVal.(type) {
                case []tele.JsonString_t:
                    expL := expVal.([]tele.JsonString_t)
                    if retL, ok := retV.([]tele.JsonString_t); !ok {
                        t.Fatalf(fatalFmt("%s: ExpVal(%T) != RetV(%T)", tid, expVal, retV))
                    } else if len(expL) != len(retL) {
                        t.Fatalf(fatalFmt("%s: len Mismatch ExpVal (%d) != retVal (%d)",
                            tid, len(expL), len(retL)))
                    } else {
                        for i, e := range expL {
                            if e != retL[i] {
                                t.Fatalf(fatalFmt("%s: val Mismatch index(%d) (%s) != (%s)",
                                    tid, e, retL[i]))
                            }
                        }
                    }
                default:
                    if expVal != retV {
                        t.Fatalf(fatalFmt("%s: ExpVal(%v) != RetV(%v)(%T)", tid, expVal, retV, retV))
                    }
                }
            }
            cache.SetVal(e.name, retV)
        }
        t.Logf(logFmt("Ended test(%d) - {%v} ....", i, entry.api))
    }
}

func setUTGlobals() {
    tele.SUB_CHANNEL_TIMEOUT = time.Duration(1) * time.Second
}

func TestRunTeleSuites(t *testing.T) {
    setUTGlobals()

    for _, suite := range testTelemetrySuites {
        testRunOneTeleSuite(t, suite)
    }
}
