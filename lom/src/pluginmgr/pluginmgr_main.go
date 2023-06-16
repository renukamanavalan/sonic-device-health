package main

import (
    "log/syslog"
    "lom/src/lib/lomcommon"
    "lom/src/pluginmgr/pluginmgr_common"

    "lom/src/plugins/plugins_files"
    "lom/src/plugins/plugins_files/sonic/plugin/linkcrc"
)

// TODO : Goutham : Temporary untill pernmant fix to include plugin files are found
var t1 = plugins_files.NewGenericPluginDetection
var t2 = linkcrc.NewLinkCRCDetectionPlugin

/*
* Main function for plugin manager
* This function does the following:
* 1. Setup logging
* 2. Setup plugin manager
* 3. Start plugin manager
* 4. Exit

* Expecting the following command line arguments:
*       proc_id - process id of the plugin manager
*               - Mandatory
*       syslog_level - syslog level for logging
*               - Optional (default: LOG_DEBUG)
*    Usage example:
*        ./LoMpluginMgr -proc_id="proc_0" -syslog_level=5
*
* Environment Variable :
*       LOM_TESTMODE_NAME - LoM run mode
*               - Optional (default: "no")
*       LOM_CONF_LOCATION - LoM configuration file location
*               - Mandatory
*
* Configuration Files : (relative to LOM_CONF_LOCATION)
*       actions.conf.json - Action plugins config file
*               - Mandatory
*       proc.conf.json - Proc Specific configuration file
*               - Mandatory
*       globals.conf.json - LOM system wide configuration file
*               - Mandatory
*
* Signals Handled :
*       SIGTERM - Stop plugin manager & exit gracefully
*
* Partner Application :
*       LoMEngine - LoM Engine running on the same device
*              - Mandatory in production mode.
*              - Optional in test mode.
* Logging :
*       Plugin Manager logs to syslog. The log level is set by the command line argument syslog_level.
*       The default log level is LOG_DEBUG.
*       Logs are wtitten to system log file . e.g. /var/log/syslog
*       The log format is as follows:
*       <timestamp> <hostname> <application name> <process id> <log level> <message>
 */
func main() {
    // setup logging
    lomcommon.SetLogLevel(syslog.LOG_INFO)

    lomcommon.LogInfo("plugin_mgr : Starting Plugin Manager")

    if err := pluginmgr_common.SetupPluginManager(); err != nil {
        lomcommon.LogPanic("plugin_mgr : SetupPluginManager failed") // exits
    }
    if err := pluginmgr_common.StartPluginManager(0); err != nil {
        lomcommon.LogPanic("plugin_mgr : StartPluginManager failed") // exits
    }

    lomcommon.LogInfo("plugin_mgr : Exiting plugin manager")
}
