/*
 * package plugins_files contains all plugins. Each plugin is a go file with a struct that implements Plugin interface.
 * Example Plugin for reference
 */

package plugins_files

import (
    "lom/src/lib/lomcommon"
    "lom/src/lib/lomipc"
    "lom/src/plugins/plugins_common"
    "time"
)

type GenericPluginDetection struct {
    // ... Internal plugin data
}

func NewGenericPluginDetection(...interface{}) plugins_common.Plugin {
    // ... create and return a new instance of MyPlugin
    return &GenericPluginDetection{
        // ... initialize internal plugin data
    }
}

func init() {
    // ... register the plugin with plugin manager
    plugins_common.RegisterPlugin("GenericPluginDetection", NewGenericPluginDetection)
}

func (gpl *GenericPluginDetection) Init(actionCfg *lomcommon.ActionCfg_t) error {
    // ... implementation

    time.Sleep(2 * time.Second)
    return nil
}

func (gpl *GenericPluginDetection) Request(hbchan chan plugins_common.PluginHeartBeat, request *lomipc.ActionRequestData) *lomipc.ActionResponseData {
    // ... implementation

    // return data from request
    return &lomipc.ActionResponseData{
        Action:            request.Action,
        InstanceId:        request.InstanceId,
        AnomalyInstanceId: request.AnomalyInstanceId,
        AnomalyKey:        "Ethernet10",
        Response:          "Ethernet10 is down",
        ResultCode:        0,         // or non zero
        ResultStr:         "Success", // or "Failure"
    }
}

func (gpl *GenericPluginDetection) Shutdown() error {
    // ... implementation

    time.Sleep(2 * time.Second)
    return nil
}

func (gpl *GenericPluginDetection) GetPluginID() plugins_common.PluginId {
    return plugins_common.PluginId{
        Name:    "GenericPluginDetection",
        Version: "1.0",
    }
}
