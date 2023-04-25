package lomcommon

import (
    "encoding/json"
    "io"
    "os"
    "path/filepath"
    "sort"
)


const (
    /* Global constants */
    ENGINE_HB_INTERVAL_SECS = "ENGINE_HB_INTERVAL_SECS"
    MAX_SEQ_TIMEOUT_SECS = "MAX_SEQ_TIMEOUT_SECS"
    MIN_PERIODIC_LOG_PERIOD_SECS = "MIN_PERIODIC_LOG_PERIOD_SECS"
)

const (
    GLOBALS_CONF_FILE = "globals.conf.json"
    ACTIONS_CONF_FILE = "actions.conf.json"
    BINDINGS_CONF_FILE = "bindings.conf.json"
)

type ConfigFiles_t struct {
    GlobalFl    string
    ActionsFl   string
    BindingsFl  string
}

const (
    Detection string = "Detection"
    SafetyCheck string = "SafetyCheck"
    Mitigation string = "Mitigation"
)

/*
 * Classified into different types, for the convenience of caller.
 * For example, converting from string to int is pre-done
 */
type GlobalConfig_t struct {
    strings map[string]string
    ints    map[string]int
    anyVal  map[string]any
}


/*
 * NOTE: This will be deprecated soon.
 * Guideline: conf should have a value for every entry
 */
func (p *GlobalConfig_t) setDefaults() {
    p.strings = make(map[string]string)
    p.ints = make(map[string]int)
    p.anyVal = make(map[string]any)

    p.ints["MAX_SEQ_TIMEOUT_SECS"] = 120
    p.ints["MIN_PERIODIC_LOG_PERIOD_SECS"] = 15
    p.ints["ENGINE_HB_INTERVAL_SECS"] = 10
}

func (p *GlobalConfig_t) readGlobalsConf(fl string) error {
    p.setDefaults()

    v := make(map[string]any)

    jsonFile, err := os.Open(fl)
    if err != nil {
        LogError("Failed to open (%s) (%v)", fl, err)
        return err
    }

    defer jsonFile.Close()

    if byteValue, err := io.ReadAll(jsonFile); err != nil {
        return LogError("Failed to read (%s) (%v)", jsonFile, err)
    } else if err := json.Unmarshal(byteValue, &v); err != nil {
        return LogError("Failed to parse (%s) (%v)", jsonFile, err)
    } else {
        for k, v := range v {
            p.anyVal[k] = v
            if s, ok := v.(string); ok {
                p.strings[k] = s
            } else if f, ok := v.(float64); ok {
                p.ints[k] = int(f)
            }

        }
        return nil
    }
}

/*
 * Get config value for given key as string. If value in config
 * is not string type or if this key is unknown, an empty string
 * is returned.
 *
 * Input:
 *  key: Config key. 
 *
 * Output:
 *  None
 *
 * Return:
 *  o/p as string
 */
func (p *GlobalConfig_t) GetValStr(key string) string {
    return p.strings[key]
}

/*
 * Get config value for given key as int. If value in config
 * is not int type or if this key is unknown, a default of 0
 * is returned.
 *
 * Input:
 *  key: Config key. 
 *
 * Output:
 *  None
 *
 * Return:
 *  o/p as int
 */
func (p *GlobalConfig_t) GetValInt(key string) int {
    return p.ints[key]
}

/*
 * Get config value for given key as any. If value in config
 * it is returned as any type or if this key is unknown, an empty i/f
 * is returned.
 *
 * Input:
 *  key: Config key. 
 *
 * Output:
 *  None
 *
 * Return:
 *  o/p as any
 */
func (p *GlobalConfig_t) GetValAny(key string) any {
    return p.anyVal[key]
}


/* Action config as read from actions.conf.json */
type ActionCfg_t struct {
    Name            string
    Type            string
    Timeout         int     /* Timeout recommended for this action */
    HeartbeatInt    int     /* Heartbeat interval */
    Disable         bool    /* true - Disabled */
    Mimic           bool    /* true - Run but don't write/update device */
    ActionKnobs     string  /* Json String with action specific knobs */
}

/* Map with action name */
type ActionsConfigList_t  map[string]ActionCfg_t

/* Action entry in sequence from binding sequence config */
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

/* Entire single binding sequence */
type BindingSequence_t struct {
    SequenceName    string
    Timeout         int     /*  >0   - timeout in seconds; else no timeout */
    Priority        int
    Actions         []*BindingActionCfg_t
}

/* Helper to compare two sequences. Return true on match, else false */
func (s *BindingSequence_t) Compare(d *BindingSequence_t) bool {
    if s == d {
        /* Same ptr */
        return true
    }
    if (s == nil) || (d == nil) {
        LogError("Unexpected nil args self(%v) arg(%v)\n", (s == nil), (d == nil))
        return false
    }

    if ((s.SequenceName != d.SequenceName) ||
            (s.Timeout != d.Timeout) ||
            (len(s.Actions) != len(d.Actions))) {
        return false
    }
     
    for i := 0; i < len(s.Actions); i++ {
        if *(s.Actions[i]) != *(d.Actions[i]) {
            return false
        }
    }
    return true
}
        
/* Binding sequence. Key = Name of first action. Value: Ordered actions first to last. */
type BindingsConfig_t map[string]BindingSequence_t


/* ConfigMgr - A single stop for all configs */
type ConfigMgr_t struct {
    globalConfig    *GlobalConfig_t
    actionsConfig   ActionsConfigList_t
    bindingsConfig  BindingsConfig_t
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

    if byteValue, err := io.ReadAll(jsonFile); err != nil {
        return err
    } else if err := json.Unmarshal(byteValue, &actions); err != nil {
        return err
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
    
    if byteValue, err := io.ReadAll(jsonFile); err != nil {
        return err
    } else if err := json.Unmarshal(byteValue, &bindings); err != nil {
        return err
    } else {
        for _, b := range bindings.Bindings {
            seq := 0
            firstAction := string("")

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
                    b.Timeout = p.GetGlobalCfgInt("MAX_SEQ_TIMEOUT_SECS")
                }
                p.bindingsConfig[firstAction] = b
            } else {
                return LogError("Internal Error: Missing actions in bindings for (%s) fl(%s)",
                        b.SequenceName, jsonFile)
            }
        }
        return nil
    }
}



func (p *ConfigMgr_t) loadConfigFiles(cfgFiles *ConfigFiles_t) error {
    if err := p.globalConfig.readGlobalsConf(cfgFiles.GlobalFl); err != nil {
        return LogError("Globals: %s: %v", cfgFiles.GlobalFl, err)
    } 
    if err := p.readActionsConf(cfgFiles.ActionsFl); err != nil {
        return LogError("Actions: %s: %v", cfgFiles.ActionsFl, err)
    } 
    if err := p.readBindingsConf(cfgFiles.BindingsFl); err != nil {
        return LogError("Bind: %s: %v", cfgFiles.BindingsFl, err)
    } 
    return nil
}

/*
 * Get global config value for given key as string. If value in config
 * is not string type or if this key is unknown, an empty string
 * is returned.
 *
 * Input:
 *  key: Config key. 
 *
 * Output:
 *  None
 *
 * Return:
 *  o/p as string
 */
func (p *ConfigMgr_t) GetGlobalCfgStr(key string) string {
    return p.globalConfig.GetValStr(key)
}

/*
 * Get global config value for given key as int. If value in config
 * is not int type or if this key is unknown, a default of 0
 * is returned.
 *
 * Input:
 *  key: Config key. 
 *
 * Output:
 *  None
 *
 * Return:
 *  o/p as int
 */
func (p *ConfigMgr_t) GetGlobalCfgInt(key string) int {
    return p.globalConfig.GetValInt(key)
}

/*
 * Get global config value for given key as any. If value in config
 * it is returned as any type or if this key is unknown, an empty i/f
 * is returned.
 *
 * Input:
 *  key: Config key. 
 *
 * Output:
 *  None
 *
 * Return:
 *  o/p as any
 */
func (p *ConfigMgr_t) GetGlobalCfgAny(key string) any {
    return p.globalConfig.GetValAny(key)
}


/*
 * IsStartSequenceAction
 *  Check if given action is first action of any binding sequence
 *
 * Input:
 *  name - Name of the action to test.
 *
 * Output:
 *  None
 *
 * Return:
 *  true - It is indeed first action of a sequence
 *  false - If not, first action of a sequence
 */
func (p *ConfigMgr_t) IsStartSequenceAction(name string) bool {
    /* Return true, if action is start of any sequence; else false */
    _, ok := p.bindingsConfig[name]
     return ok
}

/*
 * GetSequence
 *  If given action is first action of any binding sequence, it is returned.
 *  Else null with non nil error.
 *
 * Input:
 *  name - Name of the action 
 *
 * Output:
 *  None
 *
 * Return:
 *  sequence - If first action of a sequence, non-nil ptr to seq object, else nil
 *  error - If not, first action of a sequence, return non nil error, else nil
 */
func (p *ConfigMgr_t) GetSequence(name string) (*BindingSequence_t, error) {
    ret := &BindingSequence_t{}

    v, ok := p.bindingsConfig[name]
    if !ok {
        return nil, LogError("Failed to find sequence for (%s)", name)
    }

    /* Copy primitives and deep copy actions slice */
    *ret = v
    ret.Actions = make([]*BindingActionCfg_t, len(v.Actions))
    copy(ret.Actions, v.Actions)
    
    return ret, nil
}


/*
 * GetActionConfig
 *  Return config as read from actions.conf for the given action.
 *  Else null with non nil error.
 *
 * Input:
 *  name - Name of the action 
 *
 * Output:
 *  None
 *
 * Return:
 *  config - If present in conf file, return it, else nil
 *  error - If not in actions config, return non nil error, else nil
 */
func (p *ConfigMgr_t) GetActionConfig(name string) (*ActionCfg_t, error) {
    actInfo, ok := p.actionsConfig[name]
    if !ok {
        return nil, LogError("Failed to get conf for action (%s)", name)
    }
    return &actInfo, nil
}

/* TODO: Goutham's PR has this */
type ProcCfg_t struct {}
func (p *ConfigMgr_t) GetProcConfig(name string) (*ProcCfg_t, error) {
    return nil, LogError("TODO: Yet to implement")
}


/*
 * GetActionsList
 *  Return list of all actions from config, with a flag indicating if that
 *  action is anomaly or not. IsAnomaly is set to true, if first action
 *  of any sequence
 *
 * Input:
 *  None
 *
 * Output:
 *  None
 *
 * Return:
 *  List of all actions with a flag for each.
 */
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


/*
 * Initialize config or re-refresh config from files.
 *
 * Input:
 *  global_fl - Global config file
 *  actions_fl - Actions config file
 *  bind_fl - Bindings config file
 *
 * Output:
 *  None
 *
 * Return:
 *  ConfigMgr - Non nil instance, if successful, else nil
 *  error - Non nil on any failure, else nil
 */
func InitConfigMgr(p *ConfigFiles_t) (*ConfigMgr_t, error) {
    t := &ConfigMgr_t{new(GlobalConfig_t), make(ActionsConfigList_t), make(BindingsConfig_t)}

    if err := t.loadConfigFiles(p); err != nil {
        return nil, err
    } else {
        configMgr = t
        return t, nil
    }
}


func InitConfigPath(path string) error {
    cfgPath := path
    if len(path) == 0 {
        if p, err := os.Getwd(); err != nil {
            return LogError("Failed to get current working dir (%v)", err)
        } else {
            cfgPath = p
        }
    }
    cfgFiles := &ConfigFiles_t {
        GlobalFl: filepath.Join(cfgPath, GLOBALS_CONF_FILE),
        ActionsFl: filepath.Join(cfgPath, ACTIONS_CONF_FILE),
        BindingsFl: filepath.Join(cfgPath, BINDINGS_CONF_FILE),
    }

    _, err := InitConfigMgr(cfgFiles)
    return err
}

