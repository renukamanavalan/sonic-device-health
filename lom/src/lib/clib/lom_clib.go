package main

import (
    "C"
    "encoding/json"
    "fmt"
    . "lom/src/lib/lomcommon"
    . "lom/src/lib/lomipc"
)


/*
 * C-bindings for non-go clients.
 *
 * A simple wrapper to corresponding APIs
 */

/*
 * ----------------------------------------------------------------
 * Config get APIs
 * ----------------------------------------------------------------
 */

//export InitConfigPathForC
func InitConfigPathForC(pathPtr *C.char) int {
    path := C.GoString(pathPtr)
    if err := InitConfigPath(path); err != nil {
        LogError("Failed to init config for path(%s) err(%v)", path, err)
        return -1
    }
    return 0
}

//export GetGlobalCfgStr
func GetGlobalCfgStr(keyPtr *C.char) *C.char {
    key := C.GoString(keyPtr)
    ret := fmt.Sprintf("%v", GetConfigMgr().GetGlobalCfgAny(key))
    return C.CString(ret)
}

//export GetGlobalCfgInt
func GetGlobalCfgInt(keyPtr *C.char) C.int {
    key := C.GoString(keyPtr)
    val := GetConfigMgr().GetGlobalCfgAny(key)
    iVal, ok := val.(int) 
    if !ok {
        LogError("Missing key with int val (%T)/(%v)", val, val)
    }
    return C.int(iVal)     /* Defaults to 0 on failed conversion */
}

//export GetSequenceAsJson
func GetSequenceAsJson(namePtr *C.char) *C.char {
    name:= C.GoString(namePtr)
    ret := ""
    if v, err := GetConfigMgr().GetSequence(name); err != nil {
        LogError("Failed to find sequence for (%s) (%v)", name, err)
    } else if out, err := json.Marshal(v); err != nil {
        LogError("Failed to marshal sequence (%v) err(%v)", v, err)
    } else {
        ret = string(out)
    }
    return C.CString(ret)
}

//export GetActionConfigAsJson
func GetActionConfigAsJson(namePtr *C.char) *C.char {
    name:= C.GoString(namePtr)
    ret := ""
    if v, err := GetConfigMgr().GetActionConfig(name); err != nil {
        LogError("Failed to get conf for action (%s) (%v)", name, err)
    } else if out, err := json.Marshal(v); err != nil {
        LogError("Failed to marshal action config (%v) err(%v)", v, err)
    } else {
        ret = string(out)
    }
    return C.CString(ret)
}

//export GetActionsListAsJson
func GetActionsListAsJson() *C.char {
    ret := ""
    v := GetConfigMgr().GetActionsList()

    if out, err := json.Marshal(v); err != nil {
        LogError("Failed to marshal action list (%v) err(%v)", v, err)
    } else {
        ret = string(out)
    }
    return C.CString(ret)
}

//export GetProcsConfig
func GetProcsConfig(namePtr *C.char) *C.char {
    name:= C.GoString(namePtr)
    ret := ""

    if v, err := GetConfigMgr().GetProcConfig(name); err != nil {
        LogError("Failed to get proc conf for (%s) err=(%v)", name, err)
    } else if out, err := json.Marshal(v); err != nil {
        LogError("Failed to marshal proc config (%v) err(%v)", v, err)
    } else {
        ret = string(out)
    }
    return C.CString(ret)
}


/*
 * ----------------------------------------------------------------
 * Engine client side APIs
 *
 * All APIs return JSON string as o/p
 *
 *  {
 *      "retCode":  <int>       // return value. 0 implies success
 *      "retStr":   <string>    // Human readable string
 *      "response": <string>    // JSONified o/p of the API, if any. 
 *                              // Response expected only for receive server req
 *  }
 *
 * ----------------------------------------------------------------
 */

var clientSessTx *ClientTx

type RetResponse struct {
    ResultCode  int
    ResultStr   string
    RespData    interface{}
}

var UnkRetStr = `{"ResultCode":-1,"ResultStr":"Unknown error","RespData":null}`

func (p *RetResponse) String() string {
    if out, err := json.Marshal(p); err != nil {
        LogError("Internal Code Error in JSON Marshal (%v) (%v)", err, *p)
        return UnkRetStr
    } else {
        return string(out)
    }
}

func GetRetResponseWithData(err error, respData interface{}) string {
    if err != nil {
        return (&RetResponse{-1, fmt.Sprintf("%v", err), respData}).String()
    } else {
        return (&RetResponse{0, "", respData}).String()
    }
}

func GetRetResponse(err error) string {
    return GetRetResponseWithData(err, nil)
}


//export RegisterClientC
func RegisterClientC(namePtr *C.char) *C.char {
    name:= C.GoString(namePtr)

    clientSessTx = GetClientTx(0)
    ret := GetRetResponse(clientSessTx.RegisterClient(name))
    return C.CString(ret)
}

//export DeregisterClientC
func DeregisterClientC() *C.char {
    return C.CString(GetRetResponse(clientSessTx.DeregisterClient()))
}

//export RegisterActionC
func RegisterActionC(namePtr *C.char) *C.char {
    name:= C.GoString(namePtr)

    return C.CString(GetRetResponse(clientSessTx.RegisterAction(name)))
}

//export DeregisterActionC
func DeregisterActionC(namePtr *C.char) *C.char {
    name:= C.GoString(namePtr)

    return C.CString(GetRetResponse(clientSessTx.DeregisterAction(name)))
}

//export RecvServerRequestC
func RecvServerRequestC() *C.char {
    req, err := clientSessTx.RecvServerRequest()
    return C.CString(GetRetResponseWithData(err, req))
}

//export SendServerResponseC
func SendServerResponseC(respPtr *C.char) *C.char {
    respStr := C.GoString(respPtr)
    var err error
    bData := []uint8{}

    resData := &MsgSendServerResponse{}
    if err = json.Unmarshal([]byte(respStr), resData); err == nil {
        if resData.ReqType == TypeServerRequestAction {
            if bData, err = json.Marshal(resData.ResData); err == nil {
                rd := &ActionResponseData{}
                if err = json.Unmarshal(bData, rd); err == nil {
                    resData.ResData = *rd
                }
            }
        }
    }
    ret := GetRetResponse(clientSessTx.SendServerResponse(resData))
    return C.CString(ret)
}

//export NotifyHeartbeatC
func NotifyHeartbeatC(namePtr *C.char, tstamp C.longlong) *C.char {
    name:= C.GoString(namePtr)

    return C.CString(GetRetResponse(clientSessTx.NotifyHeartbeat(name, int64(tstamp))))
}

func main() {
}

