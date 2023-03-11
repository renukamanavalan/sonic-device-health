package engine

import (
    . "lib/lomcommon"
    . "lib/lomipc"
    "os"
    "os/signal"
    "syscall"
)

/*
 * If missed in config, this is the default to use, which is a max of 2 mins
 * The timeout starts from the point of receiving detection info to receiving
 * response for the last action in sequence.
 */
const MAX_SEQ_TIMEOUT_SECS = EpochSecs(120)

/* Action info is collected from register calls from clients */
type ActiveActionInfo_t {
    Action      ActionName_t
    Client      ClientName_t    /* Client that registered this action
    /*
     * Timeout to use if not set in sequence 
     */
    Timeout     int
    /* For now, Engine does not need any other config, hence not saved */
}

type ActiveClientInfo_t struct {

    /* List of registered actions by this client */
    Actions     map[ActionName_t]struct{}

type ClientRegistrations_t struct {
    activeActions   map[ActionName_t]*ActiveActionInfo_t
    activeClients   map[ClientName_t]map[ActionName_t]struct{}

    heartbeats      map[ActionName_t]EpochSecs 
}

/* Initialized object; But not exported */
var clientRegistrations *ClientRegistrations_t = nil


func GetRegistrations() *ClientRegistrations_t {
    return clientRegistrations
}


func InitRegistrations() *ClientRegistrations_t {
    clientRegistrations = &ClientRegistrations_t{
                make(map[ActionName_t]*ActiveActionInfo_t)
                make(map[ClientName_t]map[ActionName_t]struct{})
                make(map[ActionName_t]EpochSecs)
            }
    return clientRegistrations
}

func (p *ClientRegistrations_t) RegisterClient(name ClientName_t) error {
    if _, ok := p.ActiveClients[name]; ok {
        LogError("%s: Duplicate client registration", name)
        p.DeregisterClient(name)
    }
    p.ActiveClients[name] = make(map[ActionName_t]struct{})
    return nil
}

func (p *ClientRegistrations_t) RegisterAction(action *ActiveActionInfo_t) error {
    if cl, ok := p.ActiveClients[action.Client]; !ok {
        return LogError("%s: Missing client registration", action.Client)
    }
    else if r, ok := p.ActiveActions[action.Action]; ok {
        return LogError("%s: Duplicate action registration cl:(%s)", action.Action, r.Client)
    }
    else if cfg, err := GetConfigMgr().GetActionConfig(action.Action); err != nil {
        return LogError("%s: Missing action config", action.Action)
    } else if cfg.Disable {
        return LogError("%s: is disabled in config", action.Action)
    } else {
        if action.Timeout == 0 {
            action.Timeout = cfg.Timeout
        }
        cl[action.Action] = struct{}{}

        /* Make a copy and save */
        ret = &ActiveActionInfo_t{}
        *ret =  *action
        p.ActiveActions[action.Action] = ret
        return nil
    }
}

func (p *ClientRegistrations_t) GetActionInfo(name ActionName_t) *ActiveActionInfo_t {
    if r, ok := p.ActiveActions[action.Action]; ok {
        /* return a new copy */
        ret := &ActiveActionInfo_t{}
        *ret = *r
        return ret
    } else {
        return nil
    }
}


func (p *ClientRegistrations_t) DeregisterClient(name ClientName_t) {
    if cl, ok := p.ActiveClients[name]; !ok {
        return
    }

    /*
     * Delete client first, so its map will not be touched by deregister
     * action. If not, we would need to make a copy to walk and de-register
     */
    delete (p.ActiveClients, name)

    
    for k, _ := range cl {
        p.DeregisterAction(k)
    }
}

func (p *ClientRegistrations_t) DeregisterAction(actName ActionName_t) {
    if r, ok := p.ActiveActions[actName]; !ok {
        return
    } else if cl, ok := p.ActiveClients[r.Client]; ok {
        delete (cl, actName)
    }
    delete (p.ActiveActions, actName)
}

func (p *ClientRegistrations_t) NotifyHeartbeats(actName ActionName_t,
            ts EpochSecs) {
    if r, ok := p.ActiveActions[actName]; ok {
        heartbeats[actName] = ts
    }
}

func (p *ClientRegistrations_t) GetResetHeartbeats() map[ActionName_t]EpochSecs {
    r := p.heartbeats
    p.heartbeats = make(map[ActionName_t]EpochSecs)
    return r
}



type ActionRequestCacheData struct {
    Action  string
    InstanceId id
}

/* Constructed upon response from first action 
 * The sequence available in config at the 
type SequenceState_t struct {
    FirstAction             ActionName_t    /* First in sequence */
    AnomalyInstanceId       string          /* Id referred in all requests & responses for this seq */
    sequence                *ActionBindingSequence_t /* Cache it, as it might change during a sequence */
    SeqStartEpoch           EpochSecs       /* Start of the sequence */
    SeqExpEpoch             EpochSecs       /* Timepoint of expiry for sequence */
    ReqExpEpoch             EpochSecs       /* Expiry timepoint for current active request */
    CtIndex                 int             /* Index of action in progress */
    Context                 []ActionResponseData /* Ordered responses as received */
    Requests                []ActionRequestCacheData /* Ordered requests as sent */
}

func (p *SequenceState_t) GetSequence() error {
    if len(p.FirstAction) == 0 {
        return LogError("Expect non null FirstAction")
    }
    if len(p.AnomalyInstanceId) == 0 {
        return LogError("Expect non null AnomalyInstanceId")
    }
    if !IsStartSequenceAction(p.FirstAction) {
        return LogError("%s is not a first action in any sequence", p.FirstAction)
    }
    b, err := GetSequence(p.FirstAction)
    if err != nil {
        return err
    }
    p.sequence = b
    tout := MAX_SEQ_TIMEOUT_SECS
    if b.Timeout > 0 {
        tout = EpochSecs(b.Timeout)
    } else {
        LogError("%s: Binding sequence has no timeout", b.SequenceName)
    }

    p.SeqStartEpoch = time.Now().Unix()
    p.SeqExpEpoch = p.SeqStartEpoch + tout
}


const (
    SIG_NONE = iota
    SIG_HUP
    SIG_INT
    SIG_TERM
)

type SigReceived_t int

func sigHandler(chAlert chan interface{}) {
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        for {
            select {
            case sig:= <- sigs:
                switch(sig) {
                case syscall.SIGHUP:
                    chAlert <- SigReceived(SIG_HUP)
                case syscall.SIGINT:
                    chAlert <- SigReceived(SIG_INT)
                case syscall.SIGTERM:
                    chAlert <- SigReceived(SIG_TERM)
                    return
                default:
                    log_error("Internal ERROR: Unknown signal received (%v)", sig)
                }
            }
        }
    }()
}


