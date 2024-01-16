package main

import (
    "lom/src/lib/lomcommon"
    "lom/src/pluginmgr/pluginmgr_common"
    //_ "lom/src/plugins/vendors/<<BUILD_TAGS>>/plugin/<<plugin_name>>"
)

/*
* Main function for plugin manager
* This function does the following:
* 1. Setup logging
* 2. Setup plugin manager
* 3. Start plugin manager
* 4. Exit
 */
func main() {
    lomcommon.LogInfo("plugin_mgr : Starting Plugin Manager")

    if err := pluginmgr_common.SetupPluginManager(); err != nil {
        lomcommon.LogPanic("plugin_mgr : SetupPluginManager failed") // exits
    }
    if err := pluginmgr_common.StartPluginManager(0); err != nil {
        lomcommon.LogPanic("plugin_mgr : StartPluginManager failed") // exits
    }

    lomcommon.LogInfo("plugin_mgr : Exiting plugin manager")
}
