package libtest

import (
    "fmt"
    "testing"
    "time"

    tele "lom/src/lib/lomtelemetry"
    script "lom/src/lib/lomscripted"
)

func xTest_PubSub(t *testing.T) {
    ch, err := tele.GetPubChannel(tele.CHANNEL_TYPE_EVENTS, tele.CHANNEL_PRODUCER_ENGINE, "")
    /* ch close indirectly closes corresponding PUB channel too */
    defer close(ch)

    if err != nil {
        t.Fatalf(fatalFmt("Failed to get sub channel (%v)", err))
    }
    t.Logf(logFmt("Test Complete"))
}

func testRunOneTeleSuite(t *testing.T, suite *testSuite_t) {
    /* Caches all variables for reference across test entries */
    cache := script.SuiteCache_t{}

    t.Logf(logFmt("Starting test suite - {%s} ....", suite.id))

    defer func() { t.Logf(logFmt("Ended test suite - {%s} ....", suite.id)) }()

    for i, entry := range suite.tests {
        t.Logf(logFmt("Starting test[%d] - {%v} ....", i, entry.api))
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
                    t.Errorf(errorFmt("Result validation failed (%+v) retv(%+v)", entry, retV))
                    retV = nil
                }
            } else if expVal != retV {
                t.Fatalf(fatalFmt("%s: ExpVal(%v) != RetV(%v)(%T)", tid, expVal, retV, retV))
            }
            cache.SetVal(e.name, retV)
        }
        t.Logf(logFmt("Ended test - {%v} ....", entry.api))
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
