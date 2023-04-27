package plugins_common

import (
	"go/src/lib/lomipc"
	"time"
)

// Plugin interface is implemented by all plugins and is used by plugin manager to call plugin methods
type Plugin interface {
	Init(plugindata PluginData) error
	Request(hbchan chan PluginHeartBeat, request *lomipc.ActionRequestData) *lomipc.ActionResponseData
	Shutdown() error
	GetPluginID() PluginId
}

// TODO: Goutham : Clean up unnecessary fields
// PluginStage indicates the current stage of plugin. Based on  this value plugin manager decisions. For e.g.  whether to accept requests from engine or not
type PluginStage int

const (
	PluginStageUnknown PluginStage = iota // default value
	PluginStageLoadingStarted
	PluginStageLoadingError
	PluginStageLoadingSuccess
	PluginStageRequestStarted
	PluginStageRequestStartedHB // for long running
	PluginStageRequestError
	PluginStageRequestSuccess
	PluginStageRequestRunning
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

/* Plugin manager passed this data to init() */
type PluginData struct {
	Timeout      int
	HeartbeatInt int
	ActionKnobs  string
	// ... Additional fields
}

// IPluginMetadata has common methods that are used by plugin manager to manage plugins. Data remain same for all plugins
type IPluginMetadata interface {
	GetPluginStage() PluginStage
	SetPluginStage(stage PluginStage)
}

// Holds all data specific to plugin, run time info, etc
type PluginMetadata struct {
	Plugindata  PluginData
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
