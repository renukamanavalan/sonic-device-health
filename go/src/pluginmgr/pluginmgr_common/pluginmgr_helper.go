package pluginmgr_common

import (
	"errors"
	"flag"
	"fmt"
	"go/src/lib/lomcommon"
	"go/src/lib/lomipc"
	"go/src/plugins/plugins_common"
	"go/src/plugins/plugins_files"
	"log/syslog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

/*****************************************************************************************/
/* Plugin manager */
/*****************************************************************************************/

// Plugin Manager global variables
var (
	LOMConfFilesLocation string                      = ""
	goroutinetracker     *lomcommon.GoroutineTracker = nil
	configMgr            *lomcommon.ConfigMgr_t      = nil
)

// Constants for plugin manager with default values
const (
	GOLIB_TIMEOUT_DEFAULT                         = 1000 * time.Millisecond /* Default GoLIB API timeouts */
	PLUGIN_INIT_PERIODIC_FALLBACK_TIMEOUT_DEFAULT = 3600 * time.Second      /* Default Periodic logging long timeout */
	PLUGIN_INIT_PERIODIC_TIMEOUT_DEFAULT          = 300 * time.Second       /* Default periodic logging short timeout */
	PLUGIN_INIT_CALL_TIMEOUT_DEFAULT              = 60 * time.Second
	PLUGIN_LOADING_TIMEOUT_DEFAULT                = 30 * time.Second
	APP_NAME_DEAULT                               = "plugin_mgr"
	ACTIONS_CONF_DEFAULT                          = "actions_conf.json"
	PROC_CONF_DEFAULT                             = "proc_conf.json"
)

/*
This struct used to store & track periodic logging timer statistics.

Looking at this structure gives info about status of periodic timer, its active or not,
period of timer, context name where this timer is started etc. Useful in debugging
*/
type TimerData struct {
	contextname string    // plugis name or any context where this timer is started
	uid         string    // timer uuid
	status      bool      // ON/OFF
	period      int       // in seconds
	stopChannel chan bool // stop signal to stop timer
}

// Plugin Manager interface to be implemented by plugin manager
type IPluginManager interface {
	run(chan struct{}) error
	shutdown() error
	addPlugin(pluginName string, pluginVersion string, isDynamic bool) error
	registerActionWithEngine(pluginName string) error
	deRegisterActionWithEngine(pluginName string) error
	printTimerData() string
	addPeriodicLogNotice(ID string, message string, period int, context string)
	setPeriodicTimerData(contextname string, uid string, period int)
	removePeriodicLogEntry(ID string)
	updatePeriod(ID string, period int)
	addPeriodicLogWithTimeouts(ID string, message string, shortTimeout time.Duration, longTimeout time.Duration, context string) chan bool
	loadPlugins(pluginName string, pluginVersion string, isDynamic bool) error 
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
	pluginMetadata   map[string]plugins_common.IPluginMetadata /* map : pluginname -> PluginMetadata struct Object(at go/src/plugins/plugins_common) */
	clientRegistered  bool
	mu        sync.Mutex
	timerData map[string]TimerData /* map : "periodic timer ID" -> {TimerData struct object } */
	stopch chan struct{} /* channel used to stop listening on golib for server events */
						//TODO: Goutham : replace this with global stop channel
}

// InitPluginManager : Initialize plugin manager
func NewPluginManager(clientTx IClientTx) (*PluginManager, error) {
	pm := &PluginManager{
		clientTx:   clientTx,
		plugins:    make(map[string]plugins_common.Plugin),
		pluginMetadata:   make(map[string]plugins_common.IPluginMetadata),
		mu:         sync.Mutex{},
		timerData:  make(map[string]TimerData),
		stopch:     make(chan struct{}),
	}

	if pm.clientRegistered {
		return nil, errors.New("client already registered")
	}

	if err := pm.clientTx.RegisterClient(APP_NAME_DEAULT+lomcommon.ProcID+lomcommon.GetUUID()); err != nil {
		return nil, err
	}

	pm.clientRegistered = true

	return pm, nil
}

func (plmgr *PluginManager) getPlugin(plgname string) (plugins_common.Plugin, bool){
	plmgr.mu.Lock()
	plugin, ok := plmgr.plugins[plgname]
	plmgr.mu.Unlock()
	return plugin,ok
}

func (plmgr *PluginManager) getPluginMetadata(plgname string) (plugins_common.IPluginMetadata, bool){
	plmgr.mu.Lock()
	plugin, ok := plmgr.pluginMetadata[plgname]
	plmgr.mu.Unlock()
	return plugin,ok
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
			lomcommon.LogNotice("Stopping plugin manager run loop")
			return nil
		default:
			serverReq, err := plmgr.clientTx.RecvServerRequest()
			if err != nil {
				lomcommon.LogError("Error RecvServerRequest() : %s", err)
				return err
			}
			if serverReq == nil {
				return lomcommon.LogError("Error RecvServerRequest() : nil")
			}

			switch serverReq.ReqType {
			case lomipc.TypeServerRequestAction:
				actionReq, ok := serverReq.ReqData.(*lomipc.ActionRequestData)
				if !ok {
					lomcommon.LogError("Error in parsing ActionRequestData")
					continue
				}
								
				plugin, ok := plmgr.getPlugin(actionReq.Action)				
				if !ok {
					lomcommon.LogError("Plugin %s not found", actionReq.Action)
					continue
				}
				lomcommon.LogNotice("Received action request for plugin %s", plugin.GetPluginID())

				/* TODO: Goutham : Handle Request, Do error checks, pass HeartBEat channel, handle timeouts, handle heartbeats etc
				goroutinetracker.Start("plg_mgr_Run_Action_"+actionReq.Action+"_"+lomcommon.GetUUID(),
					func() {
						hbchan := make(chan plugins_common.PluginHeartBeat)// TODO: Goutham ; Do we need buffered instead ??
						res := plugin.Request(hbchan, actionReq)
						plmgr.clientTx.SendServerResponse(&lomipc.MsgSendServerResponse{
							ReqType: lomipc.TypeServerRequestAction,
							ResData: res,
						})
					})
				*/
			case lomipc.TypeServerRequestShutdown:
				// TODO: Goutham : handle shutdown, spawn goroutin here
			default:
				lomcommon.LogError("Unknown server request type")
			}
		}
	}
	return nil
}

// TODO: Goutham : For all plugins call shutdoen, change plugin state, etc
func (plmgr *PluginManager) shutdown() error {
	if err := plmgr.clientTx.DeregisterClient(); err != nil {
		return err
	}
	plmgr.stopch <- struct{}{}
	lomcommon.LogNotice("Plugin Manager shutdown")
	return nil
}

// TODO: Goutham : Dynamic plugins are not handled in V1 cut. Add this code later
/*
 * Calls plugin init() and f its successful calls registerAction() to engine. This call may block.
 *
 * Input:
 *  pluginname - plugin name
 *  pluginVersion - plugin version
 *  isDynamic - true if plugin is dynamic, false if plugin is static
 *
 * Output:
 *  none -
 *
 * Return:
 *  error - error message or nil on success
 */
func (plmgr *PluginManager) addPlugin(pluginName string, pluginVersion string, isDynamic bool) error {
	// 1.Check if plugin is already loaded
	if _, ok := plmgr.getPlugin(pluginName); ok {
		return lomcommon.LogError("plugin with name %s and version %s is already loaded", pluginName, pluginVersion)
	}

	// 2.Get plugin specific details from actions config file and add any additional info to pass to plugin's init() call
	var pluginData plugins_common.PluginData
	actionCfg, err := configMgr.GetActionConfig(pluginName)
	if err == nil {
		pluginData.Timeout = actionCfg.Timeout
		pluginData.HeartbeatInt = actionCfg.HeartbeatInt
		pluginData.ActionKnobs = actionCfg.ActionKnobs
	} else {
		return lomcommon.LogError("plugin %s not found in actions config file", pluginName)
	}

	// 3.Check if plugin disabled flag is set or not in the actions config file.
	if actionCfg.Disable {
		msg := fmt.Sprintf("plugin %s is disabled", pluginName)
		lomcommon.LogWarning(msg)
		return errors.New(msg)
	}

	// 4.Create new plugin instance
	pluginID := plugins_common.PluginId{Name: pluginName, Version: pluginVersion}
	plugin, pluginmetadata , err := CreatePluginInstance(pluginID, pluginData) // returns Plugin interface pointing to new plugin struct
	if err != nil {
		return err
	}
	
	// 5.Check if plugin name and version from proc_conf.json file matches the values in static plugin. If not log periodic log
	if id := plugin.GetPluginID(); id.Name != pluginName || id.Version != pluginVersion {
		plmgr.AddPeriodicLogWithTimeouts("Plg_mgr_AddPlugin_"+lomcommon.GetUUID()+"_"+pluginName,
			fmt.Sprintf("Plugin ID does not match provided arguments: got %s %s, expected %s %s", id.Name, id.Version, pluginName, pluginVersion),
			PLUGIN_INIT_PERIODIC_TIMEOUT_DEFAULT, PLUGIN_INIT_PERIODIC_FALLBACK_TIMEOUT_DEFAULT, "")
		return errors.New("")
	}

	// 6.Add plugin to plugin manager
	plmgr.mu.Lock()
	plmgr.plugins[pluginName] = plugin
	plmgr.pluginMetadata[pluginName] = pluginmetadata
	plmgr.mu.Unlock()

	// 7. call plugin's init() call synchronously
	err = plugin.Init(pluginData)
	if err != nil {
		//delete(plmgr.plugins, pluginName) // TODO: Goutham : verify Is this needed instead?
		//delete(plmgr.pluginMetadata, pluginName) // TODO: Goutham : verify Is this needed instead.
		return lomcommon.LogError("plugin %s init failed: %v", pluginName, err)
	}

	// 8. call plugin's registerAction() call synchronously
	err = plmgr.registerActionWithEngine(pluginName)
	if err != nil {
		//delete(plmgr.plugins, pluginName) // TODO: Goutham : verify Is this needed instead?
		//delete(plmgr.pluginMetadata, pluginName) // TODO: Goutham : verify Is this needed instead. 
		return lomcommon.LogError("plugin %s registerAction failed: %v", pluginName, err)
	}

	return nil
}

func (plmgr *PluginManager) loadPlugins(pluginName string, pluginVersion string, isDynamic bool) error {
    // Create a channel to receive the result of AddPlugin()
    resultChan := make(chan error)

    // Create a goroutine to execute AddPlugin
    go func() {
        resultChan <- plmgr.addPlugin(pluginName, pluginVersion, isDynamic)
    }()

	goroutinetracker.Start("plg_mgr_LoadPlugins"+pluginName, func() { 
		resultChan <- plmgr.addPlugin(pluginName, pluginVersion, isDynamic)
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
	plmgr.mu.Lock()
	defer plmgr.mu.Unlock()

	if err := plmgr.clientTx.RegisterAction(pluginName); err != nil {
		//delete(plmgr.plugins, pluginName) // TODO: Goutham : verify Is this needed instead. Can't we store failed plugins ?
		//delete(plmgr.pluginMetadata, pluginName) // TODO: Goutham : verify Is this needed instead. Can't we store failed plugins ?
		return lomcommon.LogError("Failed to register plugin %s with engine", pluginName)
	}

	return nil
}

// DeRegisterActionWithEngine : DeRegister plugin with engine
func (plmgr *PluginManager) deRegisterActionWithEngine(pluginName string) error {
	plmgr.mu.Lock()
	defer plmgr.mu.Unlock()

	if err := plmgr.clientTx.DeregisterAction(pluginName); err != nil {
		//delete(plmgr.plugins, pluginName) // TODO: Goutham : , Is this needed instead. Can't we store failed plugins ?
		//delete(plmgr.pluginMetadata, pluginName) // TODO: Goutham : verify Is this needed instead. Can't we store failed plugins ?
		return lomcommon.LogError("Failed to register plugin %s with engine", pluginName)
	}

	//delete(plmgr.plugins, pluginName) // TODO: Goutham : verify, Is this needed instead.
	//delete(plmgr.pluginMetadata, pluginName) // TODO: Goutham : verify Is this needed instead.
	return nil
}

// Print periodic timer data. Used for debugging purposes
func (plmgr *PluginManager) printTimerData() string {
	plmgr.mu.Lock()
	defer plmgr.mu.Unlock()

	var b strings.Builder
	b.WriteString("Timer Data:\n")

	for key, value := range plmgr.timerData {
		b.WriteString(fmt.Sprintf("  uid: %s\n", key))
		b.WriteString(fmt.Sprintf("  contextname: %s\n", value.contextname))
		b.WriteString(fmt.Sprintf("  status: %t\n", value.status))
		b.WriteString(fmt.Sprintf("  period: %d\n", value.period))
		b.WriteString("\n")
	}

	return b.String()
}

// Period in sec
// Helper function on top of periodic log framework along with storing the ID & other info for tracking purposes
func (plmgr *PluginManager) addPeriodicLogNotice(ID string, message string, period int, context string) {
	lomcommon.AddPeriodicLogNotice(ID, message, period)  // call lomcommon framework
	plmgr.setPeriodicTimerData(context, ID, period, nil) // Store this info for tracking purpose
}

// Period in sec
// Interal API to store periodic log entries for tracking purposes
func (plmgr *PluginManager) setPeriodicTimerData(contextname string, uid string, period int, stopchannel chan bool) {
	plmgr.mu.Lock()
	defer plmgr.mu.Unlock()
	timerData, ok := plmgr.timerData[uid]
	if !ok {
		timerData = TimerData{} // If ID do not exist, create new
	}
	timerData.contextname = contextname
	timerData.uid = uid
	timerData.status = true // activate
	timerData.period = period
	if stopchannel == nil {
		timerData.stopChannel = make(chan bool)
	}
	plmgr.timerData[uid] = timerData
}

// Period in sec
// Drop peridic log for ID
func (plmgr *PluginManager) removePeriodicLogEntry(ID string) {
	plmgr.mu.Lock()
	defer plmgr.mu.Unlock()
	timerData, ok := plmgr.timerData[ID]
	//Return if ID is not there or timer is inactive
	if !ok || !timerData.status {
		return
	}
	timerData.status = false
	plmgr.timerData[ID] = timerData
	// TODO: Goutham : verify, Check if its better to delete this ID entry from map and drain channel ?
	lomcommon.RemovePeriodicLogEntry(ID)
}

// Period in sec
// Update period of existing periodic log ID
func (plmgr *PluginManager) updatePeriod(ID string, period int) {
	plmgr.mu.Lock()
	defer plmgr.mu.Unlock()
	timerData, ok := plmgr.timerData[ID]
	if !ok {
		return
	}
	timerData.period = period
	plmgr.timerData[ID] = timerData
	lomcommon.UpdatePeriodicLogTime(ID, period)
}

// If requirement is to add periodic log entry with shorttime and then falback to longtime, use this function.
func (plmgr *PluginManager) AddPeriodicLogWithTimeouts(ID string, message string, shortTimeout time.Duration,
	longTimeout time.Duration, context string) chan bool {
	// Create a channel to listen for stop signals to kill timer
	stopchannel := make(chan bool)

	goroutinetracker.Start("AddPeriodicLogWithTimeouts"+ID+lomcommon.GetUUID(), func() {
		// First add periodic log witj short timeout
		lomcommon.AddPeriodicLogNotice(ID, message, lomcommon.DurationToSeconds(shortTimeout)) // call lomcommon framework
		plmgr.setPeriodicTimerData(context, ID, lomcommon.DurationToSeconds(shortTimeout), stopchannel)

		// Wait for the short timeout to expire or for stop signal
		select {
		case <-time.After(shortTimeout):
			// after short timeout expiry, update timer to longtimeout
			plmgr.updatePeriod(ID, lomcommon.DurationToSeconds(longTimeout))
			break
		case <-stopchannel:
			plmgr.removePeriodicLogEntry(ID)
			return
		}

		// Wait for the stop signal
		<-stopchannel

		// Stop signal received, remove the periodic log entry
		plmgr.removePeriodicLogEntry(ID)
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
func CreatePluginInstance(pluginID plugins_common.PluginId, pluginData plugins_common.PluginData) (plugins_common.Plugin, plugins_common.IPluginMetadata, error) {
	switch pluginID.Name {
	case "GenericPluginDetection":
		plugin := &plugins_files.GenericPluginDetection{}
		pluginmetadata := &plugins_common.PluginMetadata{
			Plugindata:  pluginData,
			StartedTime: time.Now(),
			Pluginstage: plugins_common.PluginStageUnknown,
			PluginId:    pluginID,			
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
		},
	}
	return plugin, pluginmetadata, nil*/
	default:
		return nil, nil, lomcommon.LogError("plugin not found: %s", pluginID.Name)
	}
}

// setup UNIX signals to change syslog level on running program
func SetupSyslogSignals() error {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGUSR1, syscall.SIGUSR2)

	logFun := func() {
		// Log a message with the new syslog level
		var logMessage string = "Changed syslog level to "
		log_level := lomcommon.GetLogLevel()
		switch log_level {
		case syslog.LOG_EMERG:
			lomcommon.LogMessage(log_level, logMessage+"EMERG")
		case syslog.LOG_ALERT:
			lomcommon.LogMessage(log_level, logMessage+"ALERT")
		case syslog.LOG_CRIT:
			lomcommon.LogMessage(log_level, logMessage+"CRIT")
		case syslog.LOG_ERR:
			lomcommon.LogMessage(log_level, logMessage+"ERR")
		case syslog.LOG_WARNING:
			lomcommon.LogMessage(log_level, logMessage+"WARNING")
		case syslog.LOG_NOTICE:
			lomcommon.LogMessage(log_level, logMessage+"NOTICE")
		case syslog.LOG_INFO:
			lomcommon.LogMessage(log_level, logMessage+"INFO")
		case syslog.LOG_DEBUG:
			lomcommon.LogMessage(log_level, logMessage+"DEBUG")
		}
	}

	goroutinetracker.Start("HandleSyslogSignal"+lomcommon.GetUUID(),
		func() error {
			for {
				// Wait for a signal to be received
				val, ok := <-signalChan
				if ok {
					// Update the syslog level based on the received signal
					log_level := lomcommon.GetLogLevel()
					switch val {
					case syscall.SIGUSR1:
						if log_level < syslog.LOG_DEBUG {
							lomcommon.SetLogLevel(log_level + 1)
							logFun()
						}
					case syscall.SIGUSR2:
						if log_level > syslog.LOG_EMERG {
							lomcommon.SetLogLevel(log_level - 1)
							logFun()
						}
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
	flag.StringVar(&ProcIDFlag, "proc_id", "proc_0", "Proc ID number")
	flag.IntVar(&syslogLevelFlag, "syslog_level", 5, "Syslog level")

	// Parse the command line arguments
	flag.Parse()

	// assign to variables which can be accessed from process
	lomcommon.ProcID = ProcIDFlag
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
	vpluginManager, err := NewPluginManager(vclientTx)
	if err != nil {
		return lomcommon.LogError("Error creating Plugin Manager: %v", err)
	}
	lomcommon.LogNotice("Plugin Manager created successfully")

	/* For a particular proc_X, read each plugin name and its parameters from proc_conf.json file &
	   Setup each plugin */
	procInfo, err := configMgr.GetProcsConfig(lomcommon.ProcID)
	if err != nil {
		return lomcommon.LogError("Error getting proc config for proc %s: %v", lomcommon.ProcID, err)
	}
	for pluginname, plconfig := range procInfo {
		// check for path empty to determine if plugin must be loaded from dynamic path or not
		isDynamic := false
		if plconfig.Path == "" {
			lomcommon.LogNotice("Plugin %s path is empty. Found dynamic plugin", pluginname)
			isDynamic = true
		}
		
		// TODO: Goutham : Note : Dynamic Plugins is not supported in V1 release. So no code is implemented
		lomcommon.LogNotice("Initializing plugin %s version %s isDynamic : %d", pluginname, plconfig.Version, isDynamic)
		vpluginManager.pluginMetadata[pluginname].SetPluginStage(plugins_common.PluginStageLoadingStarted)
		errv := vpluginManager.loadPlugins(pluginname, plconfig.Version, isDynamic)
		if errv != nil {
			vpluginManager.pluginMetadata[pluginname].SetPluginStage(plugins_common.PluginStageLoadingError)
			lomcommon.LogError("Error Initializing plugin %s version %s isDynamic : %d: %v", pluginname, plconfig.Version, isDynamic, errv)
		} else {
			vpluginManager.pluginMetadata[pluginname].SetPluginStage(plugins_common.PluginStageLoadingSuccess)
			lomcommon.LogNotice("plugin %s version %s isDynamic : %d successfully Initializing", pluginname, plconfig.Version, isDynamic)
		}
	}

	goroutinetracker.Start("StartPluginManager"+lomcommon.GetUUID(),
			vpluginManager.run())

	goroutinetracker.WaitAll(0)

	// Should never reach here

	return nil
}

// Setup Plugin Manager  - Parse program arguments, setup syslog signals, load environment variables, validate config files, etc
func SetupPluginManager() error {

	//parse program arguments & assign values to program variables. Hree proc_X value is read
	ParseArguments()

	//syslog level change from UNIX signals
	err := SetupSyslogSignals()
	if err != nil {
		return lomcommon.LogError("Error setting up syslog signals: %v", err)
	}

	//Read environmnet variables from system
	lomcommon.LoadEnvironmentVariables()

	// Check for ENV variable LOM_CONF_LOCATION
	if val, ok := lomcommon.GetEnvVarString("ENV_lom_conf_location"); ok && val != "" {
		LOMConfFilesLocation = val
	} else {
		return lomcommon.LogError("Error getting lom conf location from ENV")
	}

	//Get config files and validate them
	configFiles := &lomcommon.ConfigFiles_t{}
	if val, ok := lomcommon.ValidateConfigFile(LOMConfFilesLocation, ACTIONS_CONF_DEFAULT); ok != nil {
		return lomcommon.LogError("Error validating config file: %v", ACTIONS_CONF_DEFAULT)
	} else {
		configFiles.ActionsFl = val
	}
	if val, ok := lomcommon.ValidateConfigFile(LOMConfFilesLocation, PROC_CONF_DEFAULT); ok != nil {
		return lomcommon.LogError("Error validating config file: %v", PROC_CONF_DEFAULT)
	} else {
		configFiles.ProcsFl = val
	}

	// Call InitConfigMgr to initialize the config manager. This will read config files for attributes
	t, err := lomcommon.InitConfigMgr(configFiles)
	if err != nil {
		lomcommon.LogError("Error initializing config manager: %s", err)
	}
	configMgr = t

	// Create Goroutine Tracker which will be used to track all goroutines in the process
	goroutinetracker = lomcommon.NewGoroutineTracker()
	if goroutinetracker == nil {
		return lomcommon.LogError("Error creating goroutine tracker")
	}

	return nil
}

