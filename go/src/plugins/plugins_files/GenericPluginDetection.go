// Example Plugin for reference

package plugins_files

import (
	"go/src/lib/lomipc"
	"go/src/plugins/plugins_common"
)

type GenericPluginDetection struct {
	// ... Internal plugin data
}

func (gpl *GenericPluginDetection) Init(pluginConfig plugins_common.PluginData) error {
	// ... implementation
	return nil
}

func (gpl *GenericPluginDetection) Request(hbchan chan plugins_common.PluginHeartBeat, request *lomipc.ActionRequestData) (*lomipc.ActionResponseData, error) {
	// ... implementation
	return nil,nil
}

func (gpl *GenericPluginDetection) Shutdown() error {
	// ... implementation
	return nil
}

func (gpl *GenericPluginDetection) GetPluginID() plugins_common.PluginId {
	return plugins_common.PluginId{
		Name:    "GenericPluginDetection",
		Version: "1.0",
	}
}
