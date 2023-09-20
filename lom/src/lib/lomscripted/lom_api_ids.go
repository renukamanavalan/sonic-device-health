package lomscripted

import (
    cmn "lom/src/lib/lomcommon"
)

type ApiId_t string

const (
    ApiIDGetPubChannel            ApiId_t = "GetPubChannel"
    ApiIDGetSubChannel                    = "GetSubChannel"
    ApiIDRunPubSubProxy                   = "RunPubSubProxy"
    ApiIDSendClientRequest                = "SendClientRequest"
    ApiIDRegisterServerReqHandler         = "RegisterServerReqHandler"
    ApiIDDoSysShutdown                    = "DoSysShutdown"
    ApiIDWriteChannel                     = "WriteChannel"
    ApiIDReadChannel                      = "ReadChannel"
    ApiIDCloseChannel                     = "CloseChannel"
    ApiIDPause                            = "pause"
)

type ApiFn_t func(args []any) []any

var LomAPIByIds = map[ApiId_t]ApiFn_t {
    ApiIDGetPubChannel:                 callGetPubChannel,
    ApiIDGetSubChannel:                 callGetSubChannel,
    ApiIDRunPubSubProxy:                callRunPubSubProxy,
    ApiIDSendClientRequest:             callSendClientRequest,
    ApiIDRegisterServerReqHandler:      callRegisterServerReqHandler,
    ApiIDDoSysShutdown:                 callDoSysShutdown,
    ApiIDWriteChannel:                  callWriteChannel,
    ApiIDReadChannel:                   callReadChannel,
    ApiIDCloseChannel:                  callCloseChannel,
    ApiIDPause:                         callPause,
}

/*
 * Cache service for caching values for a call 
 * This could help help multiple APIs share their data
 */
 
type SuiteCache_t map[string]any

const ANONYMOUS = ""

/* A function to get i/p val */
type GetValFn_t func(name string, val any) (any, error)


func (s SuiteCache_t) Clear() {
    s = SuiteCache_t{}
}

func (s SuiteCache_t) GetVal(name string, val any, getFn GetValFn_t) (vRet any) {
    vRet = val
    if getFn != nil {
        var err error
        if vRet, err = getFn(name, val); err != nil {
            cmn.LogError("Failed to getVal from getFn name(%s) val(%v) err(%v)",
                    name, val, err)
            return
        }
    }
    if name == ANONYMOUS {
        /* Don't update cache */
    } else if val != nil {
        s.SetVal(name, val) /* overwrite */
    } else if ct, ok := s[name]; ok {
        vRet = ct
    }
    return
}

func (s SuiteCache_t) SetVal(name string, val any) {
    if name != ANONYMOUS {
        if val != nil {
            s[name] = val /* Set it */
        } else if _, ok := s[name]; ok {
            delete(s, name)
        }
    }
}

type Param_t struct {
    Name    string      /* Assign name to this var */
    Val     any         /* Val of this var */
    GetFn   GetValFn_t  /* Fn to get param val */
    /* If nil expect this var to pre-exist in cache. */
}

