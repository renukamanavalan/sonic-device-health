/*
 * A skeleton to plugin to run all binaries from specific folder periodically forever.
 *
 * It looks for all files in configured path (default: "/usr/share/lom/scripts") for files
 * that match pattern "_pl_script\\.".
 *
 * Each script is invoked periodically with a pause time (default: SCR_PAUSE_PERIOD - 5m)
 * between two invocations. Each invocation as a max timeout set (default: SCR_RUN_TIMEOUT)
 *
 * The script may do anything and by the end write a valid JSON string into its stdout.
 * This is sent to EVENT-HUB & Kusto-Syslog table.
 *
 * This plugin's Request method never returns. Kicks off a go routine per script and this
 * routine handles periodic invocation of the script.
 */

package plugins_script

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "path/filepath"
    "reflect"
    "strings"
    "syscall"
    "sync"
    "time"

    cmn "lom/src/lib/lomcommon"
    ipc "lom/src/lib/lomipc"
    tele "lom/src/lib/lomtelemetry"
    pcmn "lom/src/plugins/plugins_common"
)

type ScriptBasedPlugin struct {
    /* ... Internal plugin data */
    scriptsPath     string
    heartbeatInt    int
    files           []string
    errConsecutive  int
    scrTimeout      int
    pausePeriod     int
    wg              *sync.WaitGroup
    stopSignal      chan struct{} /* Channel to signal goroutine to stop */
}

const SCR_MAX_CONSECUTIVE_ERR_CNT_MAX = 10
const SCR_MAX_CONSECUTIVE_ERR_CNT_MIN = 3
const SCR_MAX_CONSECUTIVE_ERR_CNT = 5

/* In seconds */
const SCR_RUN_TIMEOUT_MIN = (1 * 60)        /* one min */
const SCR_RUN_TIMEOUT_MAX = (5 * 60)
const SCR_RUN_TIMEOUT = (3 * 60)

/* In seconds */
const SCR_PAUSE_PERIOD_MIN = (1 * 60)
const SCR_PAUSE_PERIOD_MAX = (5 * 60)
const SCR_PAUSE_PERIOD = (5 * 60)
const SCR_DEFAULT_PATH = "/usr/share/lom/scripts"

var scriptPatterns = []string{
    "_pl_script\\.",
}

const plugin_name = "ScriptBasedPlugin"

func NewScriptBasedPlugin(...interface{}) pcmn.Plugin {
    /* ... create and return a new instance of this Plugin */
    return &ScriptBasedPlugin{}
}

func init() {
    /* ... register the plugin with plugin manager */
    pcmn.RegisterPlugin(plugin_name, NewScriptBasedPlugin)
}

func runAScript(path string, timeout int) ([]byte, error) {
    cmn.LogInfo("Starting run for (%s) timeout(%d)", path, timeout)
    cmd := exec.Command(path)
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
    type cmdResult struct {
        outb []byte
        err  error
    }
    cmdDone := make(chan cmdResult, 1)
    go func() {
        outb, err := cmd.CombinedOutput()
        cmdDone <- cmdResult{outb, err}
    }()

    select {
    case <-time.After(time.Duration(timeout) * time.Second):
        /* sending a SIGKILL to the process ID negated, from the kill man page
         * https://man7.org/linux/man-pages/man2/kill.2.html
         */
        syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
        return []byte{}, cmn.LogError("Script (%s) killed after (%d) seconds", path, timeout)
    case ret := <-cmdDone:
        cmn.LogInfo("finished script( (%s)", path)
        return ret.outb, ret.err
    }
}

func validateOutput(path, op string) (action, updOp string, err error) {
    cmn.LogInfo("path(%s) o/p (%s)", path, string(op))

    d := map[string]any{}

    err = json.Unmarshal([]byte(op), &d)
    if err != nil {
        err = cmn.LogError("path(%s) o/p not JSON err(%v)", path, err)
    } else {
        for k, v := range d {
            if strings.ToLower(k) == "action" {
                if sval, ok := v.(string); !ok {
                    cmn.LogError("Expected string for action k(%s): v(%T)(%v)", k, v, v)
                } else {
                    action = sval
                }
                updOp = op
                break
            }
        }
        if action == "" {
            action = filepath.Base(path)
            d["Action"] = action
            data := []byte{}
            if data, err = json.Marshal(&d); err == nil {
                updOp = string(data)
            }
        }
    }
    return
}

/*
 * Runs the given path periodically.
 */

type errRet_t struct {
    Path        string
    Action      string
    RootCause   string
}

func (spl *ScriptBasedPlugin) runPlugin(path string, hbchan chan string) {
    defer spl.wg.Done()
    consecutiveErrs := 0
    errRet := errRet_t { Path: path }
    
    for {
        if consecutiveErrs >= spl.errConsecutive {
            errRet.RootCause = "Too many failures"
        } else if out, err := runAScript(path, spl.scrTimeout); err != nil {
            errRet.RootCause = fmt.Sprintf("Run failed: err(%v)", err)
            consecutiveErrs++
        } else if actionName, op, err := validateOutput(path, string(out)); err != nil {
            errRet.RootCause = fmt.Sprintf("Validate failed: Invalid o/p err(%v)", err)
            consecutiveErrs++
        } else {
            errRet.Action = actionName
            tele.PublishEvent(op)
            hbchan <- actionName
            consecutiveErrs = 0
        }
        if consecutiveErrs != 0 {
            /* A != 0 implies that this run failed. Report it */
            if op, err := json.Marshal(errRet); err != nil {
                cmn.LogPanic("Internal error: err(%v) errRet(%v)", err, errRet)
            } else {
                tele.PublishEvent(string(op))
            }
        }

        /* Sleep for pause period or until chClose is closed, whichever earlier */
        select {
        case <-time.After(time.Duration(spl.pausePeriod) * time.Second):
            cmn.LogDebug("%s: Pause (%d) seconds done", path, spl.pausePeriod)
            /* go for next run */
        case <-spl.stopSignal:
            cmn.LogInfo("%s: Exiting upon close signal", path)
            return
        }
    }
}

func (spl *ScriptBasedPlugin) Init(cfg *cmn.ActionCfg_t) (err error) {
    knobs := map[string]any{}

    spl.errConsecutive = SCR_MAX_CONSECUTIVE_ERR_CNT
    spl.scrTimeout = SCR_RUN_TIMEOUT
    spl.pausePeriod = SCR_PAUSE_PERIOD
    spl.scriptsPath = SCR_DEFAULT_PATH
    spl.heartbeatInt = cfg.HeartbeatInt

    if spl.heartbeatInt <= 0 {
         err = cmn.LogError("Invalid value for heartbeat (%d)", cfg.HeartbeatInt)
         return
     }

    if len(cfg.ActionKnobs) != 0 {
        if err = json.Unmarshal(cfg.ActionKnobs, &knobs); err != nil {
            err = cmn.LogError("Failed to unmarshal cfg.ActionKnobs (%s)", cfg.ActionKnobs)
            return
        }

        for k, v := range knobs {
            switch {
            case strings.ToLower(k) == "scriptspath":
                if s, ok := v.(string); ok {
                    spl.scriptsPath = s
                }
            case strings.ToLower(k) == "consecutiveerrcnt":
                spl.errConsecutive = cmn.ValidatedVal(v, SCR_MAX_CONSECUTIVE_ERR_CNT_MAX,
                    SCR_MAX_CONSECUTIVE_ERR_CNT_MIN, SCR_MAX_CONSECUTIVE_ERR_CNT,
                    "Script max consecutive errors")
            case strings.ToLower(k) == "scripttimeout":
                spl.scrTimeout = cmn.ValidatedVal(v, SCR_RUN_TIMEOUT_MAX,
                    SCR_RUN_TIMEOUT_MIN, SCR_RUN_TIMEOUT, "Script Run Timeout")
            case strings.ToLower(k) == "pauseperiod":
                spl.pausePeriod = cmn.ValidatedVal(v, SCR_PAUSE_PERIOD_MAX,
                    SCR_PAUSE_PERIOD_MIN, SCR_PAUSE_PERIOD, "Script Pause Period")
            }
        }
    }
    if spl.scriptsPath == "" {
        err = cmn.LogError("Init failed. Expect scriptsPath (%s)", spl.scriptsPath)
    } else if lst, e := cmn.ListFiles(spl.scriptsPath, scriptPatterns); e != nil {
        err = cmn.LogError("Init failed. Failed to read files pattern (%v) err(%v)", scriptPatterns, e)
    } else if len(lst) == 0 {
        err = cmn.LogError("Init failed. No file found in path (%s)", spl.scriptsPath)
    } else {
        spl.files = lst
        tele.PublishInit(tele.CHANNEL_PRODUCER_PLUGIN, plugin_name)
    }
    return
}

func (spl *ScriptBasedPlugin) Request(hbchan chan pcmn.PluginHeartBeat, request *ipc.ActionRequestData) *ipc.ActionResponseData {
    /* Stay for ever - calling each script periodically with configured periods */

    spl.wg = new(sync.WaitGroup)
    spl.stopSignal = make(chan struct{})
    hbLocal := make(chan string, len(spl.files))
    activeScripts := map[string]bool {}

    for _, path := range spl.files {
        spl.wg.Add(1)
        go spl.runPlugin(path, hbLocal)
    }

    /* Block here until shutdown with heartbeats */
    for {
        select {
        case s := <-hbLocal:
            activeScripts[s] = true

        case <-time.After(time.Duration(spl.heartbeatInt) * time.Second):
            /* TODO  send list of active scripts in heartbeat */
            hbchan <- pcmn.PluginHeartBeat{plugin_name, time.Now().Unix()}
            cmn.LogDebug("LoM: ActiveScripts (%v)", reflect.ValueOf(activeScripts).MapKeys())
            /* reset - So only active scripts since last heartbeat gets reported */
            activeScripts =  map[string]bool {}

        case <-spl.stopSignal:
            return nil
        }
    }
    return nil
}

func (spl *ScriptBasedPlugin) Shutdown() error {
    /* Signal the goroutine to stop by closing the stopSignal channel */
    if spl.stopSignal != nil {
        close(spl.stopSignal)
        spl.stopSignal = nil

        /* Wait for the goroutine to complete */
        spl.wg.Wait()
    }
    return nil
}

func (spl *ScriptBasedPlugin) GetPluginID() pcmn.PluginId {
    return pcmn.PluginId{
        Name:    plugin_name,
        Version: "1.0.0.0",
    }
}
