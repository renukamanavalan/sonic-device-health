package main

import (
	"go/src/lib/lomcommon"
	"go/src/pluginmgr/pluginmgr_common"
	"log/syslog"
)

func main() {
	lomcommon.SetLogLevel(syslog.LOG_NOTICE)
	lomcommon.LogMessage(syslog.LOG_NOTICE, "plugin_mgr : Starting Plugin Manager")

	if err := pluginmgr_common.SetupPluginManager(); err != nil {
		lomcommon.LogMessage(syslog.LOG_ERR, "plugin_mgr : SetupPluginManager failed")
		return
	}
	if err := pluginmgr_common.StartPluginManager(); err != nil {
		lomcommon.LogMessage(syslog.LOG_ERR, "plugin_mgr : StartPluginManager failed")
		return
	}

	lomcommon.LogMessage(syslog.LOG_NOTICE, "plugin_mgr : Exiting plugin manager")
}
