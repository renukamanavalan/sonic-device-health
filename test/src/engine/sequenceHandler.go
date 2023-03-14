package engine

import (
    "encoding/json"
    . "lib/lomcommon"
    . "lib/lomipc"
)

/*
 * Active requests - Requests sent out awaiting response with no timeout/sequence
 * First action in sequence -- In other works detection action requests
 */
type ActiveRequest_t struct {
    req         *ServerRequestData
    anomalyID   string
    ReqExpEpoch EpochSecs   /* Expiry time. A value of 0 means no expiry */
}

/*
 * Sequence is kicked off when first request of a sequence responds.
 *
 * There can be many sequences. But only one sequence can be active.
 *
 * SequenceInfo and related data is read & filled upon sequence creation
 * which is when the first action of the sequence responds.
 * 
 * Only one sequence can be active. Any sequence created when one is active
 * is added to pending Q. Going active implies invoking other actions of the
 * sequence, sequentially. Sequence remains active until all actions of the
 * sequence are complete or timeout, whichever comes earlier.
 *
 * When there are multiple sequences ready to go active, they are first sorted
 * by Priority and then by order of arrival within same priority
 *
 * All sequences have timeout. Some sequence may timeout, even before going 
 * active.
 *
 */
type SequenceState_t struct {
    AnomalyInstanceId       string          /* Id referred in all requests & responses for this seq */
    sequence                *BindingSequence_t /* Read from config; Cache as config may change. */
    SeqStartEpoch           EpochSecs       /* Start of the sequence */
    SeqExpEpoch             EpochSecs       /* Timepoint of expiry for sequence */
    CtIndex                 int             /* Index of action in progress */
    Context                 []*ActionResponseData /* Ordered responses as received */
    CurrentRequest          *ActiveRequest_t /* Current request in progress */
}

func (p *SequenceState_t) ExpiryEpoch() EpochSecs {
    if p.ActiveRequest_t != nil {
        return p.ActiveRequest_t.ReqExpEpoch
    } else {
        return p.SeqExpEpoch
    }
}

/*
 * Action vs request -- All active requests
 *
 * NOTE: Only one outstanding request per Action
 * Hence can be keyed off by action.
 */
type ActiveRequestsList_t map[string]*ActiveRequest_t


/*
 * map<anomaly id>sequence_state_t
 *
 * Sequences are keyed off of anomaly instance Id 
 */
type Sequences_t map<string>*SequenceState_t

/*
 * timeout:
 * Two possible candidates:
 *      1. Currently active request for currently active sequence
 *      2. All pending sequences.
 */
type SortedSequences_t []SequenceState_t

func GetSortedByExpiry(lst *Sequences_t) *SortedSequences_t {
    ret := make([]*SequenceState_t, 0, len(lst))

    for _, v := range lst {
        ret = append(ret, v)
    }
    sort.Slice(ret, func(i, j int) bool {
        return ret[i].ExpiryEpoch < ret[j].ExpiryEpoch
    }
    return ret
}


type SeqHandler_t struct {
    activeRequests          *ActiveRequestsList_t

    sequencesByAnomaly      *Sequences_t
    sequencesByFirstAction  *Sequences_t

    sortedSequences         *SortedSequences_t
}

var seqHandler *SeqHandler_t = nil

func GetSeqHandler() {
    return seqHandler
}


func InitSeqHandler() {
    seqHandler = &SeqHandler_t { make(ActiveRequestsList_t), make(Sequences_t),
                make(Sequences_t), make(SortedSequences_t) }
}


/*
 * Called upon action registration by client or upon sequence completion
 *
 * Raise request, if this is first action in any configured sequence
 */
func (p *SeqHandler_t) RaiseRequest(action string) error {
    regF = GetRegistrations()

    /* Is action registered */
    if  regF.GetActiveActionInfo(action) == nil {
        return LogError("Internal: Failing to get active action info (%s)", action)
    }

    /* Is any active request for this action */
    if _, ok := p.activeRequests[action]; ok {
        return LogError("Internal: An active request exists (%s)", action)
    }

    cfg := GetConfigMgr()
    /* Is this action start of a sequence */
    if !cfg.IsStartSequenceAction(action) {
        /* Not an initial action of any sequence */
        return nil
    }

    /* Is there an active/pending sequence for this action */
    if s, ok := p.sequencesByFirstAction[action]; ok {
        return LogError(
                "Internal: An active sequence is in progress ID(%s) index(%d) due(%v)seconds",
                s.AnomalyInstanceId, s.CtIndex, time.Now.Unix() - int64(p.SeqExpEpoch))
    }

    /* All clear. Fire request for the first action of a sequence */
    uuid := GetUUID()
    /* Make request. Add to client & active */
    req := &ServerRequestData {
            TypeServerRequestAction,
            &ActionRequestData  {
                Action: action,
                InstanceId: uuid,
                AnomalyInstanceId: uuid,
                Timeout: 0                      /* No timeout for first action in sequence */
            }
        }
    }

    /* Add to client's pending Q  to send to client upon client asking for a request. */
    /* Stays in this Q until client reads it */
    if err := regF.AddServerRequest(action, req) = nil {
        LogPanic("Internal error: Failed to AddServerRequest (%s)", action)
    }

    /* Track it in our active requests;  Waits here till response */
    p.activeRequests[action] = &ActiveRequest_t {req, uuid, 0}

}


/* Called upon action de-registration */
func (p *SeqHandler_t) DropRequest(action string) {
    if r, ok := p.activeRequests[action]; !ok {
        return
    } else {
        delete (p.activeRequests, action)
        if s, ok := p.sequencesByAnomaly[r.anomalyID]; ok {
            s.Context = append(s.Context, &ActionResponseData {
                Action: action,
                AnomalyInstanceId: r.anomalyID,
                ResultCode: -1,         /* TODO: get list of codes */
                ResultStr: "Dropping request due to de-registration"
            })
            p.ProcessSequence(s)
        }
    }
}


func (p *SeqHandler_t) PublisgResponse(res *ActionResponseData, 
            seq *SequenceState_t) {
                TODO
    Publish this response
    Add state if it is first action
    re-publish first action on sequence complete
}


func (p *SeqHandler_t) Processresponse(msg *MsgSendServerResponse) {
    /* TODO: Honor BindingActionCfg_t:Mandatory */

    seq := (*SequenceState_t)(nil)
    anomalyID := ""
    anomalyAction := ""

    if msg.ReqType != TypeServerRequestAction {
        LogError("Unexpected response req type (%d)/(%s)",
                msg.ReqType, ServerReqTypeToStr(msg.ReqType))
        return
    }
    p := msg.ResData
    data, ok := p.(*ActionResponseData)
    if !ok {
        LogError("Unexpected response res data (%T)/(%T)", data, p)
        return
    }
    anomalyID = data.AnomalyInstanceId

    seq, ok := p.sequencesByAnomaly[anomalyID]
    if !ok {
        seq = nil
        if anomalyID == data.InstanceId {
            anomalyAction = data.Action
        }
    } else {
        anomalyAction = seq.Actions[0].Name
        seq.CurrentRequest = nil
    }

    /* Drop from active requests */
    delete (p.activeRequests, data.Action)

    /* Publish received response; Add state=init, if first action */
    PublishResponse(data.ToMap(false))

    defer func() {
        /* If sequence is complete / aborted, re-publish anomaly */
        if len(anomalyID) == 0 {
            /* Invalid response. Nothing todo */
            return
        }

        anomalyResp := data
        if seq != nil {
            if seq.CurrentRequest != nil {
                /* Sequence is in progress. No re-publish yet */
                return
            }

            anomalyResp = seq.Context[0]
            if len(seq.Context) > 1 {
                /* Copy response code from the last response */
                lastResp := seq.Context[len(seq.Context)-1]
                anomalyResp.ResultCode = lastResp.ResultCode
                anomalyResp.ResultStr = lastResp.ResultStr
            }
        }
        PublishResponse(anomalyResp.ToMap(true)
        p.RaiseRequest(anomalyResp.Action)
    }()

    if !ok {
        /*
         * No existing sequence found.
         * Two possibilities
         * 1. Response from first action and sequence to be built.
         * 2. Stale response for an aborted sequence
         */
        if data.ResultCode != 0 {
            /*
             * No point in creating sequence. Call Raise request, which will succeed
             * if first action in sequence.
             *
             * NOTE: An action may go crazy returning too frequently. This is guarded
             * as below.
             * 1. A plugin's conf has min interval between two callbacks for same anomaly key.
             * 2. PluginMgr has a moving window of last 100 instances for a key of action+anomalyKey
             *    If that crosses an average of N seconds between 2, it disables the action and
             *    de-register it.
             *
             * If first action, re-publish is not done, as detection has failed.
             */
            return
        }
        seq = &SequenceState_t { AnomalyInstanceId: anomalyID, CtIndex: 1 }
        seq.SeqStartEpoch = time.Now().Unix()
        bs, err := GetConfigMgr().GetSequence(data.Action)
        if err != nil {
            /* No sequence for this action. Likely stale non-first action or config changed */
            bs = nil
            seq.Context = make([]ActionResponseData, 0, 1)
        } else {
            seq.SeqExpEpoch = seq.SeqStartEpoch + bs.Timeout
            seq.sequence = bs
            seq.Context = make([]ActionResponseData, 0, len(bs.Actions))
        }
        seq.Context = append(seq.Context, data)

        if bs == nil {
            /* No seq; Bail out. Defer will re-publish anomaly as needed */
            return
        }

        p.sequencesByAnomaly[anomalyID] = seq
        p.sequencesByFirstAction[data.Action] = seq
    } else {
        seq.Context = append(seq.Context, data)
        seq.CtIndex++
    }

    /* Process the sequence */
    if data.ResultCode != 0 {
        /* Need to abort the sequence */
        return
    }

    if seq.CtIndex >= len(seq.sequence.Actions) {
        /* WooHoo we are really complete 
        return
    }

    nextAction = seq.Actions[seq.CtIndex]

    /* Is any active request for this action */
    resultCode := 0
    if regF.GetActiveActionInfo(action) == nil {
        resultCode = -1
        ResultStr = "Action is no more registered"
    } else if _, ok := p.activeRequests[nextAction.Name]; ok {
        resultCode = -1
        ResultStr = "Action is still pending last request raised. Hence skipped"
    }

    if resultCode != 0 {
        LogError("(%s): (%s) action in seq(%s)", ResultStr, nextAction.Name,
                    seq.sequence.SequenceName)
        seq.Context = append(seq.Context, &ActionResponseData {
                        Action: nextAction.Name,
                        AnomalyInstanceId: anomalyID,
                        ResultCode: resultCode,  
                        ResultStr: ResultStr,
                    })
        return
    }

    /* All clear. Fire request for the next action in sequence */
    req := &ServerRequestData {
        TypeServerRequestAction,
        &ActionRequestData  {
            Action: nextAction.Name,
            InstanceId: GetUUID(),
            AnomalyInstanceId: anomalyID,
            AnomalyKey: seq.Context[0].AnomalyKey,
            Timeout : nextAction.Timeout,
            Context: seq.Context
        }
    }

    /* Add to client's pending Q  to send to client upon client asking for a request. */
    /* Stays in this Q until client reads it */
    if err := regF.AddServerRequest(action, req) = nil {
        LogPanic("Internal error: Failed to AddServerRequest (%s)", action)
    }

    /* Track it in our active requests;  Waits here till response */
    p.activeRequests[action] = &ActiveRequest_t {req, uuid, 0}

    seq.CurrentRequest = &ActiveRequest_t {
        req: req,
        anomalyID:anomalyID,
        ReqExpEpoch: time.Now().Unix() + nextAction.Timeout 
    }

}


