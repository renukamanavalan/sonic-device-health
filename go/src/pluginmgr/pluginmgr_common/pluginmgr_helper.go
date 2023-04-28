package pluginmgr_common

import (
	"flag"
	"fmt"
	"go/src/lib/lomcommon"
	"go/src/lib/lomipc"
	"go/src/plugins/plugins_common"
	"go/src/plugins/plugins_files"
	"log/syslog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

/*****************************************************************************************/
/* Plugin manager */
/*****************************************************************************************/

// Plugin Manager global variables
var (
	ProcID    string         = ""
	pluginMgr *PluginManager = nil
)

// TODO: Goutham : Add this to global_conf.json
// Constants for plugin manager with default values
const (
	GOLIB_TIMEOUT_DEFAULT                         = 0 * time.Millisecond /* Default GoLIB API timeouts */
	PLUGIN_INIT_PERIODIC_FALLBACK_TIMEOUT_DEFAULT = 3600 * time.Second   /* Default Periodic logging long timeout */
	PLUGIN_INIT_PERIODIC_TIMEOUT_DEFAULT          = 300 * time.Second    /* Default periodic logging short timeout */
	PLUGIN_LOADING_TIMEOUT_DEFAULT                = 30 * time.Second
	APP_NAME_DEAULT                               = "plgMgr"
)

// Plugin Manager interface to be implemented by plugin manager
type IPluginManager interface {
	run(chan struct{}) error
	shutdown() error
	addPlugin(pluginName string, pluginVersion string) error
	registerActionWithEngine(pluginName string) error
	deRegisterActionWithEngine(pluginName string) error
	addPeriodicLogWithTimeouts(ID string, message string, shortTimeout time.Duration, longTimeout time.Duration, context string) chan bool
	loadPlugin(pluginName string, pluginVersion string) error
	getPlugin(plgname string) (plugins_common.Plugin, bool)
	getPluginMetadata(plgname string) (plugins_common.IPluginMetadata, bool)
}

// Interface  for lomipc.ClientTx
type IClientTx interface {
	RegisterClient(client string) error
	DeregisterClient() error
	RegisterAction(action string) error
	DeregisterAction(action string) error
	RecvServerRequest() (*lomipc.ServerRequestData, error)
	SendServerResponse(res *lomipc.MsgSendServerResponse) error
	NotifyHeartbeat(action string, tstamp int64) error
}

type PluginManager struct {
	clientTx IClientTx
	plugins  map[string]plugins_common.Plugin /* map : pluginname -> Plugin struct Object(at go/src/plugins/plugins_common)
	e.g. "linkflapdetection" -->  {linkFlapDetection object} */
	pluginMetadata map[string]plugins_common.IPluginMetadata /* map : pluginname -> PluginMetadata struct Object(at go/src/plugins/plugins_common) */
	stopch         chan struct{}                             /* channel used to stop listening on golib for server events */
	//TODO: Goutham : replace this with global stop channel
}

// InitPluginManager : Initialize plugin manager & register with server
func GetPluginManager(clientTx IClientTx) *PluginManager {
	if pluginMgr != nil {
		return pluginMgr
	}
	vpluginMgr := &PluginManager{
		clientTx:       clientTx,
		plugins:        make(map[string]plugins_common.Plugin),
		pluginMetadata: make(map[string]plugins_common.IPluginMetadata),
		stopch:         make(chan struct{}),
	}
	if err := vpluginMgr.clientTx.RegisterClient(ProcID); err != nil {
		panic("Error in registering client")
	}
	pluginMgr = vpluginMgr
	return pluginMgr
}

// getPluginManager : Get plugin manager object
func (plmgr *PluginManager) getPlugin(plgname string) (plugins_common.Plugin, bool) {
	plugin, ok := plmgr.plugins[plgname]
	return plugin, ok
}

// getPluginMetadata : Get plugin metadata object
func (plmgr *PluginManager) getPluginMetadata(plgname string) (plugins_common.IPluginMetadata, bool) {
	pluginmetadata, ok := plmgr.pluginMetadata[plgname]
	return pluginmetadata, ok
}

/*
 * Listens for server requests via golib. This call blocks.
 *
 * Input:
 *  stop - stop channel used to stop the blocking receive server call
 *
 * Output:
 *  none -
 *
 * Return:
 *  error - error message or nil on success
 */
func (plmgr *PluginManager) run() error {
	for {
		select {
		case <-plmgr.stopch:
			lomcommon.LogInfo("RecvServerRequest() : Stopping plugin manager run loop")
			return nil
		default:
			serverReq, err := plmgr.clientTx.RecvServerRequest()
			if err != nil {
				lomcommon.LogError("Error RecvServerRequest() : %v", err)
			} else if serverReq == nil {
				lomcommon.LogError("Error RecvServerRequest() : nil")
			} else {
				switch serverReq.ReqType {
				case lomipc.TypeServerRequestAction:
					actionReq, ok := serverReq.ReqData.(*lomipc.ActionRequestData)
					if !ok {
						lomcommon.LogError("RecvServerRequest() : Error in parsing ActionRequestData for type : %v, data : %v",
							serverReq.ReqType, serverReq.ReqData)
					} else {
						plugin, ok := plmgr.getPlugin(actionReq.Action)
						if !ok {
							lomcommon.LogError("RecvServerRequest() : Plugin %s not found", actionReq.Action)
						} else {
							lomcommon.LogInfo("In RecvServerRequest: Received action request for plugin %v", plugin)

							/* TODO: Goutham : Handle Request, Do error checks, pass HeartBEat channel, handle timeouts, handle heartbeats etc
							lomcommon.GetGoroutineTracker().Start("plg_mgr_Run_Action_"+actionReq.Action+"_"+lomcommon.GetUUID(),
								func() {
									hbchan := make(chan plugins_common.PluginHeartBeat)// TODO: Goutham ; Do we need buffered instead ??
									res := plugin.Request(hbchan, actionReq)
									plmgr.clientTx.SendServerResponse(&lomipc.MsgSendServerResponse{
										ReqType: lomipc.TypeServerRequestAction,
										ResData: res,
									})
								})
							*/
						}
					}
				case lomipc.TypeServerRequestShutdown:
					// TODO: Goutham :  handle shutdown(wait for timeout), haqndle synchronously, do not listen on responses from plugins, send deregister with server/
					// exit process, use goroutinrtracker to see if any routines are running and log to syslog
				default:
					lomcommon.LogError("RecvServerRequest() : Unknown server request type : %v", serverReq.ReqType)
				}
			}
		}
	}
	return nil
}

// TODO: Goutham : For all plugins call shutdoen, change plugin state, wait for a timeout, Log error on any running go routines per tracker, etc
func (plmgr *PluginManager) shutdown() error {
	if err := plmgr.clientTx.DeregisterClient(); err != nil {
		lomcommon.LogError("Error in deregistering client")
	}
	plmgr.stopch <- struct{}{}
	lomcommon.LogInfo("Plugin Manager shutdown")
	return nil
}

// TODO: Goutham : Dynamic plugins are not handled in V1 cut. Add this code later
/*
 * Calls plugin init() and f its successful calls registerAction() to engine. This call may block.
 *
 * Input:
 *  pluginname - plugin name
 *  pluginVersion - plugin version
 *
 * Output:
 *  none -
 *
 * Return:
 *  error - error message or nil on success
 */
func (plmgr *PluginManager) addPlugin(pluginName string, pluginVersion string) error {
	retMsg := ""

	defer func() {
		if retMsg != "" {
			plmgr.AddPeriodicLogWithTimeouts("Plgmgr_AddPlugin_"+lomcommon.GetUUID()+"_"+pluginName,
				retMsg, PLUGIN_INIT_PERIODIC_TIMEOUT_DEFAULT, PLUGIN_INIT_PERIODIC_FALLBACK_TIMEOUT_DEFAULT, "")
		}
	}()

	// 1.Check if plugin is already loaded
	if _, ok := plmgr.getPlugin(pluginName); ok {
		retMsg = fmt.Sprintf("plugin with name %s and version %s is already loaded", pluginName, pluginVersion)
		return lomcommon.LogError(retMsg)
	}

	// 2.Get plugin specific details from actions config file and add any additional info(future) to pass to plugin's init() call
	actionCfg, err := lomcommon.GetConfigMgr().GetActionConfig(pluginName)
	if err != nil {
		retMsg = fmt.Sprintf("plugin %s not found in actions config file", pluginName)
		return lomcommon.LogError(retMsg)
	}

	// 3.Check if plugin disabled flag is set or not in the actions config file.
	if actionCfg.Disable {
		lomcommon.LogWarning("Plugin %s is disabled", pluginName)
		return nil
	}

	// 4.Create new plugin instance
	pluginID := plugins_common.PluginId{Name: pluginName, Version: pluginVersion}
	plugin, pluginmetadata, err := CreatePluginInstance(pluginID, actionCfg) // returns Plugin interface pointing to new plugin struct
	if err != nil {
		retMsg = fmt.Sprintf("Error creating plugin instance for %s %s: %s", pluginName, pluginVersion, err)
		return lomcommon.LogError(retMsg)
	}

	// 5.Check if plugin name and version from proc_conf.json file matches the values in static plugin. If not log periodic log
	if id := plugin.GetPluginID(); id.Name != pluginName || id.Version != pluginVersion {
		retMsg = fmt.Sprintf("Plugin ID does not match provided arguments: got %s %s, expected %s %s", id.Name, id.Version, pluginName, pluginVersion)
		return lomcommon.LogError(retMsg)
	}

	// 6. call plugin's init() call synchronously
	err = plugin.Init(actionCfg)
	if err != nil {
		//delete(plmgr.plugins, pluginName) // TODO: Goutham : verify Is this needed instead?
		//delete(plmgr.pluginMetadata, pluginName) // TODO: Goutham : verify Is this needed instead.
		retMsg = fmt.Sprintf("plugin %s init failed: %v", pluginName, err)
		return lomcommon.LogError(retMsg)
	}

	// 7. call plugin's registerAction() call synchronously
	err = plmgr.registerActionWithEngine(pluginName)
	if err != nil {
		//delete(plmgr.plugins, pluginName) // TODO: Goutham : verify Is this needed instead?
		//delete(plmgr.pluginMetadata, pluginName) // TODO: Goutham : verify Is this needed instead.
		retMsg = fmt.Sprintf("plugin %s registerAction failed: %v", pluginName, err)
		return lomcommon.LogError(retMsg)
	}

	// 8.Add plugin to plugin manager's map
	plmgr.plugins[pluginName] = plugin
	plmgr.pluginMetadata[pluginName] = pluginmetadata

	return nil
}

func (plmgr *PluginManager) loadPlugin(pluginName string, pluginVersion string) error {
	// Create a channel to receive the result of AddPlugin()
	resultChan := make(chan error)

	lomcommon.GetGoroutineTracker().Start("plg_mgr_LoadPlugin"+pluginName, func() {
		resultChan <- plmgr.addPlugin(pluginName, pluginVersion)
	})

	// Wait for either the result or the timeout
	select {
	case err := <-resultChan:
		// AddPlugin completed within the timeout
		return err
	case <-time.After(PLUGIN_LOADING_TIMEOUT_DEFAULT):
		// AddPlugin timed out
		lomcommon.LogPanic("Registering plugin took too long pluginname : %s version : %s\n", pluginName, pluginVersion) // exits program
	}
	return nil
}

// RegisterActionWithEngine : Register plugin with engine
func (plmgr *PluginManager) registerActionWithEngine(pluginName string) error {
	if err := plmgr.clientTx.RegisterAction(pluginName); err != nil {
		//delete(plmgr.plugins, pluginName) // TODO: Goutham : verify Is this needed instead. Can't we store failed plugins ?
		//delete(plmgr.pluginMetadata, pluginName) // TODO: Goutham : verify Is this needed instead. Can't we store failed plugins ?
		return lomcommon.LogError("Failed to register plugin %s with engine", pluginName)
	}

	return nil
}

// DeRegisterActionWithEngine : DeRegister plugin with engine
func (plmgr *PluginManager) deRegisterActionWithEngine(pluginName string) error {
	if err := plmgr.clientTx.DeregisterAction(pluginName); err != nil {
		//delete(plmgr.plugins, pluginName) // TODO: Goutham : , Is this needed instead. Can't we store failed plugins ?
		//delete(plmgr.pluginMetadata, pluginName) // TODO: Goutham : verify Is this needed instead. Can't we store failed plugins ?
		return lomcommon.LogError("Failed to register plugin %s with engine", pluginName)
	}

	//delete(plmgr.plugins, pluginName) // TODO: Goutham : verify, Is this needed instead.
	//delete(plmgr.pluginMetadata, pluginName) // TODO: Goutham : verify Is this needed instead.
	return nil
}

// TODO: Goutham : Add this function in lomcommon framework
// If requirement is to add periodic log entry with shorttime and then falback to longtime, use this function.
func (plmgr *PluginManager) AddPeriodicLogWithTimeouts(ID string, message string, shortTimeout time.Duration,
	longTimeout time.Duration, context string) chan bool {
	// Create a channel to listen for stop signals to kill timer
	stopchannel := make(chan bool)

	lomcommon.GetGoroutineTracker().Start("AddPeriodicLogWithTimeouts"+ID+lomcommon.GetUUID(), func() {
		// First add periodic log witj short timeout
		lomcommon.AddPeriodicLogInfo(ID, message, int(shortTimeout.Seconds())) // call lomcommon framework

		// Wait for the short timeout to expire or for stop signal
		select {
		case <-time.After(shortTimeout):
			// after short timeout expiry, update timer to longtimeout
			lomcommon.UpdatePeriodicLogTime(ID, int(longTimeout.Seconds()))
			break
		case <-stopchannel:
			lomcommon.RemovePeriodicLogEntry(ID)
			return
		}

		// Wait for the stop signal
		<-stopchannel

		// Stop signal received, remove the periodic log entry
		lomcommon.RemovePeriodicLogEntry(ID)
	})

	return stopchannel
}

/*****************************************************************************************/
/* 	  Helper Functions										      			             */
/*****************************************************************************************/

/*
 *  Create Plugin Instance. Multiple plugins can be created from this function
 * When a new plugin is added, add a case here to create the plugin instance and define the plugin in plugins_common package
 *
 * Input:
 *  pluginID - Plugin ID(pluginname, version) from plugins_common package
 *  pluginData - Plugin Data passed to plugin. For e.g. heartbeats timer, etc.
 *
 * Output:
 *  none -
 *
 * Return:
 *  plugin - Plugin instance
 *  pluginmetadata - Plugin metadata
 *  error - Error if any
 */
//TODO : Goutham : Look into beetter way to do this.
func CreatePluginInstance(pluginID plugins_common.PluginId, actionCfg *lomcommon.ActionCfg_t) (plugins_common.Plugin, plugins_common.IPluginMetadata, error) {
	switch pluginID.Name {
	case "GenericPluginDetection":
		plugin := &plugins_files.GenericPluginDetection{}
		pluginmetadata := &plugins_common.PluginMetadata{
			ActionCfg:   actionCfg,
			StartedTime: time.Now(),
			Pluginstage: plugins_common.PluginStageUnknown,
			PluginId:    pluginID,
			// ... other common metadata fields
		}
		return plugin, pluginmetadata, nil
	/*case "LinkFlapPluginDetection":
	plugin := &plugins_files.LinkFlapPluginDetection{}
	pluginmetadata := &LinkFlapPluginDetection{
		PluginMetadata: PluginMetadata{
			PluginData:   pluginData,
			StartedTime: time.Now(),
			Pluginstage:   PluginStageUnknown,
			PluginId:     pluginID,
			// ... other common metadata fields
		},
	}
	return plugin, pluginmetadata, nil*/
	default:
		return nil, nil, lomcommon.LogError("plugin not found: %s", pluginID.Name)
	}
}

// setup UNIX signals
func SetupSignals() error {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)

	lomcommon.GetGoroutineTracker().Start("HandleSyslogSignal"+lomcommon.GetUUID(),
		func() error {
			for {
				// Wait for a signal to be received
				val, ok := <-signalChan
				if ok {
					switch val {
					case syscall.SIGTERM:
						lomcommon.LogWarning("Received SIGTERM signal. Exiting plugin mgr:%s", ProcID)
						os.Exit(0)
					}
				}
			}
			return nil
		})

	return nil
}

// parse program command line arguments
func ParseArguments() {
	// Declare the command line flags
	var ProcIDFlag string
	var syslogLevelFlag int

	// Define the command line flags
	flag.StringVar(&ProcIDFlag, "proc_id", "", "Proc ID number")
	flag.IntVar(&syslogLevelFlag, "syslog_level", 7, "Syslog level")

	// Parse the command line arguments
	flag.Parse()

	if ProcIDFlag == "" {
		panic("Proc ID is not provided")
	}

	// assign to variables which can be accessed from process
	ProcID = ProcIDFlag
	lomcommon.SetLogLevel(syslog.Priority(syslogLevelFlag))

	fmt.Printf("Program Arguments : proc ID : %s, Syslog Level : %d\n", ProcIDFlag, syslogLevelFlag)
}

/****************************************************************************************/
/* 									Setup Functions 									*/
/****************************************************************************************/

// Start Plugin Manager - Create Plugin Manager, read each plugin name and its parameters from actions_conf file & Setup each plugin
func StartPluginManager() error {
	// Create & Start Plugin Manager and do registration with engine
	vclientTx := lomipc.GetClientTx(int(GOLIB_TIMEOUT_DEFAULT))
	vpluginManager := GetPluginManager(vclientTx)
	lomcommon.LogInfo("Plugin Manager created successfully")

	/* For a particular proc_X, read each plugin name and its parameters from proc_conf.json file &
	   Setup each plugin */
	procInfo, err := lomcommon.GetConfigMgr().GetProcsConfig(ProcID)
	if err != nil {
		return lomcommon.LogError("Error getting proc config for proc %s: %v", ProcID, err)
	}

	// TODO: Goutham : Note : Dynamic Plugins is not supported in V1 release. So no code is implemented for dynamic plugins
	for pluginname, plconfig := range procInfo {
		lomcommon.LogInfo("Initializing plugin %s version %s", pluginname, plconfig.Version)
		errv := vpluginManager.loadPlugin(pluginname, plconfig.Version)
		if errv != nil {
			lomcommon.LogError("Error Initializing plugin %s version %s : %v", pluginname, plconfig.Version, errv)
		} else {
			vpluginManager.pluginMetadata[pluginname].SetPluginStage(plugins_common.PluginStageLoadingSuccess)
			lomcommon.LogInfo("plugin %s version %s successfully Initialized", pluginname, plconfig.Version)
		}
	}

	lomcommon.GetGoroutineTracker().Start("StartPluginManager"+lomcommon.GetUUID(),
		vpluginManager.run())

	lomcommon.GetGoroutineTracker().WaitAll(0)

	// Reaches herer only when plugin manager is stopped

	return nil
}

// Setup Plugin Manager  - Parse program arguments, setup syslog signals, load environment variables, validate config files, etc
func SetupPluginManager() error {

	//parse program arguments & assign values to program variables. Hree proc_X value is read
	ParseArguments()

	// setup application prefix for logging
	lomcommon.SetPrefix(APP_NAME_DEAULT + ProcID)

	//syslog level change from UNIX signals
	err := SetupSignals()
	if err != nil {
		return lomcommon.LogError("Error setting up signals: %v", err)
	}

	// Initialize the config manager. This will read ENV config path location and  will read config files for attributes from there
	err = lomcommon.InitConfigPath("")
	if err != nil {
		lomcommon.LogError("Error initializing config manager: %s", err)
	}

	return nil
}
