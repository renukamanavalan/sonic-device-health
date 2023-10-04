package libtest

import (
    "fmt"
    "testing"
    "time"

    cmn "lom/src/lib/lomcommon"
    . "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

func testRunOneTeleSuite(t *testing.T, suite *ScriptSuite_t) {
    /* Caches all variables for reference across test entries */
    cache := ResetSuiteCache()
    defer ResetSuiteCache()

    t.Logf(logFmt("Starting test suite - {%s} ....", suite.Id))

    defer func() { t.Logf(logFmt("Ended test suite - {%s} ....", suite.Id)) }()

    for i, entry := range suite.Entries {
        tid := fmt.Sprintf("%s:%d:%s", suite.Id, i, entry.Api)
        t.Logf(logFmt("%s: Starting test[%d] - {%v} {%s}....", tid, i, entry.Api, entry.Message))

        retVals, ok := CallByApiID(entry.Api, entry.Args, cache)

        if !ok {
            t.Fatalf(fatalFmt("%s: Failed to find API (%v)", tid, entry.Api))
        }
        if len(retVals) != len(entry.Result) {
            t.Fatalf(fatalFmt("%s: Return length (%d) != expected (%d)", tid, len(retVals), len(entry.Result)))
        }
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
                    t.Fatalf(fatalFmt("%s:Result validation failed testID(%d) res-index(%d) retv(%+v)",
                        tid, i, j, retV))
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
            cache.SetVal(e.Name, retV)
        }
        t.Logf(logFmt("%s: Ended test(%d) - {%v} ....", tid, i, entry.Api))
    }
}

func TestRunTeleSuites(t *testing.T) {
    ctTimeout := tele.SUB_CHANNEL_TIMEOUT
    tele.SUB_CHANNEL_TIMEOUT = time.Duration(1) * time.Second
    cmn.InitSysShutdown()   /* Ensure clean init of the object */

    defer func() {
        tele.SUB_CHANNEL_TIMEOUT = ctTimeout
        cmn.InitSysShutdown()   /* Ensure clean init of the object */
    }()

    for _, suite := range testTelemetrySuites {
        testRunOneTeleSuite(t, suite)
        if !tele.IsTelemetryIdle() {
            t.Fatalf(fatalFmt("Telemetry not idle after suite=%s", suite.Id))
            break
        }
    }
}
