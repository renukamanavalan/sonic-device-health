package plugins_common

import (
	"go/src/lib/lomipc"
	"sync"
	"time"
)

// Plugin interface is implemented by all plugins and is used by plugin manager to call plugin methods
type Plugin interface {
	Request(hbchan chan PluginHeartBeat, request *lomipc.ActionRequestData) *lomipc.ActionResponseData
	Init(plugindata PluginData) error
	Shutdown() error
	GetPluginID() PluginId

	CommonPluginMethods
}

// CommonPluginMethods has common methods that are used by plugin manager to manage plugins. Methods remain same for all plugins
type CommonPluginMethods interface {
	GetMetadata() PluginMetadata
	SetMetadata(metadata PluginMetadata)
	GetPluginStage() PluginStage
	SetPluginStage(stage PluginStage)
}

// PluginStage indicates the current stage of plugin. Based on  this value plugin manager decisions. For e.g.  whether to accept requests from engine or not
type PluginStage int

const (
	PluginStageUnknown PluginStage = iota // default value
	PluginStageLoadingStarted
	PluginStageLoadingError
	PluginStageLoadingSuccess
	PluginStageInitStarted
	PluginStageInitFailure
	PluginStageInitSuccess
	PluginStageServerRegistrationStarted
	PluginStageServerRegistrationSuccess // only accepts requests from engine at this stage
	PluginStageServerRegistrationFailed
	PluginStageServerDeRegistrationStarted
	PluginStageServerDeRegistrationSuccess
	PluginStageServerDeRegistrationFailed
	PluginStageRequestStarted
	PluginStageRequestStartedHB // for long running
	PluginStageRequestError
	PluginStageRequestSuccess
	PluginStageRequestRunning
	PluginStageShutdownStarted
	PluginStageShutdownSuccess
	PluginStageShutdownError
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

// Holds all data specific to plugin, run time info, etc
type PluginMetadata struct {
	Plugindata  PluginData
	StartedTime time.Time
	Pluginstage PluginStage // indicate the current plugin stage
	PluginId
	sync.Mutex
	// ... other common metadata fields
}
