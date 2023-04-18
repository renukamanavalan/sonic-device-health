// Example Plugin for reference

package plugins_files

import (
	"gocode/src/lib/lomipc"
	"gocode/src/plugins/plugins_common"
)

type GenericPluginDetection struct {
	plugins_common.PluginMetadata
	// ... other Internal plugin data
}

func (gpl *GenericPluginDetection) Request(hbchan chan plugins_common.PluginHeartBeat, request *lomipc.ActionRequestData) *lomipc.ActionResponseData {
	// ... implementation
	return nil
}

func (gpl *GenericPluginDetection) Init(pluginConfig plugins_common.PluginData) error {
	// ... implementation
	return nil
}

func (gpl *GenericPluginDetection) Shutdown() error {
	// ... implementation
	return nil
}

func (gpl *GenericPluginDetection) GetPluginID() plugins_common.PluginId {
	return gpl.PluginMetadata.PluginId
}

/********************* Plugin's Management Methods common across all plugins *****************/

func (gpl *GenericPluginDetection) GetMetadata() plugins_common.PluginMetadata {
	gpl.PluginMetadata.Lock()
	defer gpl.PluginMetadata.Unlock()
	return gpl.PluginMetadata
}

func (gpl *GenericPluginDetection) SetMetadata(metadata plugins_common.PluginMetadata) {
	gpl.PluginMetadata.Lock()
	defer gpl.PluginMetadata.Unlock()
	gpl.PluginMetadata = metadata
}

func (gpl *GenericPluginDetection) GetPluginStage() plugins_common.PluginStage {
	gpl.PluginMetadata.Lock()
	defer gpl.PluginMetadata.Unlock()
	return gpl.PluginMetadata.Pluginstage
}

func (gpl *GenericPluginDetection) SetPluginStage(stage plugins_common.PluginStage) {
	gpl.PluginMetadata.Lock()
	defer gpl.PluginMetadata.Unlock()
	gpl.PluginMetadata.Pluginstage = stage
}
