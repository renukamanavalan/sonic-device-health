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
     * 0    - means no timeout set.
     * -1   - means run w/o timeout.
     * >0   - timeout in seconds
     *
     */
    Timeout     int     /* Timeout to use while in this sequence */
    Sequence    int     /* Sequence index */
}

type BindingSequence_t struct {
    SequenceName    string
    Timeout         int
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


/* Unexported globals */

var bindingsConfig = BindingsConfig_t{}
var actionsConfig = ActionsConfigList_t{}


func readActionsConf(fl string) error {
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
        for _, p := range actions.Actions {
            actionsConfig[p.Name] = p
        }
        return nil
    }
}


func readBindingsConf(fl string) error {
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
        for _, p := range bindings.Bindings {
            seq := 0
            firstAction := ActionName_t("")
            for i, a := range p.Actions {
                actInfo, ok := actionsConfig[a.Name]
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
            bindingsConfig[firstAction] = p
        }
        return nil
    }
}



func LoadConfigFiles(actions_fl string, bind_fl string) error {
    bindingsConfig = BindingsConfig_t{}
    actionsConfig = ActionsConfigList_t{}

    if err := readActionsConf(actions_fl); err != nil {
        return LogError("Actions: %s: %v", actions_fl, err)
    } 
    if err := readBindingsConf(bind_fl); err != nil {
        return LogError("Bind: %s: %v", bind_fl, err)
    } 
    return nil
}

func IsStartSequenceAction(name ActionName_t) bool {
    /* Return true, if action is start of any sequence; else false */
    _, ok := bindingsConfig[name]
     return ok
}

func GetSequence(name ActionName_t) (*BindingSequence_t, error) {
    ret := &BindingSequence_t{}

    v, ok := bindingsConfig[name]
    if !ok {
        return nil, LogError("Failed to find sequence for (%s)", name)
    }

    /* Copy primitives and deep copy actions slice */
    *ret = v
    LogDebug("*ret=(%v)", *ret)
    LogDebug("v=(%v)", v)
    ret.Actions = make([]BindingActionInfo_t, len(v.Actions))
    copy(ret.Actions, v.Actions)
    
    return ret, nil
}


func GetActionConfig(name ActionName_t) (*ActionInfo_t, error) {
    actInfo, ok := actionsConfig[name]
    if !ok {
        return nil, LogError("Failed to get conf for action (%s)", name)
    }
    return &actInfo, nil
}

func GetActionsList() map[ActionName_t]struct{IsAnomaly bool} {

    ret := make(map[ActionName_t]struct{IsAnomaly bool})

    for k, _ := range actionsConfig {
        _, ok := bindingsConfig[k]
        ret[k] = struct{IsAnomaly bool} { ok }
    }
    return ret
}

