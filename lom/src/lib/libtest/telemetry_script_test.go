package libtest

import (
    "reflect"
    "strconv"
    "strings"
    "testing"

    cmn "lom/src/lib/lomcommon"
    . "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

var ch0 = make(chan tele.JsonString_t, 3)
var ch0W chan<- tele.JsonString_t = ch0
var ch0R <-chan tele.JsonString_t = ch0

var FailRunOneScriptSuites = []ScriptSuite_t{
    ScriptSuite_t{
        Id:          "Non-Existing-API",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{"xyz", []Param_t{}, []Result_t{}, "Fail to find API"},
        },
    },
    ScriptSuite_t{
        Id:          "IdleChk-res-len-incorrect",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{ApiIDIsTelemetryIdle, []Param_t{}, []Result_t{}, "2 res expected"},
        },
    },
    ScriptSuite_t{
        Id:          "IdleChk-res-mismatch-bool",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDIsTelemetryIdle,
                []Param_t{},
                []Result_t{TEST_FOR_FALSE, NIL_ERROR},
                "Result validation fails",
            },
        },
    },
    ScriptSuite_t{
        Id:          "IdleChk-res-mismatch-err",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDIsTelemetryIdle,
                []Param_t{},
                []Result_t{TEST_FOR_TRUE, NON_NIL_ERROR},
                "Result validation fails",
            },
        },
    },
    ScriptSuite_t{
        Id:          "Expect Json string",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDIsTelemetryIdle,
                []Param_t{},
                []Result_t{Result_t{ANONYMOUS, []tele.JsonString_t{}, nil}, NON_NIL_ERROR},
                "Result validation fails",
            },
        },
    },
    ScriptSuite_t{
        Id:          "Write strings 0",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDWriteJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0W, nil},
                    Param_t{ANONYMOUS, []tele.JsonString_t{tele.JsonString_t("foo")}, nil},
                    Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, nil, ValidateNonNil},
                }, /*Expect nil error */
                "Create data in chan",
            },
        },
    },
    ScriptSuite_t{
        Id:          "Read strings 0",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDReadJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0R, nil},
                    Param_t{ANONYMOUS, 1, nil}, /* read 1 */
                    Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, []tele.JsonString_t{}, nil},
                    Result_t{ANONYMOUS, nil, ValidateNil},
                }, /*Expect nil error */
                "read len mismatch",
            },
        },
    },
    ScriptSuite_t{
        Id:          "Write strings 0",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDWriteJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0W, nil},
                    Param_t{ANONYMOUS, []tele.JsonString_t{tele.JsonString_t("foo")}, nil},
                    Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, nil, ValidateNonNil},
                }, /*Expect nil error */
                "Create data in chan",
            },
        },
    },
    ScriptSuite_t{
        Id:          "Read strings 0",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDReadJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0R, nil},
                    Param_t{ANONYMOUS, 1, nil}, /* read 1 */
                    Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, []tele.JsonString_t{tele.JsonString_t("bar")}, nil},
                    Result_t{ANONYMOUS, nil, ValidateNil},
                }, /*Expect nil error */
                "read len mismatch",
            },
        },
    },
    ScriptSuite_t{
        Id:          "Write strings 0",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDWriteJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0W, nil},
                    Param_t{ANONYMOUS, []tele.JsonString_t{tele.JsonString_t("foo")}, nil},
                    Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, nil, ValidateNonNilError},
                }, /*Expect nil error */
                "Create data in chan",
            },
        },
    },
    ScriptSuite_t{
        Id:          "Read strings 0",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDReadJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0R, nil},
                    Param_t{ANONYMOUS, 1, nil}, /* read 1 */
                    Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
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

func toInt(lst []string) (ret []int, err error) {
    ret = []int{}
    for _, s := range lst {
        i := 0
        if i, err = strconv.Atoi(s); err != nil {
            return
        }
        ret = append(ret, i)
    }
    return
}

func getInt(s string, defVal int) int {
    if ctVal := GetSuiteCache().GetVal(s, nil, nil); ctVal != nil {
        if i, ok := ctVal.(int); ok {
            return i
        }
    }
    return defVal
}

func testAnyFn(name string, val any) (ret any, err error) {
    if name == ANONYMOUS {
        err = cmn.LogError("Need non-Anonymous name")
    } else if s, ok := val.(string); !ok {
        err = cmn.LogError("Unknown val type(%T) (%v) for testAnyFn", val, val)
    } else if lst := strings.Split(s, ","); len(lst) < 1 {
        err = cmn.LogError("len(%d) < 1 (%v)", len(lst), lst)
    } else if lst[0] == "SET" {
        ret = func() []any {
            ctCnt := getInt(name, 0) + 1
            GetSuiteCache().SetVal(name, ctCnt)
            // cmn.LogDebug("Called SET for (%s) setCnt(%d)", name, ctCnt)
            return []any{}
        }
    } else if lst[0] == "GET" {
        ret = func() []any {
            getRet := GetSuiteCache().GetVal(name, nil, nil)
            // cmn.LogDebug("Called GET for (%s) get(%v)(%T)", name, getRet, getRet)
            return []any{getRet}
        }
    } else if lst[0] == "LOOP" {
        var indices []int
        if len(lst) < 4 {
            err = cmn.LogError("Loop: len(%d) < 4 (%v)", len(lst), lst)
        } else if name == ANONYMOUS {
            err = cmn.LogError("Loop: Need name to save ct val")
        } else if indices, err = toInt(lst[1:]); err != nil {
        } else {
            ctIndex := getInt(name, indices[0])
            ret = func() []any {
                if ctIndex < indices[1] {
                    GetSuiteCache().SetVal(LOOP_CACHE_INDEX_NAME, indices[2])
                    GetSuiteCache().SetVal(name, ctIndex+1)

                }
                // cmn.LogDebug("Called LOOP LoopCt(%d) lst(%v)", ctIndex, indices)
                return []any{}
            }
        }
    } else {
        err = cmn.LogError("Unknown val type(%T) (%v) for testAnyFn", val, val)
    }
    return
}

var xGoodRunOneScriptSuites = []ScriptSuite_t{
    ScriptSuite_t{
        Id:          "IdleChk-good",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDIsTelemetryIdle,
                []Param_t{},
                []Result_t{TEST_FOR_TRUE, NIL_ERROR},
                "Result validation fails",
            },
        },
    },
    ScriptSuite_t{
        Id:          "Write strings fail",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDWriteJsonStringsChannel,
                []Param_t{
                    Param_t{ANONYMOUS, []tele.JsonString_t{tele.JsonString_t("foo")}, nil},
                    Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
                },
                []Result_t{
                    Result_t{ANONYMOUS, nil, ValidateNonNilError},
                }, /*Expect nil error */
                "Fail genuine",
            },
        },
    },
    ScriptSuite_t{
        Id:          "close channel",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDCloseChannel,
                []Param_t{
                    Param_t{ANONYMOUS, ch0W, nil},
                },
                []Result_t{NIL_ERROR},
                "close it",
            },
        },
    },
}

var GoodRunOneScriptSuites = []ScriptSuite_t{
    ScriptSuite_t{
        Id:          "TestAny",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDAny,
                []Param_t{Param_t{"Foo", "SET,", testAnyFn}},
                []Result_t{NIL_ERROR},
                "Call Any test",
            },
            ScriptEntry_t{
                ApiIDAny,
                []Param_t{Param_t{"LoopI", "LOOP,0,1,0", testAnyFn}},
                []Result_t{NIL_ERROR},
                "Call Any loop",
            },
            ScriptEntry_t{
                ApiIDAny,
                []Param_t{Param_t{"Foo", "GET,", testAnyFn}},
                []Result_t{Result_t{ANONYMOUS, 2, nil}, NIL_ERROR},
                "Check SET called twice",
            },
        },
    },
}

func TestRunOneScriptSuite(t *testing.T) {
    cmn.InitSysShutdown() /* Ensure clean init of the object */
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
