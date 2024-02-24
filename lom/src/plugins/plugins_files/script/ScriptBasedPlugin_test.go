package plugins_script

import (
    cmn "lom/src/lib/lomcommon"
    "os"
    "path/filepath"
    "testing"
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

func testInitCase (t *testing.T, index int, tc *testData) {
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
        testInitCase(t, index, &tc)
    }
}

