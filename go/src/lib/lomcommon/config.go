package lomcommon

import (
    "encoding/json"
    "io/ioutil"
    "os"
)


type ClientName_t string
type ActionName_t string
type ActionType_t string
type ActionKnobJson_t string    /* Json String with action specific knobs */

const (
    Detection ActionType_t = "Detection"
    SafetyCheck ActionType_t = "SafetyCheck"
    Mitigation ActionType_t = "Mitigation"
)

type ActionInfo_t struct {
    Name            ActionName_t
    Type            ActionType_t
    Timeout         int     /* Timeout recommended for this action */
    HeartbeatInt    int     /* Heartbeat interval */
    Disable         bool    /* true - Disabled */
    Mimic           bool    /* true - Run but don't write/update device */
    ActionKnobs     ActionKnobJson_t
}

type ActionsConfigList_t  map[ActionName_t]ActionInfo_t

type BindingActionInfo_t struct {
    Name        ActionName_t
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
    Actions         []BindingActionInfo_t
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
type BindingsConfig_t map[ActionName_t]BindingSequence_t


type ConfigMgr_t struct {
    bindingsConfig BindingsConfig_t
    actionsConfig  ActionsConfigList_t
}

func (p *ConfigMgr_t) readActionsConf(fl string) error {
    actions := struct {
        Actions []ActionInfo_t
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
            firstAction := ActionName_t("")
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
            p.bindingsConfig[firstAction] = b
        }
        return nil
    }
}



func (p *ConfigMgr_t) loadConfigFiles(actions_fl string, bind_fl string) error {
    if err := p.readActionsConf(actions_fl); err != nil {
        return LogError("Actions: %s: %v", actions_fl, err)
    } 
    if err := p.readBindingsConf(bind_fl); err != nil {
        return LogError("Bind: %s: %v", bind_fl, err)
    } 
    return nil
}

func (p *ConfigMgr_t) IsStartSequenceAction(name ActionName_t) bool {
    /* Return true, if action is start of any sequence; else false */
    _, ok := p.bindingsConfig[name]
     return ok
}

func (p *ConfigMgr_t) GetSequence(name ActionName_t) (*BindingSequence_t, error) {
    ret := &BindingSequence_t{}

    v, ok := p.bindingsConfig[name]
    if !ok {
        return nil, LogError("Failed to find sequence for (%s)", name)
    }

    /* Copy primitives and deep copy actions slice */
    *ret = v
    ret.Actions = make([]BindingActionInfo_t, len(v.Actions))
    copy(ret.Actions, v.Actions)
    
    return ret, nil
}


func (p *ConfigMgr_t) GetActionConfig(name ActionName_t) (*ActionInfo_t, error) {
    actInfo, ok := p.actionsConfig[name]
    if !ok {
        return nil, LogError("Failed to get conf for action (%s)", name)
    }
    return &actInfo, nil
}

func (p *ConfigMgr_t) GetActionsList() map[ActionName_t]struct{IsAnomaly bool} {

    ret := make(map[ActionName_t]struct{IsAnomaly bool})

    for k, _ := range p.actionsConfig {
        _, ok := p.bindingsConfig[k]
        ret[k] = struct{IsAnomaly bool} { ok }
    }
    return ret
}

func GetConfigMgr(actions_fl string, bind_fl string) (*ConfigMgr_t, error) {
    t := &ConfigMgr_t{}
    t.actionsConfig = make(ActionsConfigList_t)
    t.bindingsConfig = make(BindingsConfig_t)

    if err := t.loadConfigFiles(actions_fl, bind_fl); err != nil {
        return nil, err
    } else {
        return t, nil
    }
}


