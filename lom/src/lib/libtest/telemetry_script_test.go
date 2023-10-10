package libtest

import (
    "fmt"
    "reflect"
    "testing"

    cmn "lom/src/lib/lomcommon"
    . "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

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


func toInt(lst []string) (ret []int, err error) {
    ret = []int{}
    for _, s := range lst {
        i = 0
        if i, err = strconv.Atoi(s); err != nil {
            return
        }
        ret = append(ret, i)
    }
    return
}


func TestAnyFn(name string, val any) (any, error) {
    var err error
    var ret any

    if s, ok := val.(string); !ok {
        err = cmn.LogError("Unknown val type(%T) (%v) for TestAnyFn", val, val)
    } else if lst = strings.Split(s, ","); len(lst) < 2 {
        err = cmn.LogError("len(%d) < 2 (%v)", len(lst), lst)
    } else if lst[0] == "SET" {
        cnt := 0
        ret = func() []any {
            cnt++
            GetSuiteCache().SetVal(lst[1], cnt)
            return []any{}
        }
    } else if lst[0] == "GET" {
        cnt := 0
        ret = func() []any {
            cnt++
            ret := GetSuiteCache().GetVal(lst[1], cnt)
            return []any{ret}
        }
    } else if (lst[0] == "LOOP") {
        var indices []int
        if len(lst) < 4 {
            err = cmn.LogError("Loop: len(%d) < 4 (%v)", len(lst), lst)
        } else if indices, err = toInt(lst[1:]); err != nil {
        } else {
            ct := indices[0]
            ret = func() []any {
                if ct < indices[1] {
                    GetSuiteCache().SetVal(LOOP_CACHE_INDEX_NAME, indices[2])
                    ct++
                } else {
                    GetSuiteCache().SetVal(LOOP_CACHE_INDEX_NAME, nil)
                }
                return []any{}
            }
        }
    } else {
        err = cmn.LogError("Unknown val type(%T) (%v) for TestAnyFn", val, val)
    }
    return
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
    ScriptSuite_t {
        Id: "TestAny",
        Description: "",
        Entries: []ScriptEntry_t{
            ScriptEntry_t{
                ApiIDAny,
                []Param_t{Param_t{ANONYMOUS, "SET,foo", TestAnyFn} },
                []Result_t{ NIL_ERROR },
                "Call Any test",
            },
            ScriptEntry_t{
                ApiIDAny,
                []Param_t{Param_t{ANONYMOUS, "LOOP,0,1,0", TestAnyFn} },
                []Result_t{ NIL_ERROR },
                "Call Any loop",
            },
            ScriptEntry_t{
                ApiIDAny,
                []Param_t{Param_t{ANONYMOUS, "GET,foo", TestAnyFn} },
                []Result_t{Result_t{ANONYMOUS, 2, nil}, NIL_ERROR },
                "Check SET called twice",
            },
        },
    },

}


func TestRunOneScriptSuite(t *testing.T) {
    cmn.InitSysShutdown()   /* Ensure clean init of the object */
    defer cmn.InitSysShutdown()

    /*
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
    */
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


