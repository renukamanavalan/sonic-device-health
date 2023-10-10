package libtest

import (
    "fmt"
    "reflect"
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
        tid := fmt.Sprintf("%s:%d:%s: ", suite.Id, i, entry.Api)
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

var ch0 = make(chan tele.JsonString_t, 3)
var ch0W chan<- tele.JsonString_t = ch0
var ch0R <-chan tele.JsonString_t = ch0

var FailRunOneScriptSuites = []ScriptSuite_t {
    ScriptSuite_t { 
        Id:     "Non-Existing-API",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t { "xyz", []Param_t{}, []Result_t{}, "Fail to find API" },
        },
    },
    ScriptSuite_t {
        Id: "IdleChk-res-len-incorrect",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t { ApiIDIsTelemetryIdle, []Param_t{}, []Result_t{}, "2 res expected" },
        },
    },
    ScriptSuite_t {
        Id: "IdleChk-res-mismatch-bool",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t {
                ApiIDIsTelemetryIdle,
                []Param_t{},
                []Result_t{TEST_FOR_FALSE, NIL_ERROR},
                "Result validation fails",
            },
        },
    },
    ScriptSuite_t {
        Id: "IdleChk-res-mismatch-err",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t {
                ApiIDIsTelemetryIdle,
                []Param_t{},
                []Result_t{TEST_FOR_TRUE, NON_NIL_ERROR},
                "Result validation fails",
            },
        },
    },
    ScriptSuite_t {
        Id: "Expect Json string",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t {
                ApiIDIsTelemetryIdle,
                []Param_t{},
                []Result_t{Result_t{ANONYMOUS, []tele.JsonString_t{}, nil}, NON_NIL_ERROR},
                "Result validation fails",
            },
        },
    },
    ScriptSuite_t {
        Id: "Write strings 0",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDWriteJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0W, nil},
                    Param_t{ANONYMOUS, []tele.JsonString_t{ tele.JsonString_t("foo") }, nil},
                    Param_t{ANONYMOUS, 1, nil},            /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, nil, ValidateNonNil},
                }, /*Expect nil error */
                "Create data in chan",
            },
        },
    },
    ScriptSuite_t {
        Id: "Read strings 0",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDReadJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0R, nil},
                    Param_t{ANONYMOUS, 1, nil},             /* read 1 */
                    Param_t{ANONYMOUS, 1, nil},             /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, []tele.JsonString_t{}, nil},
                    Result_t{ANONYMOUS, nil, ValidateNil},
                }, /*Expect nil error */
                "read len mismatch",
            },
        },
    },
    ScriptSuite_t {
        Id: "Write strings 0",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDWriteJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0W, nil},
                    Param_t{ANONYMOUS, []tele.JsonString_t{ tele.JsonString_t("foo") }, nil},
                    Param_t{ANONYMOUS, 1, nil},            /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, nil, ValidateNonNil},
                }, /*Expect nil error */
                "Create data in chan",
            },
        },
    },
    ScriptSuite_t {
        Id: "Read strings 0",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDReadJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0R, nil},
                    Param_t{ANONYMOUS, 1, nil},             /* read 1 */
                    Param_t{ANONYMOUS, 1, nil},             /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, []tele.JsonString_t{tele.JsonString_t("bar")}, nil},
                    Result_t{ANONYMOUS, nil, ValidateNil},
                }, /*Expect nil error */
                "read len mismatch",
            },
        },
    },
    ScriptSuite_t {
        Id: "Write strings 0",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDWriteJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0W, nil},
                    Param_t{ANONYMOUS, []tele.JsonString_t{ tele.JsonString_t("foo") }, nil},
                    Param_t{ANONYMOUS, 1, nil},            /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, nil, ValidateNonNilError},
                }, /*Expect nil error */
                "Create data in chan",
            },
        },
    },
    ScriptSuite_t {
        Id: "Read strings 0",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDReadJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0R, nil},
                    Param_t{ANONYMOUS, 1, nil},             /* read 1 */
                    Param_t{ANONYMOUS, 1, nil},             /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, nil, ValidateNonNilError},
                    Result_t{ANONYMOUS, nil, ValidateNil},
                }, /*Expect nil error */
                "read len mismatch",
            },
        },
    },
}


var GoodRunOneScriptSuites = []ScriptSuite_t {
    ScriptSuite_t {
        Id: "IdleChk-good",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t {
                ApiIDIsTelemetryIdle,
                []Param_t{},
                []Result_t{TEST_FOR_TRUE, NIL_ERROR},
                "Result validation fails",
            },
        },
    },
    ScriptSuite_t {
        Id: "Write strings fail",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDWriteJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, []tele.JsonString_t{ tele.JsonString_t("foo") }, nil},
                    Param_t{ANONYMOUS, 1, nil},            /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, nil, ValidateNonNilError},
                }, /*Expect nil error */
                "Fail genuine",
            },
        },
    },
    ScriptSuite_t {
        Id: "close channel",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDCloseChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0W, nil},
                },
                []Result_t{ NIL_ERROR },
                "close it",
            },
        },
    },
}


func TestRunOneScriptSuite(t *testing.T) {
    cmn.InitSysShutdown()   /* Ensure clean init of the object */
    defer cmn.InitSysShutdown()

    for _, suite := range FailRunOneScriptSuites {
        t.Logf(logFmt("Starting test suite - {%s} ....", suite.Id))
        chSt := GetSuiteCache()
        if err := RunOneScriptSuite(&suite); err == nil {
            t.Fatalf(fatalFmt("Failed to fail for suite (%v)", suite.Id))
        }
        chCt := GetSuiteCache()
        if reflect.ValueOf(chSt).Pointer() == reflect.ValueOf(chCt).Pointer() {
            t.Fatalf(fatalFmt("Failed to refresh cache"))
        }
        t.Logf(logFmt("Ended test suite - {%s} ....", suite.Id))
    }
    for _, suite := range GoodRunOneScriptSuites {
        t.Logf(logFmt("Starting test suite - {%s} ....", suite.Id))
        chSt := GetSuiteCache()
        if err := RunOneScriptSuite(&suite); err != nil {
            t.Fatalf(fatalFmt("Failed to fail for suite (%v)", suite.Id))
        }
        chCt := GetSuiteCache()
        if reflect.ValueOf(chSt).Pointer() == reflect.ValueOf(chCt).Pointer() {
            t.Fatalf(fatalFmt("Failed to refresh cache in Good loop"))
        }
        t.Logf(logFmt("Ended test suite - {%s} ....", suite.Id))
    }
}


