package plugins_script

import (
    "encoding/json"
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
    pcmn "lom/src/plugins/plugins_common"
)

type testData struct {
    scrDir string
    dirs   []string
    files  []string
    cfg    cmn.ActionCfg_t
    fail   bool
    spl    ScriptBasedPlugin
    desc   string
}

func initTestCase(t *testing.T, index int, tc *testData) {
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
    lstCases := []testData{
        {
            fail: true,
            desc: "Invalid heartbeat. 0",
        },
        {
            cfg:  cmn.ActionCfg_t{HeartbeatInt: 5, ActionKnobs: []byte("foo")},
            fail: true,
            desc: "Invalid JSON string for knobs",
        },
        {
            cfg:  cmn.ActionCfg_t{HeartbeatInt: 5, ActionKnobs: []byte{}},
            fail: true,
            desc: "Empty dir",
        },
        {
            cfg:  cmn.ActionCfg_t{HeartbeatInt: 5, ActionKnobs: []byte(`{"ScriptsPath": ""}`)},
            fail: true,
            desc: "Empty scripts path",
        },
        {
            cfg:  cmn.ActionCfg_t{HeartbeatInt: 5, ActionKnobs: []byte(`{"ScriptsPath": "/zzz"}`)},
            fail: true,
            desc: "Non existing scripts path",
        },
        {
            cfg:  cmn.ActionCfg_t{HeartbeatInt: 5, ActionKnobs: []byte(`{"ScriptsPath": "/etc/ssl/private"}`)},
            fail: true,
            desc: "Fail to read for permission",
        },
        {
            scrDir: "/tmp/xxz",
            dirs:   []string{"foo", "foo/bar"},
            files: []string{
                "fooX.py",
                "fooX_pl_script.py",
                "foo/foo_pl_script_xx.py",
                "foo/bar/_pl_script.",
                "foo/x_pl_script_aa.py",
                "foo/bar/x_pl.script_aa.py",
            },
            cfg:  cmn.ActionCfg_t{HeartbeatInt: 5, ActionKnobs: []byte(`{"ScriptsPath": "/tmp/xxz"}`)},
            fail: false,
            spl: ScriptBasedPlugin{
                scriptsPath:    "/tmp/xyz",
                heartbeatInt:   5,
                files:          []string{"/tmp/xxz/fooX_pl_script.py", "/tmp/xxz/foo/bar/_pl_script."},
                errConsecutive: SCR_MAX_CONSECUTIVE_ERR_CNT,
                scrTimeout:     SCR_RUN_TIMEOUT,
                pausePeriod:    SCR_PAUSE_PERIOD,
            },
            desc: "Good case with default knobs.",
        },
        {
            scrDir: "/tmp/xxz",
            files:  []string{"fooZ_pl_script.py"},
            cfg: cmn.ActionCfg_t{
                HeartbeatInt: 4,
                ActionKnobs: []byte(`{
                            "ScriptsPath": "/tmp/xxz", 
                            "ConsecutiveErrCnt": 7,
                            "ScriptTimeout": 240,
                            "PausePeriod": 120 }`)},
            fail: false,
            spl: ScriptBasedPlugin{
                scriptsPath:    "/tmp/xxz",
                heartbeatInt:   4,
                files:          []string{"/tmp/xxz/fooZ_pl_script.py"},
                errConsecutive: 7,
                scrTimeout:     240,
                pausePeriod:    120,
            },
            desc: "Good case non default knobs.",
        },
        {
            scrDir: "/tmp/xxz",
            files:  []string{"fooZ_pl_script.py"},
            cfg: cmn.ActionCfg_t{
                HeartbeatInt: 4,
                ActionKnobs: []byte(`{
                            "ScriptsPath": "/tmp/xxz", 
                            "ConsecutiveErrCnt": 700,
                            "ScriptTimeout": 0,
                            "PausePeriod": 0 }`)},
            fail: false,
            spl: ScriptBasedPlugin{
                scriptsPath:    "/tmp/xxz",
                heartbeatInt:   4,
                files:          []string{"/tmp/xxz/fooZ_pl_script.py"},
                errConsecutive: 5,
                scrTimeout:     180,
                pausePeriod:    300,
            },
            desc: "Good case with default path and invalid action knobs, hence default.",
        },
    }

    for index, tc := range lstCases {
        initTestCase(t, index, &tc)
    }
}

type runTestData struct {
    spl         ScriptBasedPlugin
    script      string
    rcEvents    []string
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
        t.Fatalf("Failed to create temp file (%s) (%v)", path, err)
    } else {
        testF.Write([]byte(data.script))
        testF.Chmod(0777)
        testF.Close()
    }

    /* We got to mock tele.PublishEvent as pub/sub will not work when run from
     * within same process.
     */
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
    spl.wg = new(sync.WaitGroup)
    spl.stopSignal = make(chan struct{})
    defer close(spl.stopSignal)

    rcEventsCnt := len(data.rcEvents)
    hbCnt := len(data.hbData)

    hbChan := make(chan string, hbCnt)

    spl.wg.Add(1)
    /* Start code under test */
    go spl.runPlugin(path, hbChan)

tLoop:
    for (rcEventsCnt + hbCnt) != 0 {
        select {
        case evStr := <-evtCh:
            if rcEventsCnt > 0 {
                index := len(data.rcEvents) - rcEventsCnt
                eRet := errRet_t{}
                if err := json.Unmarshal([]byte(evStr), &eRet); err != nil {
                    t.Errorf("Failed to unmarshal (%s) as (%T) err(%v)", evStr, eRet, err)
                    failed = true
                } else if !strings.HasPrefix(eRet.RootCause, data.rcEvents[index]) {
                    t.Errorf("Unexepected evt. exp(%s) rcvd(%s)",
                        data.rcEvents[index], eRet.RootCause)
                    failed = true
                }
                rcEventsCnt--
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
            t.Fatalf("Test aborted due to timeout rcEventsCnt(%d) hbCnt(%d)", rcEventsCnt, hbCnt)
            break tLoop
        }
    }
    if !failed && ((rcEventsCnt + hbCnt) != 0) {
        failed = true
    }
    if failed {
        t.Fatalf("Test failed: (%s)", data.desc)
    }
}

const sleepForeverScript = `#! /bin/bash
echo "sleeping ...."
sleep 1h
`
const invalidJsonScript = `#! /bin/bash
echo "UT to fail unmarshal"
`

const missActionScript = `#! /bin/bash
echo -n "{}"
`

const goodScript = `#! /bin/bash
echo -n "{\"action\": \"UTTest\", \"res\": 0}"
`

func TestRunPlugin(t *testing.T) {
    lstCases := []runTestData{
        {
            spl:         ScriptBasedPlugin{errConsecutive: 3, scrTimeout: 3, pausePeriod: 1},
            script:      sleepForeverScript,
            rcEvents:    []string{"Run failed:", "Run failed:", "Run failed:", "Too many fail"},
            testTimeout: 30,
            desc:        "Test timeout",
        },
        {
            spl:         ScriptBasedPlugin{errConsecutive: 3, scrTimeout: 3, pausePeriod: 1},
            script:      invalidJsonScript,
            rcEvents:    []string{"Validate failed:", "Validate failed:", "Validate failed:", "Too many failures"},
            testTimeout: 30,
            desc:        "Invalid JSON string",
        },
        {
            spl:         ScriptBasedPlugin{errConsecutive: 3, scrTimeout: 3, pausePeriod: 1},
            script:      missActionScript,
            rcEvents:    []string{"", "", ""},
            hbData:      []string{"tc_", "tc_", "tc_"},
            testTimeout: 30,
            desc:        "Valid but missing action",
        },
        {
            spl:         ScriptBasedPlugin{errConsecutive: 3, scrTimeout: 3, pausePeriod: 1},
            script:      goodScript,
            rcEvents:    []string{"", "", ""},
            hbData:      []string{"UTTest", "UTTest", "UTTest"},
            testTimeout: 30,
            desc:        "Valid with action",
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
    flName := "/tmp/xxx"

    spl := ScriptBasedPlugin{
        errConsecutive: 3,
        scrTimeout:     3,
        pausePeriod:    1,
        files:          []string{flName},
        heartbeatInt:   1,
    }

    if testF, err := os.Create(flName); err != nil {
        t.Fatalf("Failed to create file(%s) (%v)", flName, err)
    } else {
        testF.Write([]byte(goodScript))
        testF.Chmod(0777)
        testF.Close()
    }

    hbChan := make(chan pcmn.PluginHeartBeat)
    hbCnt := 0

    chEnd := make(chan int)

    wg := sync.WaitGroup{}

    wg.Add(1)
    /* Async drain of heartbeat with validation and count */
    go func() {
    loop1:
        for {
            select {
            case hb, more := <-hbChan:
                if !more {
                    /* channel closed */
                    cmn.LogInfo("Heartbeat channel closed")
                    break loop1

                } else if hb.PluginName != plugin_name {
                    cmn.LogError("Heartbeat: Plugin name incorrect (%s) != ScriptBasedPlugin", hb.PluginName)
                } else if hb.EpochTime <= 0 {
                    cmn.LogError("Heartbeat: hb.EpochTime (%d)", hb.EpochTime)
                } else {
                    hbCnt++
                }
            case <-chEnd:
                /* Request returned */
                cmn.LogInfo("Request complete")
                break loop1
            }
        }
        wg.Done()
    }()

    /* call shutdown after N seconds */
    wg.Add(1)
    go func() {
        time.Sleep(time.Duration(spl.heartbeatInt*5) * time.Second)
        spl.Shutdown()
        wg.Done()
    }()

    /* Run Request method under test.
     * This expected to return upon shutdown called by above async routine.
     * The successful return is indicated by close of chEnd
     */
    go func() {
        spl.Request(hbChan, nil)
        close(hbChan)
        close(chEnd)
    }()

    timeout := 30

    /* End the test after Request returns or timeout */
    select {
    case <-chEnd:
        /* Wait for request end */

    case <-time.After(time.Duration(timeout) * time.Second):
        t.Fatalf("TestRequest is aborted after (%d) seconds", timeout)
    }

    if hbCnt == 0 {
        t.Fatalf("No heartbeat observed")
    }
    wg.Wait() /* Ensure heartbeat drain & shutdown routines ended */
}

func TestMisc(t *testing.T) {
    spl := ScriptBasedPlugin{}

    id := spl.GetPluginID()

    if id.Name != plugin_name {
        t.Errorf("spl.GetPluginID name(%s) != exp(%s)", id.Name, plugin_name)
    }
    if id.Version == "" {
        t.Errorf("spl.GetPluginID Expect non empty version")
    }
}
