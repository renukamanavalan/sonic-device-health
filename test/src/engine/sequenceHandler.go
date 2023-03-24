package engine

import (
    "encoding/json"
    . "lib/lomcommon"
    . "lib/lomipc"
)

/*
 * Active request - Requests sent out awaiting response with or w/o timeout
 */
type ActiveRequest_t struct {
    req         *ServerRequestData
    anomalyID   string
    ReqExpEpoch EpochSecs   /* Expiry time. A value of 0 means no expiry */
}

/*
 * Action vs request -- All active requests
 *
 * NOTE: Only one outstanding request per Action
 * Hence can be keyed off by action.
 */
type ActiveRequestsList_t map[string]*ActiveRequest_t


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
    CtIndex                 int             /* Index of action in sequence in-progress */
    Context                 []*ActionResponseData /* Ordered responses as received, so far */
    CurrentRequest          *ActiveRequest_t /* Current request in progress */
}

func (p *SequenceState_t) ExpiryEpoch() EpochSecs {
    /* If current request expire early, send it; else send seq expiry */
    if p.CurrentRequest != nil {
        if p.CurrentRequest.ReqExpEpoch < p.SeqExpEpoch {
            return p.CurrentRequest.ReqExpEpoch
    }
    return p.SeqExpEpoch
}


/*
 * map[anomaly id]sequence_state_t
 *
 * Sequences are keyed off of anomaly instance Id 
 */
type Sequences_t map[string]*SequenceState_t

/*
 * timeout:
 * Two possible candidates:
 *      1. Currently active request for currently active sequence
 *      2. All pending sequences, who may timeout before getting chance to execute sequence.
 */
type SortedSequences_t []SequenceState_t


type SeqHandler_t struct {
    /* All Active requests by action name */
    activeRequests          *ActiveRequestsList_t

    /* Collected sequences by anomaly */
    sequencesByAnomaly      *Sequences_t

    /* collected sequences by first action */
    sequencesByFirstAction  *Sequences_t

    sortedSequences         *SortedSequences_t

    chTimer                 chan int64  /* Channel to convey earliest timeout */
}

var seqHandler *SeqHandler_t = nil

func GetSeqHandler() {
    return seqHandler
}


func InitSeqHandler() {
    seqHandler = &SeqHandler_t { make(ActiveRequestsList_t), make(Sequences_t),
                make(Sequences_t), make(SortedSequences_t), make(chan int64) }
    go seqHandler.processTimeout()
}

func (p *SeqHandler_t) Close() {
    for _, v := p.activeRequests {
        p.AbortRequest(LoMShutdown)
    }
}


func (p *SeqHandler_t) processTimeout() {
    for {
        select {
        case = <- p.chTimer:
            // TODO here


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
            res := &ActionResponseData {
                Action: action,
                AnomalyInstanceId: r.anomalyID,
                ResultCode: LomActionDeregistered,         /* TODO: get list of codes */
                ResultStr: "Dropping request due to de-registration"
            }
            p.ProcessActionResponse(res)
        }
    }
}

func (p *SeqHandler_t) AddSequence(seq *SequenceState_t) {
    p.sequencesByAnomaly[seq.AnomalyInstanceId] = seq
    p.sequencesByFirstAction = seq.Context[0].Action
    p.sortedSequences = append(p.sortedSequences, seq)
    /* Caller will sort it */
}


func (p *SeqHandler_t) DropSequence(seq *SequenceState_t) {
    if seq != nil {
        delete (p.sequencesByAnomaly, seq.AnomalyInstanceId)
        delete (p.sequencesByFirstAction, seq.Context[0].Action)
        p.sortedSequences = make(Sequences_t, len(p.sequencesByAnomaly))
        i := 0
        for _, v := range(p.sequencesByAnomaly) {
            p.sortedSequences[i] = v
            i++
        }
        /* Caller will sort it */
    }
}


func (p *SeqHandler_t) PublishResponse(res *ActionResponseData, 
            seq *SequenceState_t) {
    PublishEvent(res.ToMap(seq == nil))
}


func (p *SeqHandler_t) ProcessResponse(msg *MsgSendServerResponse) {
    /* TODO: Honor BindingActionCfg_t:Mandatory */

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
    p.ProcessActionResponse(data)
}


func (p *SeqHandler_t) ProcessActionResponse(data *ActionResponseData) {
    anomalyAction := ""
    anomalyID = ""
    errCode := -1
    errStr := "Unknown error"

    /* Publish received response, even if stale/unexpected. */
    /* In any case it is result posted by plugin, hence publish/record */
    PublishResponse(data.ToMap(false))

    /* Drop from active requests */
    if r, ok := p.activeRequests[data.Action]; ok {
        if r.InstanceId != data.InstanceId {
            /* stale response */
            LogError("Active req vs res instance ID mismatch (%v) (%v)", r, data)
            return
        }
    } else {
        LogError("No matching req for res (%v)", data)
        return
    }

    /* Response matched active request */
    delete (p.activeRequests, data.Action)

    /* Get sequence */
    seq, ok := p.sequencesByAnomaly[anomalyID]
    if !ok {
        seq = nil
        if data.anomalyID == data.InstanceId {
            anomalyAction = data.Action
            /* Result from first action. Create sequence */
        } else {
            /* Stale response. Non-first action w/o sequence. */
            LogError("Stale response (%v)", data)
            return
        }
    } else {
        if data.anomalyID == data.InstanceId {
            /* Anomaly response creates sequence. Hence unexpected for existing seq */
            /* Duplicate post by plugin / pluginMgr ? */
            LogError("First action response for existing seq (%v) res(%v)", seq, data)
            return 
        }
        if (seq.CurrentRequest == nil) {
            /* Don't expect sequence with no active request */
            LogError("Internal error. Seq with no acive request (%v) res(%v)", seq, data)
        } else if data.InstanceId != seq.CurrentRequest.InstanceId {
            /* Some stale response. Drop it. */
            LogError("Response (%v) not for current req seq(%v)", data, seq)
            return
        } else {
            /* reset current request */
            seq.CurrentRequest = nil
        }
        anomalyAction = seq.Actions[0].Name
    }
    anomalyID = data.AnomalyInstanceId

    defer func() {
        /* If sequence is complete / aborted, re-publish anomaly */
        if len(anomalyID) == 0 {
            /* Invalid response. Nothing todo */
            return
        }

        anomalyResp := (*ActionResponseData)(nil)
        if seq != nil {
            if seq.CurrentRequest == nil {
                /* Sequence is NOT in progress. In other words complete; Re-publish Anomaly */
                anomalyResp = seq.Context[0]
            }
            /* Sequence is still active. */

        } else if anomalyID == data.InstanceId {
            /* This is anomaly/first action. Failed to create sequence. Re-publish. */
            anomalyResp = data
        }

        if anomalyResp != nil {
            anomalyResp.ResultCode = errCode
            anomalyResp.ResultStr = errStr
            PublishResponse(anomalyResp.ToMap(true)
            p.RaiseRequest(anomalyResp.Action)
            p.DropSequence(seq)
        }
        sort.Slice(p.sortedSequences, func(i, j int) bool {
            return p.sortedSequences[i].ExpiryEpoch() < p.sortedSequences[i].ExpiryEpoch()
        })()
    }()

    if seq == nil {
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
            errCode = LomFirstActionFailed
            errStr = "First action / Anomaly detection failed. Nothing more to do"
            return
        }
        bs, err := GetConfigMgr().GetSequence(data.Action)
        if err != nil {
            /* No sequence for this action. Likely stale non-first action or config changed */
            errCode = LomMissingSequence
            errStr = "No sequence found"
            return
        }
        tnow := time.Now().Unix()
        ctx := make([]ActionResponseData, len(bs.Actions))
        ctx[0] = data

        seq = &SequenceState_t {
            AnomalyInstanceId: anomalyID,
            sequence: bs,
            SeqStartEpoch = tnow,
            SeqExpEpoch: tnow + bs.Timeout,
            CtIndex: 1,
            Context: ctx
        }
        p.AddSequence(seq)

    } else {
        seq.Context[seq.CtIndex] = data
        seq.CtIndex++
    }

    /* Process the sequence */
    if data.ResultCode != 0 {
        /* Need to abort the sequence */
        errCode = data.ResultCode
        errStr = data.ResultStr
        return
    }

    if seq.CtIndex >= len(seq.sequence.Actions) {
        /* WooHoo we are really complete 
        errCode = 0
        errStr = "Sequence complete successfully"
        return
    }

    nextAction = seq.Actions[seq.CtIndex]

    /* validate action */
    if regF.GetActiveActionInfo(action) == nil {
        errCode = LoMActionNotRegistered
        errStr = fmt.Sprintf("%v", LogError("%s: %s not registered. Abort", seq.sequence.SequenceName, action))
        return
    } else if _, ok := p.activeRequests[nextAction.Name]; ok {
        errCode = LoMActionActive
        errStr := fmt.Sprintf("%v", LogError("%s: %s request active. Abort", seq.sequence.SequenceName, action))
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
            Context: seq.Context,
        }
    }

    /* Add to client's pending Q  to send to client upon client asking for a request. */
    /* Stays in this Q until client reads it */
    if err := regF.AddServerRequest(action, req) = nil {
        LogPanic("Internal error: Failed to AddServerRequest (%s)", action)
    }

    /* Track it in our active requests;  Waits here till response */
    act := &ActiveRequest_t {req, anomalyID, ReqExpEpoch: time.Now().Unix() + nextAction.Timeout}
    p.activeRequests[action] = act

    seq.CurrentRequest = act
}


}])
