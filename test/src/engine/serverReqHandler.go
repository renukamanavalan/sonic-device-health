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
    LoMErrorEnd
)

var LoMResponseStr = map[int]string {
    "",
    "Unknown request",
    "Incorrect Msg type",
    "Request failed",
    "END"
}

func GetLoMResponseStr(code int) string {
    if (code <LoMResponseOk) || (code >= LoMErrorEnd) {
        return "Unknown error code"
    }
    return LoMResponseStr[code]
}


func getLomResponse(code int, msg string, data interface{})
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
    switch req.Req.ReqType {
    case TypeRegClient:
        req.ChResponse <- p.registerClient(req.Req)
    case TypeDeregClient:
        req.ChResponse <- p.deregisterClient(req.Req)
    case TypeRegAction:
        req.ChResponse <- p.registerAction(req.Req)
    case TypeDeregAction:
        req.ChResponse <- p.deregisterAction(req.Req)
    case TypeRecvServerRequest:
        req.ChResponse <- p.recvServerRequest(req.Req)
    case TypeSendServerResponse:
        req.ChResponse <- p.sendServerResponse(req.Req)
    case TypeNotifyActionHeartbeat:
        req.ChResponse <- p.notifyHeartbeat(req.Req)
    default:
        req.ChResponse <- getLomResponse(LoMUnknownReqType, "", nil)
    }
}


func (p *serverHandler_t) registerClient(req *LoMRequest) *LomResponse {
    if _, ok := req.ReqData.(MsgRegClient); !ok {
        return getLomResponse(LoMIncorrectReqData, "", nil)
    }
    e := GetRegistrations().RegisterClient(ClientName_t(req.Client))
    if e != nil {
        return getLomResponse(LoMReqFailed, fmt.Sprint(e), nil)
    }
    return getLomResponse(LoMResponseOk, "", nil)
}


func (p *serverHandler_t) deregisterClient(req *LoMRequest) *LomResponse {
    if _, ok := req.ReqData.(MsgDeregClient); !ok {
        return getLomResponse(LoMIncorrectReqData, "", nil)
    }
    GetRegistrations().DeregisterClient(ClientName_t(req.Client))
    return getLomResponse(LoMResponseOk, "", nil)
}


func (p *serverHandler_t) registerAction(req *LoMRequest) *LomResponse {
    if m, ok := req.ReqData.(MsgRegAction); !ok {
        return getLomResponse(LoMIncorrectReqData, "", nil)
    }
    info := &ActiveActionInfo_t { ActionName_t(m.Action), ClientName_t(req.Client), 0 }
    e := GetRegistrations().RegisterAction(info)
    if e != nil {
        return getLomResponse(LoMReqFailed, fmt.Sprint(e), nil)
    }
    return getLomResponse(LoMResponseOk, "", nil)
}


func (p *serverHandler_t) deregisterAction(req *LoMRequest) *LomResponse {
    if m, ok := req.ReqData.(MsgDeregAction); !ok {
        return getLomResponse(LoMIncorrectReqData, "", nil)
    }
    GetRegistrations().DeregisterAction(ActionName_t(m.Action))
    return getLomResponse(LoMResponseOk, "", nil)
}


func (p *serverHandler_t) notifyHeartbeat(req *LoMRequest) *LomResponse {
    if m, ok := req.ReqData.(MsgNotifyHeartbeat); !ok {
        return getLomResponse(LoMIncorrectReqData, "", nil)
    }
    GetRegistrations().NotifyHeartbeats(ActionName_t(m.Action), m.Timestamp)
    return getLomResponse(LoMResponseOk, "", nil)
}


func (p *serverHandler_t) recvServerRequest(req *LoMRequest) *LomResponse {
    if m, ok := req.ReqData.(MsgNotifyHeartbeat); !ok {
        return getLomResponse(LoMIncorrectReqData, "", nil)
    }
    GetRegistrations().NotifyHeartbeats(ActionName_t(m.Action), m.Timestamp)
    return getLomResponse(LoMResponseOk, "", nil)
}


func GetServerReqHandler() *serverHandler_t {
    return &serverHandler_t{}
}

