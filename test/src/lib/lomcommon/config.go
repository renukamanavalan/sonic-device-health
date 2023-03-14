package lomcommon

import (
    "encoding/json"
    "io/ioutil"
    "os"
)


const (
    Detection string = "Detection"
    SafetyCheck string = "SafetyCheck"
    Mitigation string = "Mitigation"
)

type GlobalConfig_t map[string]string

func (p *GlobalConfig_t) GetVal(key string) string {
    if s, ok := p[key]; ok {
        return s
    }
    switch(key) {
    case "MAX_SEQ_TIMEOUT_SECS":
        return "120"                /* Default of 2 mins */
    case "MIN_PERIODIC_LOG_PERIOD":
        return "15"
    case "ENGINE_HB_INTERVAL":
        return "10"
    default:
        return "UNKNOWN"
    }
}


type ActionCfg_t struct {
    Name            string
    Type            string
    Timeout         int     /* Timeout recommended for this action */
    HeartbeatInt    int     /* Heartbeat interval */
    Disable         bool    /* true - Disabled */
    Mimic           bool    /* true - Run but don't write/update device */
    ActionKnobs     string  /* Json String with action specific knobs */
}

type ActionsConfigList_t  map[string]ActionCfg_t

type BindingActionCfg_t struct {
    Name        string
    Mandatory   bool    /* Once sequence kicked off, mandatory to call this */
    /*
     * Timeout to use while in this sequence
     * <= 0 - means no timeout set.
     * >0   - timeout in seconds
     *
     */
    Timeout     int     /* Timeout to use while in this sequence */
    Sequence    int     /* Sequence index */
}

type BindingSequence_t struct {
    SequenceName    string
    Timeout         int     /*  >0   - timeout in seconds; else no timeout */
    Priority        int
    Actions         []BindingActionCfg_t
}

func (s *BindingSequence_t) Compare(d *BindingSequence_t) bool {
    if ((s.SequenceName != d.SequenceName) ||
            (s.Timeout != d.Timeout) ||
            (len(s.Actions) != len(d.Actions))) {
        return false
    }
     
    for i := 0; i < len(s.Actions); i++ {
        if s.Actions[i] != d.Actions[i] {
            return false
        }
    }
    return true
}
        
/* Binding sequence. Key = Name of first action. Value: Ordered actions first to last. */
type BindingsConfig_t map[string]BindingSequence_t


type ConfigMgr_t struct {
    globalConfig    *GlobalConfig_t
    actionsConfig   *ActionsConfigList_t
    bindingsConfig  *BindingsConfig_t
}

func (p *ConfigMgr_t) readGlobalsConf(fl string) error {
    v := make(map[string]any)

    jsonFile, err := os.Open(fl)
    if err != nil {
        return err
    }

    defer jsonFile.Close()

    if byteValue, err1 := ioutil.ReadAll(jsonFile); err1 != nil {
        return err1
    } else if err1 := json.Unmarshal(byteValue, &v); err1 != nil {
        return err1
    } else {
        for k, v := range v {
            s, ok := v.(string)
            if !ok {
                LogPanic("Global config %s:%s is not string but (%T)", k, v, v)
            }
            p.globalConfig[k] = s
        }
        return nil
    }
}


func (p *ConfigMgr_t) readActionsConf(fl string) error {
    actions := struct {
        Actions []ActionCfg_t
    }{}

    jsonFile, err := os.Open(fl)
    if err != nil {
        return err
    }

    defer jsonFile.Close()

    if byteValue, err1 := ioutil.ReadAll(jsonFile); err1 != nil {
        return err1
    } else if err1 := json.Unmarshal(byteValue, &actions); err1 != nil {
        return err1
    } else {
        for _, a := range actions.Actions {
            p.actionsConfig[a.Name] = a
        }
        return nil
    }
}


func (p *ConfigMgr_t) readBindingsConf(fl string) error {
    bindings := struct {
        Bindings []BindingSequence_t
    }{}

    jsonFile, err := os.Open(fl)
    if err != nil {
        return err
    }
    
    defer jsonFile.Close()
    
    if byteValue, err1 := ioutil.ReadAll(jsonFile); err1 != nil {
        return err1
    } else if err1 := json.Unmarshal(byteValue, &bindings); err1 != nil {
        return err1
    } else {
        for _, b := range bindings.Bindings {
            seq := 0
            firstAction := string("")
            ordered := make([]BindingActionCfg_t, 0, len(b.Actions))

            for i, a := range b.Actions {
                actInfo, ok := p.actionsConfig[a.Name]
                if !ok {
                    return LogError("%s: %d: Failed to get conf for action (%s)",
                            fl, i, a.Name)
                }
                if i == 0 {
                    seq = a.Sequence
                    firstAction = a.Name
                } else if (seq == a.Sequence) {
                    return LogError("%s: %d: Duplicate sequence (%d/%s) vs (%d/%s)",
                            fl, i, seq, firstAction, a.Sequence, a.Name)
                } else if (seq > a.Sequence) {
                    seq = a.Sequence
                    firstAction = a.Name
                }
                if a.Timeout == 0 {
                    a.Timeout = actInfo.Timeout
                }
            }
            if len(b.Actions) > 0 {
                /* Sort by sequence */
                sort.Slice(b.Actions, func(i, j int) bool {
                    return b.Actions[i].Sequence < b.Actions[j].Sequence
                })
                if b.Timeout == 0 {
                    s := p.GetGlobalCfg("MAX_SEQ_TIMEOUT_SECS")
                    if t, e := strconv.Atoi(s); e != nil {
                        LogError("COnfig Error: Failed to convert MAX_SEQ_TIMEOUT_SECS=%s to int (%v)", s, e)
                        b.Timeout = 120
                    } else {
                        b.Timeout = t
                    }
                }
                p.bindingsConfig[firstAction] = b
            } else {
                LogError("Internal Error: Missing actions in bindings for (%s) fl(%s)",
                        b.SequenceName, jsonFile)
            }
        }
        return nil
    }
}



func (p *ConfigMgr_t) loadConfigFiles(globals_fl, actions_fl string, bind_fl string) error {
    if err := p.readGlobalsConf(globals_fl); err != nil {
        return LogError("Actions: %s: %v", actions_fl, err)
    } 
    if err := p.readActionsConf(actions_fl); err != nil {
        return LogError("Actions: %s: %v", actions_fl, err)
    } 
    if err := p.readBindingsConf(bind_fl); err != nil {
        return LogError("Bind: %s: %v", bind_fl, err)
    } 
    return nil
}

func (p *ConfigMgr_t) GetGlobalCfg(key string) string {
    return globalConfig.GetVal(key)
}


func (p *ConfigMgr_t) IsStartSequenceAction(name string) bool {
    /* Return true, if action is start of any sequence; else false */
    _, ok := p.bindingsConfig[name]
     return ok
}

func (p *ConfigMgr_t) GetSequence(name string) (*BindingSequence_t, error) {
    ret := &BindingSequence_t{}

    v, ok := p.bindingsConfig[name]
    if !ok {
        return nil, LogError("Failed to find sequence for (%s)", name)
    }

    /* Copy primitives and deep copy actions slice */
    *ret = v
    ret.Actions = make([]BindingActionCfg_t, len(v.Actions))
    copy(ret.Actions, v.Actions)
    
    return ret, nil
}


func (p *ConfigMgr_t) GetActionConfig(name string) (*ActionCfg_t, error) {
    actInfo, ok := p.actionsConfig[name]
    if !ok {
        return nil, LogError("Failed to get conf for action (%s)", name)
    }
    return &actInfo, nil
}

func (p *ConfigMgr_t) GetActionsList() map[string]struct{IsAnomaly bool} {

    ret := make(map[string]struct{IsAnomaly bool})

    for k, _ := range p.actionsConfig {
        _, ok := p.bindingsConfig[k]
        ret[k] = struct{IsAnomaly bool} { ok }
    }
    return ret
}


var configMgr *ConfigMgr_t = nil

func GetConfigMgr() *ConfigMgr_t {
    return configMgr
}


func InitConfigMgr(global_fl, actions_fl, bind_fl string) (*ConfigMgr_t, error) {
    t := &ConfigMgr_t{make(GlobalConfig_t), make(ActionsConfigList_t), make(BindingsConfig_t)}

    if err := t.loadConfigFiles(global_fl, actions_fl, bind_fl); err != nil {
        return nil, err
    } else {
        configMgr = t
        return t, nil
    }
}


