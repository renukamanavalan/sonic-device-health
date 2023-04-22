package main

import (
    "C"
    "encoding/json"
    "fmt"
    . "lom/src/lib/lomcommon"
)


/*
 * C-bindings for non-go clients.
 *
 * A simple wrapper to corresponding APIs
 */

//export TestStr
func TestStr(pathPtr *C.char) {
    path := C.GoString(pathPtr)
    fmt.Printf("TestStr(%s) called\n", path)
}

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


func main() {
}

