package plugins_script

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "testing"
    "time"

    "github.com/agiledragon/gomonkey/v2"

    cmn "lom/src/lib/lomcommon"
    tele "lom/src/lib/lomtelemetry"
)

type testData struct {
    scrDir  string
    dirs    []string
    files   []string
    cfg     cmn.ActionCfg_t
    fail    bool
    spl     ScriptBasedPlugin
    desc    string
}

func initTestCase (t *testing.T, index int, tc *testData) {
    failed := false
    defer func() {
        if failed {
            t.Logf("TestInit: tc (%d) desc:(%s) ---- FAILED ---", index, tc.desc)
        } else {
            t.Logf("TestInit: tc (%d) desc:(%s) ---- SUCCEEDED ---", index, tc.desc)
        }
    }()

    spl, ok := NewScriptBasedPlugin().(*ScriptBasedPlugin)
    if !ok {
        failed = true
        t.Fatalf("Unable to get *NewScriptBasedPlugin (%T)", NewScriptBasedPlugin())
    } else if tc.scrDir != "" {
        failed = true
        if err := os.RemoveAll(tc.scrDir); err != nil {
            t.Errorf("Failed to remove dir (%s)", tc.scrDir)
        } else if err := os.Mkdir(tc.scrDir, 0777); err != nil {
            t.Errorf("Failed to mkdir dir (%s)", tc.scrDir)
        } else {
            failed = false
        }
    }
    if !failed {
        for _, dir := range tc.dirs {
            path := filepath.Join(tc.scrDir, dir)
            if err := os.Mkdir(path, 0777); err != nil {
                t.Errorf("Failed to mkdir dir (%s)", path)
                failed = true
                break
            }   
        }
    }
    if !failed {
        for _, file := range tc.files {
            path := filepath.Join(tc.scrDir, file)
            if fl, err := os.Create(path); err != nil {
                t.Errorf("Failed to create file (%s)", path)
                failed = true
                break
            } else {
                fl.Close()
            }
        }
    }
    if !failed {
        if err := spl.Init(&tc.cfg); tc.fail != (err != nil) {
            t.Errorf("Expect fail(%v) err(%v)", tc.fail, err)
            failed = true
        } 
    }
    if !failed && !tc.fail {
        failed = true
        if err := spl.Init(&tc.cfg); tc.fail != (err != nil) {
            t.Errorf("Expect fail(%v) err(%v)", tc.fail, err)
        } else if tc.fail {
            /* No more data checks onb expected failure */
        } else if spl.heartbeatInt != tc.cfg.HeartbeatInt {
            t.Errorf("Heartbeat mismatch exp(%d) ret(%v)", tc.cfg.HeartbeatInt, tc.spl.heartbeatInt)
        } else if err = cmn.CompareSlices(spl.files, tc.spl.files); err != nil {
            t.Errorf("Files mismatch: (%v)", err)
        } else if spl.errConsecutive != tc.spl.errConsecutive {
            t.Errorf("errConsecutive mismatch exp(%d) ret(%v)", tc.spl.errConsecutive, spl.errConsecutive)
        } else if spl.scrTimeout != tc.spl.scrTimeout {
            t.Errorf("scrTimeout mismatch exp(%d) ret(%v)", tc.spl.scrTimeout, spl.scrTimeout)
        } else if spl.pausePeriod != tc.spl.pausePeriod {
            t.Errorf("pausePeriod mismatch exp(%d) ret(%v)", tc.spl.pausePeriod, spl.pausePeriod)
        } else if spl.wg != tc.spl.wg {
            t.Errorf("wg mismatch exp(%d) ret(%v)", tc.spl.wg, spl.wg)
        } else if spl.stopSignal != tc.spl.stopSignal {
            t.Errorf("stopSignal mismatch exp(%d) ret(%v)", tc.spl.stopSignal, spl.stopSignal)
        } else {
            failed = false
        }
    }
}

func TestInit(t *testing.T) {
    lstCases := []testData {
        {
            fail:       true,
            desc:       "Invalid heartbeat. 0",
        },
        {
            cfg :       cmn.ActionCfg_t{ HeartbeatInt: 5, ActionKnobs: []byte("foo") },
            fail:       true,
            desc:       "Invalid JSON string for knobs",
        },
        {
            cfg :       cmn.ActionCfg_t{ HeartbeatInt: 5, ActionKnobs: []byte{} },
            fail:       true,
            desc:       "Empty dir",
        },
        {
            cfg :       cmn.ActionCfg_t{ HeartbeatInt: 5, ActionKnobs: []byte(`{"ScriptsPath": ""}`) },
            fail:       true,
            desc:       "Empty scripts path",
        },
        {
            cfg :       cmn.ActionCfg_t{ HeartbeatInt: 5, ActionKnobs: []byte(`{"ScriptsPath": ""}`) },
            fail:       true,
            desc:       "Empty scripts path",
        },
        {
            cfg :       cmn.ActionCfg_t{ HeartbeatInt: 5, ActionKnobs: []byte(`{"ScriptsPath": "/zzz"}`) },
            fail:       true,
            desc:       "Non existing scripts path",
        },
        {
            scrDir:     "/tmp/xxz",
            dirs:       []string {"foo", "foo/bar" },
            files:      []string {
                            "fooX.py",
                            "fooX_pl_script.py",
                            "foo/foo_pl_script_xx.py",
                            "foo/bar/_pl_script.",
                            "foo/x_pl_script_aa.py",
                            "foo/bar/x_pl.script_aa.py",
                        },
            cfg:        cmn.ActionCfg_t{ HeartbeatInt: 5, ActionKnobs: []byte(`{"ScriptsPath": "/tmp/xxz"}`) },
            fail:       false,
            spl:        ScriptBasedPlugin {
                scriptsPath:    "/tmp/xyz",
                heartbeatInt:   5,
                files:          [] string { "/tmp/xxz/fooX_pl_script.py", "/tmp/xxz/foo/bar/_pl_script." },
                errConsecutive: SCR_MAX_CONSECUTIVE_ERR_CNT,
                scrTimeout:     SCR_RUN_TIMEOUT,
                pausePeriod:    SCR_PAUSE_PERIOD,
            },
            desc:       "Good case with default knobs.",
        },
        {
            scrDir:     "/tmp/xxz",
            files:      []string {"fooZ_pl_script.py"},
            cfg:        cmn.ActionCfg_t { 
                    HeartbeatInt: 4,
                    ActionKnobs: []byte(`{
                            "ScriptsPath": "/tmp/xxz", 
                            "ConsecutiveErrCnt": 7,
                            "ScriptTimeout": 240,
                            "PausePeriod": 120 }`) },
            fail:       false,
            spl:        ScriptBasedPlugin {
                scriptsPath:    "/tmp/xxz",
                heartbeatInt:   4,
                files:          [] string {"/tmp/xxz/fooZ_pl_script.py"},
                errConsecutive: 7,
                scrTimeout:     240,
                pausePeriod:    120,
            },
            desc:       "Good case non default knobs.",
        },
        {
            scrDir:     "/tmp/xxz",
            files:      []string {"fooZ_pl_script.py"},
            cfg:        cmn.ActionCfg_t { 
                    HeartbeatInt: 4,
                    ActionKnobs: []byte(`{
                            "ScriptsPath": "/tmp/xxz", 
                            "ConsecutiveErrCnt": 700,
                            "ScriptTimeout": 0,
                            "PausePeriod": 0 }`) },
            fail:       false,
            spl:        ScriptBasedPlugin {
                scriptsPath:    "/tmp/xxz",
                heartbeatInt:   4,
                files:          [] string {"/tmp/xxz/fooZ_pl_script.py"},
                errConsecutive: 5,
                scrTimeout:     180,
                pausePeriod:    300,
            },
            desc:       "Good case with default path and invalid action knobs, hence default.",
        },
    }
    

    for index, tc := range lstCases {
        initTestCase(t, index, &tc)
    }
}

type runTestData struct {
    spl         ScriptBasedPlugin
    script      string
    events      []string
    hbData      []string
    testTimeout int
    desc        string
}

func runPluginCase(t *testing.T, tdir string, tcIndex int, data *runTestData) {
    failed := false
    defer func() {
        if failed {
            t.Logf("RunPluginCase: (%d): (%s) completed ----- FAILED ------", tcIndex, data.desc)
        } else {
            t.Logf("RunPluginCase: (%d): (%s) completed ****** SUCCEEDED ******", tcIndex, data.desc)
        }
    }()

    path := filepath.Join(tdir, fmt.Sprintf("tc_%d", tcIndex))
    if testF, err := os.Create(path); err != nil {
        t.Fatalf("Failed to create temp file (%v)", err)
    } else {
        testF.Write([]byte(data.script))
        testF.Chmod(0777)
        testF.Close()
    }

    evtCh := make(chan string, 1)
    mockPub := gomonkey.ApplyFunc(tele.PublishEvent, func(data any) (err error) {
        if s, ok := data.(string); ok {
            evtCh <- s
            return nil
        } else {
            return cmn.LogError("Published data type (%T) != string", data)
        }
    })
    defer mockPub.Reset()
        

    spl := data.spl
    spl.wg =  new(sync.WaitGroup)
    spl.stopSignal = make(chan struct{})
    defer close(spl.stopSignal)

    eventsCnt := len(data.events)
    hbCnt := len(data.hbData)

    hbChan := make (chan string, hbCnt)

    spl.wg.Add(1)
    /* Start code under test */
    go spl.runPlugin(path, hbChan)

tLoop:
    for (eventsCnt + hbCnt) != 0 {
        select {
        case evStr := <-evtCh:
            if eventsCnt > 0 {
                index := len(data.events) - eventsCnt
                if !strings.HasPrefix(string(evStr), data.events[index]) {
                    t.Errorf("Unexepected evt. exp(%s) rcvd(%s)", 
                            data.events[index], evStr)
                    failed = true
                }
                eventsCnt--
            }
        case hbStr := <-hbChan:
            if hbCnt > 0 {
                index := len(data.hbData) - hbCnt
                if !strings.HasPrefix(hbStr, data.hbData[index]) {
                    t.Errorf("Unexepected hb. exp(%s) rcvd(%s)",
                            data.hbData[index], hbStr)
                    failed = true
                }
                hbCnt--
            }
        case <-time.After(time.Duration(data.testTimeout) * time.Second):
            failed = true
            t.Fatalf("Test aborted due to timeout eventsCnt(%d) hbCnt(%d)", eventsCnt, hbCnt)
            break tLoop
        }
    }
    if !failed && ((eventsCnt + hbCnt) != 0) {
        failed = true
    }
    if failed {
        t.Fatalf("Test failed: (%s)", data.desc)
    }
}

const sleepForever = `#! /bin/bash
echo "sleeping ...."
sleep 1h
`

func TestRunPlugin(t *testing.T) {
    lstCases := []runTestData {
        {
            spl:    ScriptBasedPlugin {errConsecutive: 3, scrTimeout: 3, pausePeriod: 2 },
            script: sleepForever,
            events: []string {"","", "", ""},
            testTimeout: 30,
            desc:   "Test timeout",
        },
    }

    tdir, err := os.MkdirTemp("", "testingRunPlugin")
    if err != nil {
        t.Fatalf("Failed to create tempdir (%v)", err)
    }

    for i, tdata := range lstCases {
        runPluginCase(t, tdir, i, &tdata)
    }
    //os.RemoveAll(tdir)
}


func TestRequest(t *testing.T) {
   
}
