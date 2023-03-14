package engine

import (
    "flag"
    . "lib/lomcommon"
    . "lib/lomipc"
    "os"
    "os/signal"
    "runtime"
    "syscall"
)


const (
    LoMResponseOk = iota,
    LoMUnknownReqType,
    LoMIncorrectReqData,
    LoMReqFailed,
    LoMReqTimeout,
    LoMErrorEnd
)

var LoMResponseStr = map[int]string {
    "",
    "Unknown request",
    "Incorrect Msg type",
    "Request failed",
    "Request Timed out"
    "END"
}

func GetLoMResponseStr(code int) string {
    if (code <LoMResponseOk) || (code >= LoMErrorEnd) {
        return "Unknown error code"
    }
    return LoMResponseStr[code]
}


/* Helper to construct LoMResponse object */
func createLomResponse(code int, msg string, data interface{})
{
    if (code <LoMResponseOk) || (code >= LoMErrorEnd) {
        LogPanic("Unexpected error code (%d) range (%d to %d)",
                code, LoMResponseOk, LoMErrorEnd)
    }
    s := msg
    if (len(s) == 0) && (code != LoMResponseOk) {
        /* Prefix caller name to provide context */
        if pc, _, _, ok := runtime.Caller(1); ok {
            details := runtime.FuncForPC(pc)
            s = details.Name() + ": " + LoMResponseStr[code]
        }
    }
    return &LoMResponse { code, s, data }
}


type serverHandler_t struct {
}

func (p *serverHandler_t) processRequest(req *LoMRequestInt) {
    vat res *LomResponse) := nil

    switch req.Req.ReqType {
    case TypeRegClient:
        res = p.registerClient(req.Req)
    case TypeDeregClient:
        res = p.deregisterClient(req.Req)
    case TypeRegAction:
        res = p.registerAction(req.Req)
    case TypeDeregAction:
        res = p.deregisterAction(req.Req)
    case TypeRecvServerRequest:
        res = p.recvServerRequest(req.Req)
    case TypeSendServerResponse:
        res = p.sendServerResponse(req.Req)
    case TypeNotifyActionHeartbeat:
        res = p.notifyHeartbeat(req.Req)
    default:
        res = createLomResponse(LoMUnknownReqType, "", nil)
    }
    req.ChResponse <- res
    if res.ResultCode == 0 {
        switch req.Req.ReqType {
        case TypeRegAction:
            m, _ := req.ReqData.(MsgRegAction)
            GetSeqHandler().RaiseRequest(m.Action)
        case TypeSendServerResponse:
            m, _ := req.ReqData.(MsgSendServerResponse)
            GetSeqHandler().ProcessResponse(m)
        }
    }
}


func (p *serverHandler_t) registerClient(req *LoMRequest) *LomResponse {
    if _, ok := req.ReqData.(MsgRegClient); !ok {
        return createLomResponse(LoMIncorrectReqData, "", nil)
    }
    e := GetRegistrations().RegisterClient(ClientName_t(req.Client))
    if e != nil {
        return createLomResponse(LoMReqFailed, fmt.Sprint(e), nil)
    }
    return createLomResponse(LoMResponseOk, "", nil)
}


func (p *serverHandler_t) deregisterClient(req *LoMRequest) *LomResponse {
    if _, ok := req.ReqData.(MsgDeregClient); !ok {
        return createLomResponse(LoMIncorrectReqData, "", nil)
    }
    GetRegistrations().DeregisterClient(ClientName_t(req.Client))
    return createLomResponse(LoMResponseOk, "", nil)
}


func (p *serverHandler_t) registerAction(req *LoMRequest) *LomResponse {
    if m, ok := req.ReqData.(MsgRegAction); !ok {
        return createLomResponse(LoMIncorrectReqData, "", nil)
    }
    info := &ActiveActionInfo_t { m.Action, ClientName_t(req.Client), 0 }
    e := GetRegistrations().RegisterAction(info)
    if e != nil {
        return createLomResponse(LoMReqFailed, fmt.Sprint(e), nil)
    }
    return createLomResponse(LoMResponseOk, "", nil)
}


func (p *serverHandler_t) deregisterAction(req *LoMRequest) *LomResponse {
    if m, ok := req.ReqData.(MsgDeregAction); !ok {
        return createLomResponse(LoMIncorrectReqData, "", nil)
    }
    GetRegistrations().DeregisterAction(m.Action)
    return createLomResponse(LoMResponseOk, "", nil)
}


func (p *serverHandler_t) notifyHeartbeat(req *LoMRequest) *LomResponse {
    if m, ok := req.ReqData.(MsgNotifyHeartbeat); !ok {
        return createLomResponse(LoMIncorrectReqData, "", nil)
    }
    GetRegistrations().NotifyHeartbeats(m.Action, m.Timestamp)
    return createLomResponse(LoMResponseOk, "", nil)
}


func (p *serverHandler_t) recvServerRequest(req *LoMRequest) *LomResponse {
    if m, ok := req.ReqData.(MsgRecvServerRequest); !ok {
        return createLomResponse(LoMIncorrectReqData, "", nil)
    }
    if err := GetRegistrations().SendServerRequest(req); err == nil {
        /* ClientRegistrations_t will send the request whenever available */
        return nil
    } else {
        return createLomResponse(LoMReqFailed, fmt.Sprintf("%v", err), nil)
    }
}


func (p *serverHandler_t) sendServerResponse(req *LoMRequest) *LomResponse {
    if m, ok := req.ReqData.(MsgSendServerResponse); !ok {
        return createLomResponse(LoMIncorrectReqData, "", nil)
    }
    return createLomResponse(LoMResponseOk, "", sreq)
    /* Process response called in caller after sending response back */
}


func GetServerReqHandler() *serverHandler_t {
    return &serverHandler_t{}
}

