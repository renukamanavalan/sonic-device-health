/*
 * package plugins_common contains common interfaces and structs that are used by all plugins
 * It is used by plugin manager to manage plugins
 */

package plugins_common

import (
    "lom/src/lib/lomcommon"
    "lom/src/lib/lomipc"
    "time"
)

// Plugin interface is implemented by all plugins and is used by plugin manager to call plugin methods
type Plugin interface {
    Init(actionCfg *lomcommon.ActionCfg_t) error
    Request(hbchan chan PluginHeartBeat, request *lomipc.ActionRequestData) *lomipc.ActionResponseData
    Shutdown() error
    GetPluginID() PluginId
}

// TODO: Goutham : Clean up unnecessary fields
// PluginStage indicates the current stage of plugin. Based on  this value plugin manager decisions. For e.g.  whether to accept requests from engine or not
type PluginStage int

const (
    PluginStageUnknown PluginStage = iota // default value
    PluginStageLoadingSuccess
    PluginStageRequestStarted
    PluginStageRequestStartedHB // for long running
    PluginStageRequestError
    PluginStageRequestSuccess
)

// sent from plugin to plugin manager via heartbeat channel
type PluginHeartBeat struct {
    PluginName string
    EpochTime  int64
}

// sent from plugin to plugin manager as a responce to getPluginId()
type PluginId struct {
    Name    string
    Version string
}

// IPluginMetadata has common methods that are used by plugin manager to manage plugins. Data remain same for all plugins
type IPluginMetadata interface {
    GetPluginStage() PluginStage
    SetPluginStage(stage PluginStage)
}

// Holds all data specific to plugin, run time info, etc
type PluginMetadata struct {
    ActionCfg   *lomcommon.ActionCfg_t
    StartedTime time.Time
    Pluginstage PluginStage // indicate the current plugin stage
    PluginId
    // ... other common metadata fields
}

func (gpl *PluginMetadata) GetPluginStage() PluginStage {
    return gpl.Pluginstage
}

func (gpl *PluginMetadata) SetPluginStage(stage PluginStage) {
    gpl.Pluginstage = stage
}

// TODO: Goutham : Add more common methods here
