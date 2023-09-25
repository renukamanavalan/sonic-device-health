package lomscripted

import (
    cmn "lom/src/lib/lomcommon"
    tele "lom/src/lib/lomtelemetry"
)

type ApiId_t string

/*
 * Cache service for caching values for a call
 * This could help help multiple APIs share their data
 */

type SuiteCache_t map[string]any

const (
    ApiIDGetPubChannel            ApiId_t = "GetPubChannel"
    ApiIDGetSubChannel                    = "GetSubChannel"
    ApiIDRunPubSubProxy                   = "RunPubSubProxy"
    ApiIDWriteJsonStringsChannel          = "WriteJsonStringsChannel"
    ApiIDReadJsonStringsChannel           = "ReadJsonStringsChannel"
    ApiIDSendClientRequest                = "SendClientRequest"
    ApiIDReadClientResponse               = "ReadClientResponse"
    ApiIDRegisterServerReqHandler         = "RegisterServerReqHandler"
    ApiIDReadClientRequest                = "ReadClientRequest"
    ApiIDSendClientResponse               = "SendClientResponse"
    ApiIDCloseRequestChannel              = "CloseRequestChannel"
    ApiIDCloseChannel                     = "CloseChannel"
    ApiIDPause                            = "pause"
    ApiIDIsTelemetryIdle                  = "IsTelemetryIdle"
)

type ApiFn_t func(args []any, cache SuiteCache_t) []any

var LomAPIByIds = map[ApiId_t]ApiFn_t{
    ApiIDGetPubChannel:            callGetPubChannel,
    ApiIDGetSubChannel:            callGetSubChannel,
    ApiIDRunPubSubProxy:           callRunPubSubProxy,
    ApiIDWriteJsonStringsChannel:  callWriteJsonStringsChannel,
    ApiIDReadJsonStringsChannel:   callReadJsonStringsChannel,
    ApiIDSendClientRequest:        callSendClientRequest,
    ApiIDReadClientResponse:       callReadClientResponse,
    ApiIDRegisterServerReqHandler: callRegisterServerReqHandler,
    ApiIDReadClientRequest:        callReadClientRequest,
    ApiIDSendClientResponse:       callSendClientResponse,
    ApiIDCloseRequestChannel:      callCloseRequestChannel,
    ApiIDCloseChannel:             callCloseChannel,
    ApiIDPause:                    callPause,
    ApiIDIsTelemetryIdle:          callIsTelemetryIdle,
}

const ANONYMOUS = ""

/* A function to get i/p val */
type GetValFn_t func(name string, val any) (any, error)

type StreamingDataEntity_t struct {
    Data []tele.JsonString_t /* Data to write. */
    More bool                /* true - more data to come. false - not any more */
}

/* Takes index & cache as args and return StreamingDataEntity_t & err */
type GetValStreamingFn_t func(int, SuiteCache_t) (*StreamingDataEntity_t, error)

/* Takes index, string & cache as args and return more as bool with error */
/* caller continues to write untul fn returns more=false */
type PutValStreamingFn_t func(int, tele.JsonString_t, SuiteCache_t) (bool, error)

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
        /* Neither read nor write into cache */
    } else if vRet != nil {
        switch vRet.(type) {
        case func(int, SuiteCache_t) (*StreamingDataEntity_t, error):
        case func(int, tele.JsonString_t, SuiteCache_t) (bool, error):
        default:
            /* Save only values not functions */
            s.SetVal(name, val) /* overwrite */
        }
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
    Name  string     /* Assign name to this var */
    Val   any        /* Val of this var */
    GetFn GetValFn_t /* Fn to get param val */
    /* If nil expect this var to pre-exist in cache. */
}
