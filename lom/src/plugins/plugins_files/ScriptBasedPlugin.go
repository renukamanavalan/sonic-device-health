/*
 * A skeleton to plugin to run all binaries from spefic folder periodically forever.
 */

package plugins_files

import (
    "context"
    "encoding/json"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"
    "time"

    cmn "lom/src/lib/lomcommon"
    ipc "lom/src/lib/lomipc"
    pcmn "lom/src/plugins/plugins_common"
)

type ScriptBasedPlugin struct {
    /* ... Internal plugin data */
    cfg         *cmn.ActionCfg_t
    scriptsPath string
    files       []string
    scrTimeout  int
    pausePeriod int
    wg          *sync.WaitGroup
    stopSignal  chan struct{} /* Channel to signal goroutine to stop */
}

const SCR_RUN_TIMEOUT_MIN = (1 * 60)
const SCR_RUN_TIMEOUT_MAX = (5 * 60)
const SCR_RUN_TIMEOUT = (3 * 60)

const SCR_PAUSE_TIMEOUT_MIN = (1 * 60)
const SCR_PAUSE_TIMEOUT_MAX = (5 * 60)
const SCR_PAUSE_TIMEOUT = (5 * 60)
const SCR_DEFAULT_PATH = "/usr/share/lom/scripts"

var scriptPatterns = []string{
    "_pl_script\\.",
}

func NewScriptBasedPlugin(...interface{}) pcmn.Plugin {
    /* ... create and return a new instance of this Plugin */
    return &ScriptBasedPlugin{}
}

func init() {
    /* ... register the plugin with plugin manager */
    pcmn.RegisterPlugin("ScriptBasedPlugin", NewScriptBasedPlugin)
}

func runAScript(path string, timeout int) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
    defer cancel()

    return exec.CommandContext(ctx, path).Output()
}

func validateOutput(path string, op []byte) (action string, updOp any, err error) {
    cmn.LogInfo("path(%s) o/p (%s)", path, string(op))

    d := map[string]any{}

    err = json.Unmarshal(op, &d)
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
            updOp = d
        }
    }
    return
}

/*
 * Runs the given path periodically.
 */
func (spl *ScriptBasedPlugin) runPlugin(path string, hbchan chan pcmn.PluginHeartBeat) {
    defer spl.wg.Done()

    for {
        /* Run the script with
           if out, err := runAScript(path, spl.scrTimeout); err != nil {
               cmn.LogError("%s: Failed: err(%v)", path, err)
           } else if plName, op, err := validateOutput(path, out); err != nil {
               cmn.LogError("%s: Failed: Invalid err(%v)", path, err)
           } else {
               hbchan <- PluginHeartBeat { plName, time.Now().Unix() }
               tele.PublishEvent(op)
           }

           /* Sleep for pause time or until chClose is closed, whichever earlier */
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

func (spl *ScriptBasedPlugin) Init(actionCfg *cmn.ActionCfg_t) (err error) {
    knobs := map[string]any{}

    cmn.LogInfo("ScriptBasedPlugin Init called")
    spl.scrTimeout = SCR_RUN_TIMEOUT
    spl.pausePeriod = SCR_PAUSE_TIMEOUT
    spl.scriptsPath = SCR_DEFAULT_PATH

    s := []byte{}
    if s, err = json.Marshal(actionCfg.ActionKnobs); err != nil {
        err = cmn.LogError("Failed to marshal actionCfg.ActionKnobs (%v)", actionCfg.ActionKnobs)
        return
    } else if string(s) != "" {
        if err = json.Unmarshal(s, &knobs); err != nil {
            err = cmn.LogError("Failed to unmarshal actionCfg.ActionKnobs (%s)", s)
            return
        }

        for k, v := range knobs {
            if s, ok := v.(string); ok && (s != "") {
                switch {
                case strings.ToLower(k) == "scriptspath":
                    spl.scriptsPath = s
                case strings.ToLower(k) == "scripttimeout":
                    spl.scrTimeout = cmn.ValidatedVal(s, SCR_RUN_TIMEOUT_MAX,
                        SCR_RUN_TIMEOUT_MIN, SCR_RUN_TIMEOUT, "Script Run Timeout")
                case strings.ToLower(k) == "pausePeriod":
                    spl.pausePeriod = cmn.ValidatedVal(s, SCR_PAUSE_TIMEOUT_MAX,
                        SCR_PAUSE_TIMEOUT_MIN, SCR_PAUSE_TIMEOUT, "Script Pause Timeout")
                }
            }
        }
    }
    if spl.scriptsPath == "" {
        err = cmn.LogError("Init failed. Expect scriptsPath (%s)", s)
    } else if lst, err := cmn.ListFiles(spl.scriptsPath, scriptPatterns); err != nil {
        err = cmn.LogError("Init failed. Failed to read files err(%v)", err)
    } else if len(lst) == 0 {
        err = cmn.LogError("Init failed. No file found in path (%s)", spl.scriptsPath)
    } else {
        spl.files = lst
        spl.cfg = actionCfg
    }
    return
}

func (spl *ScriptBasedPlugin) Request(hbchan chan pcmn.PluginHeartBeat, request *ipc.ActionRequestData) *ipc.ActionResponseData {
    /* Stay for ever - calling each script periodically with configured periods */

    spl.wg = new(sync.WaitGroup)
    spl.stopSignal = make(chan struct{})

    for _, path := range spl.files {
        spl.wg.Add(1)
        go spl.runPlugin(path, hbchan)
    }
    /* Block here until shutdown */
    <-spl.stopSignal
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
        Name:    "ScriptBasedPlugin",
        Version: "1.0.0.0",
    }
}
