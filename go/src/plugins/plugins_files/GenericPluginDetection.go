/*
package plugins_files contains all plugins. Each plugin is a go file with a struct that implements Plugin interface.

Example Plugin for reference
*/

package plugins_files

import (
	"go/src/lib/lomcommon"
	"go/src/lib/lomipc"
	"go/src/plugins/plugins_common"
)

type GenericPluginDetection struct {
	// ... Internal plugin data
}

func (gpl *GenericPluginDetection) Init(actionCfg *lomcommon.ActionCfg_t) error {
	// ... implementation
	return nil
}

func (gpl *GenericPluginDetection) Request(hbchan chan plugins_common.PluginHeartBeat, request *lomipc.ActionRequestData) *lomipc.ActionResponseData {
	// ... implementation
	return nil
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
