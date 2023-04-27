package main

import (
	"go/src/lib/lomcommon"
	"go/src/pluginmgr/pluginmgr_common"
	"log/syslog"
)

func main() {
	// setup logging
	lomcommon.SetLogLevel(syslog.LOG_INFO)

	lomcommon.LogInfo("plugin_mgr : Starting Plugin Manager")

	if err := pluginmgr_common.SetupPluginManager(); err != nil {
		lomcommon.LogPanic("plugin_mgr : SetupPluginManager failed") // exits
	}
	if err := pluginmgr_common.StartPluginManager(); err != nil {
		lomcommon.LogPanic("plugin_mgr : StartPluginManager failed") // exits
	}

	lomcommon.LogInfo("plugin_mgr : Exiting plugin manager")
}
