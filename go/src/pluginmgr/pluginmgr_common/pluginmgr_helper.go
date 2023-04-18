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

// Plugin Manmamger interface to be implemented by plugin manager
type IPluginManager interface {
	Run(chan struct{}) error
	Shutdown() error
	AddPlugin(pluginName string, pluginVersion string, isDynamic bool) error
	RegisterActionWithEngine(pluginName string) error
	DeRegisterActionWithEngine(pluginName string) error
	PrintTimerData() string
	AddPeriodicLogNotice(ID string, message string, period int, context string)
	SetPeriodicTimerData(contextname string, uid string, period int)
	RemovePeriodicLogEntry(ID string)
	UpdatePeriod(ID string, period int)
	AddPeriodicLogWithTimeouts(ID string, message string, shortTimeout time.Duration, longTimeout time.Duration, context string) chan bool
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
	mu        sync.Mutex
	timerData map[string]TimerData /* map : "periodic timer ID" -> {TimerData struct object } */
}

// InitPluginManager : Initialize plugin manager
func NewPluginManager(clientTx IClientTx) (*PluginManager, error) {
	if err := clientTx.RegisterClient(LOMConfFilesLocation); err != nil {
		return nil, err
	}

	return &PluginManager{
		clientTx:  clientTx,
		plugins:   make(map[string]plugins_common.Plugin),
		mu:        sync.Mutex{},
		timerData: make(map[string]TimerData),
	}, nil
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
func (plmgr *PluginManager) Run(stop chan struct{}) error {
	for {
		select {
		case <-stop:
			return nil
		default:
			serverReq, err := plmgr.clientTx.RecvServerRequest()
			if err != nil {
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

				plmgr.mu.Lock()
				plugin, ok := plmgr.plugins[actionReq.Action]
				plmgr.mu.Unlock()
				if !ok {
					lomcommon.LogError("Plugin %s not found", actionReq.Action)
					continue
				}
				lomcommon.LogNotice("Received action request for plugin %s", plugin.GetPluginID())

				/* TO-DO : Goutham : Handle Request, Do error checks, pass HeartBEat channel, handle timeouts, handle heartbeats etc
				goroutinetracker.Start("plg_mgr_Run_Action_"+actionReq.Action+"_"+lomcommon.GetUUID(),
					func() {
						hbchan := make(chan plugins_common.PluginHeartBeat)// TODO : Goutham ; Do we need buffered instead ??
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
func (plmgr *PluginManager) Shutdown() error {
	if err := plmgr.clientTx.DeregisterClient(); err != nil {
		return err
	}
	lomcommon.LogNotice("Plugin Manager shutdown")
	return nil
}

// To-Do : Goutham : Dynamic plugins are not handled in V1 cut. Add this code later
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
func (plmgr *PluginManager) AddPlugin(pluginName string, pluginVersion string, isDynamic bool) error {
	// 1.Check if plugin is already loaded
	plmgr.mu.Lock()
	if _, ok := plmgr.plugins[pluginName]; ok {
		return lomcommon.LogError("plugin with name %s and version %s is already loaded", pluginName, pluginVersion)
	}
	plmgr.mu.Unlock()

	// 2.Get plugin specific details from actions config file and add any additional info to pass to plugin's init() call
	var pluginData plugins_common.PluginData
	if actionCfg, ok := configMgr.ActionsConfig[pluginName]; ok {
		pluginData.Timeout = actionCfg.Timeout
		pluginData.HeartbeatInt = actionCfg.HeartbeatInt
		pluginData.ActionKnobs = actionCfg.ActionKnobs
	} else {
		return lomcommon.LogError("plugin %s not found in actions config file", pluginName)
	}

	// 3.Check if plugin disabled flag is set or not in the actions config file.
	if configMgr.ActionsConfig[pluginName].Disable {
		return lomcommon.LogError("plugin %s is disabled", pluginName)
	}

	// 4.Create new plugin instance
	pluginID := plugins_common.PluginId{Name: pluginName, Version: pluginVersion}
	plugin, err := CreatePluginInstance(pluginID, pluginData) // returns Plugin interface pointing to new plugin struct
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
	plmgr.plugins[pluginName].SetPluginStage(plugins_common.PluginStageLoadingSuccess)
	plmgr.mu.Unlock()

	// 7.periodic timer unique string
	uid := "Plg_mgr_AddPlugin_" + lomcommon.GetUUID() + "_" + pluginName

	// 8.timer for plugin's init() call errors - SHort timeout - Default 5min
	logTimer := time.NewTimer(PLUGIN_INIT_PERIODIC_TIMEOUT_DEFAULT)
	if logTimer == nil {
		delete(plmgr.plugins, pluginName) // Verify : Goutham : Is this needed. Can't we store failed plugins ?
		return lomcommon.LogError("Failed to create logTimer")
	}
	if !logTimer.Stop() {
		<-logTimer.C
	}
	defer logTimer.Stop() // safe to call even if timer already expired
	logfreq := 0

	// 9.Start plugin init() call timeout's , 60sec by default untill init() responds
	initTimeout := time.NewTimer(PLUGIN_INIT_CALL_TIMEOUT_DEFAULT)
	if initTimeout == nil {
		delete(plmgr.plugins, pluginName) // Verify : Goutham : Is this needed. Can't we store failed plugins ?
		return lomcommon.LogError("Failed to create initTimeout")
	}

	// 10.Initialize plugin in goroutine and wait for completion or timeout. Pass
	initErrChan := make(chan error)
	initstartTime := time.Now()
	goroutinetracker.Start("plg_mgr_AddPlugin"+pluginName,
		func() {
			plmgr.plugins[pluginName].SetPluginStage(plugins_common.PluginStageInitStarted)
			initErrChan <- plugin.Init(pluginData)
		})

	initTimeoutCalled := false

	// 11. Wait for plugin's init() call to complete or timeout
	for {
		select {
		case err, ok := <-initErrChan: // error in plugin's init() call
			isError := false
			// for any errors, first log every short interval(5 min default)
			if !ok {
				plmgr.AddPeriodicLogNotice(uid, fmt.Sprintf("Failed to initialize plugin %s: initErrChan closed unexpectedly", pluginName),
					lomcommon.DurationToSeconds(PLUGIN_INIT_PERIODIC_TIMEOUT_DEFAULT), "")
				isError = true
			}
			if err != nil {
				plmgr.AddPeriodicLogNotice(uid, fmt.Sprintf("Failed to initialize plugin %s: %s", pluginName, err.Error()),
					lomcommon.DurationToSeconds(PLUGIN_INIT_PERIODIC_TIMEOUT_DEFAULT), "")
				isError = true
			}
			if isError {
				initTimeoutCalled = false
				if !initTimeout.Stop() {
					<-initTimeout.C
				}
				logTimer.Reset(PLUGIN_INIT_PERIODIC_TIMEOUT_DEFAULT) // log very frequently every 5min default
				continue
			}
			if time.Since(initstartTime) <= (PLUGIN_INIT_CALL_TIMEOUT_DEFAULT + 1*time.Second) {
				// Got some non error status from init() within 60sec (default), so plujgin is sucecssfully initialized.
				// To-Do : Goutham : Define & Change remaining metadata fields
				pluginMetadata := plugin.GetMetadata()
				pluginMetadata.StartedTime = time.Now()
				plugin.SetMetadata(pluginMetadata)
				plmgr.plugins[pluginName].SetPluginStage(plugins_common.PluginStageInitSuccess)

				// Call register action with engine
				goroutinetracker.Start("plg_mgr_AddPlugin_register"+pluginName,
					plmgr.RegisterActionWithEngine(pluginName), pluginName)
				return nil

			} else {
				// Got success from plugin's init() after 60 seconds(default time). So discard this status and clear any periodic logging. Treat it like Plugin initialization failed
				pluginMetadata := plugin.GetMetadata()
				pluginMetadata.StartedTime = time.Now()
				plugin.SetMetadata(pluginMetadata)
				plmgr.RemovePeriodicLogEntry(uid)
				plmgr.plugins[pluginName].SetPluginStage(plugins_common.PluginStageInitFailure)
				//delete(plmgr.plugins, pluginName) // Verify : Goutham : Is this needed instead?
				return lomcommon.LogError("Failed to initialize plugin %s: init() returned success after 60sec(default)", pluginName)
			}
		case <-logTimer.C: // Fired after every 5min (default)
			if logfreq == 0 || logfreq%12 == 0 {
				plmgr.UpdatePeriod(uid, lomcommon.DurationToSeconds(PLUGIN_INIT_PERIODIC_FALLBACK_TIMEOUT_DEFAULT)) // fall back from frequent logging to 1Hr(default) window to avoid syslog pollution
				logfreq = logfreq % 12
				if !logTimer.Stop() {
					<-logTimer.C
				}
				if !initTimeoutCalled {
					break // only break if there is error returned from plugin's init()
				}
			}
			logfreq++
		case <-initTimeout.C: // Fired after 60sec(default) , init() didn't returned anything
			initTimeoutCalled = true
			plmgr.AddPeriodicLogNotice(uid, fmt.Sprintf("Timed out on waiting for plugin %s to initialize", pluginName),
				lomcommon.DurationToSeconds(PLUGIN_INIT_PERIODIC_TIMEOUT_DEFAULT), "")
			logTimer.Reset(PLUGIN_INIT_PERIODIC_TIMEOUT_DEFAULT) // log very frequently every 5 min(default)
		}
	}
}

func (plmgr *PluginManager) RegisterActionWithEngine(pluginName string) error {
	plmgr.mu.Lock()
	defer plmgr.mu.Unlock()

	plmgr.plugins[pluginName].SetPluginStage(plugins_common.PluginStageServerRegistrationStarted)

	if err := plmgr.clientTx.RegisterAction(pluginName); err != nil {
		plmgr.plugins[pluginName].SetPluginStage(plugins_common.PluginStageServerRegistrationFailed)
		//delete(plmgr.plugins, pluginName) // verify : Goutham : Is this needed instead. Can't we store failed plugins ?
		return lomcommon.LogError("Failed to register plugin %s with engine", pluginName)
	}

	plmgr.plugins[pluginName].SetPluginStage(plugins_common.PluginStageServerRegistrationSuccess)
	return nil
}

// DeRegisterActionWithEngine : DeRegister plugin with engine
func (plmgr *PluginManager) DeRegisterActionWithEngine(pluginName string) error {
	plmgr.mu.Lock()
	defer plmgr.mu.Unlock()

	plmgr.plugins[pluginName].SetPluginStage(plugins_common.PluginStageServerDeRegistrationStarted)

	if err := plmgr.clientTx.DeregisterAction(pluginName); err != nil {
		plmgr.plugins[pluginName].SetPluginStage(plugins_common.PluginStageServerDeRegistrationFailed)
		//delete(plmgr.plugins, pluginName) // verify : Goutham : Is this needed instead. Can't we store failed plugins ?
		return lomcommon.LogError("Failed to register plugin %s with engine", pluginName)
	}

	plmgr.plugins[pluginName].SetPluginStage(plugins_common.PluginStageServerDeRegistrationSuccess)
	//delete(plmgr.plugins, pluginName) // verify : Goutham : Is this needed instead. Can't we store failed plugins ?
	return nil
}

// Print periodic timer data. Used for debugging purposes
func (plmgr *PluginManager) PrintTimerData() string {
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
func (plmgr *PluginManager) AddPeriodicLogNotice(ID string, message string, period int, context string) {
	if !logPeriodicActive {
		return
	}
	lomcommon.AddPeriodicLogNotice(ID, message, period)  // call lomcommon framework
	plmgr.SetPeriodicTimerData(context, ID, period, nil) // Store this info for tracking purpose
}

// Period in sec
// Interal API to store periodic log entries for tracking purposes
func (plmgr *PluginManager) SetPeriodicTimerData(contextname string, uid string, period int, stopchannel chan bool) {
	if !logPeriodicActive {
		return
	}
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
func (plmgr *PluginManager) RemovePeriodicLogEntry(ID string) {
	if !logPeriodicActive {
		return
	}
	plmgr.mu.Lock()
	defer plmgr.mu.Unlock()
	timerData, ok := plmgr.timerData[ID]
	//Return if ID is not there or timer is inactive
	if !ok || !timerData.status {
		return
	}
	timerData.status = false
	plmgr.timerData[ID] = timerData
	//verify : Goutham : Check if its better to delete this ID entry from map and drain channel ?
	lomcommon.RemovePeriodicLogEntry(ID)
}

// Period in sec
// Update period of existing periodic log ID
func (plmgr *PluginManager) UpdatePeriod(ID string, period int) {
	if !logPeriodicActive {
		return
	}
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
	if !logPeriodicActive {
		return nil
	}

	// Create a channel to listen for stop signals to kill timer
	stopchannel := make(chan bool)

	goroutinetracker.Start("AddPeriodicLogWithTimeouts"+ID+lomcommon.GetUUID(), func() {
		// First add periodic log witj short timeout
		lomcommon.AddPeriodicLogNotice(ID, message, lomcommon.DurationToSeconds(shortTimeout)) // call lomcommon framework
		plmgr.SetPeriodicTimerData(context, ID, lomcommon.DurationToSeconds(shortTimeout), stopchannel)

		// Wait for the short timeout to expire or for stop signal
		select {
		case <-time.After(shortTimeout):
			// after short timeout expiry, update timer to longtimeout
			plmgr.UpdatePeriod(ID, lomcommon.DurationToSeconds(longTimeout))
			break
		case <-stopchannel:
			plmgr.RemovePeriodicLogEntry(ID)
			return
		}

		// Wait for the stop signal
		<-stopchannel

		// Stop signal received, remove the periodic log entry
		plmgr.RemovePeriodicLogEntry(ID)
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
 *  error - error message or nil on success
 */
func CreatePluginInstance(pluginID plugins_common.PluginId, pluginData plugins_common.PluginData) (plugins_common.Plugin, error) {
	switch pluginID.Name {
	case "GenericPluginDetection":
		plugin := &plugins_files.GenericPluginDetection{
			PluginMetadata: plugins_common.PluginMetadata{
				Plugindata:  pluginData,
				StartedTime: time.Now(),
				Pluginstage: plugins_common.PluginStageUnknown,
				PluginId:    pluginID,
			},
		}
		return plugin, nil
	/*case "LinkFlapPluginDetection":
	plugin := &LinkFlapPluginDetection{
		PluginMetadata: PluginMetadata{
			PluginData:   pluginData,
			StartedTime: time.Now(),
			Pluginstage:   PluginStageUnknown,
			PluginId:     pluginID,
		},
	}
	return plugin, nil*/
	default:
		return nil, lomcommon.LogError("plugin not found: %s", pluginID.Name)
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

/*****************************************************************************************/
/* 									Log Periodic Module helpers								 */
/*****************************************************************************************/
// LogPeriodic module variables and functions to initialize and cleanup the module
var (
	logPeriodicCleanup func()
	logPeriodicActive  bool
)

func SetupLogPeriodic() {
	// Initialize LogPeriodic module
	chAbort := make(chan interface{})
	logPeriodicActive = true
	lomcommon.LogPeriodicInit(chAbort)

	// Cleanup LogPeriodic module
	logPeriodicCleanup = func() {
		if chAbort != nil {
			close(chAbort)
			logPeriodicActive = false
		}
	}
}

func AbortLogPeriodic() {
	if logPeriodicCleanup != nil {
		logPeriodicCleanup()
		logPeriodicCleanup = nil
	}
}

/****************************************************************************************/
/* 									Setup Functions 									*/
/****************************************************************************************/

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
	lomcommon.LoadEnvironemntVariables()

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

	//setup log periodic module to log periodic messages
	SetupLogPeriodic()

	return nil
}

// Start Plugin Manager - Create Plugin Manager, read each plugin name and its parameters from actions_conf file & Setup each plugin
func StartPluginManager() error {
	// Create & Start Plugin Manager and do registration with engine
	clientTx := lomipc.GetClientTx(int(GOLIB_TIMEOUT_DEFAULT))
	pluginManager, err := NewPluginManager(clientTx)
	if err != nil {
		return lomcommon.LogError("Error creating Plugin Manager: %v", err)
	}

	/* For a particular proc_X, read each plugin name and its parameters from actions_conf file &
	   Setup each plugin */
	for pluginname, plconfig := range configMgr.ProcsConfig {
		// check for path empty to determine if plugin must be loaded from dynamic path or not
		isDynamic := false
		if plconfig.Path == "" {
			lomcommon.LogNotice("Plugin %s path is empty. Found dynamic plugin", pluginname)
			isDynamic = true
		}

		//To-Do : Goutham : Note : Dynamic Plugins is not supported in V1 release. So no code is implemented
		err := pluginManager.AddPlugin(pluginname, plconfig.Version, isDynamic)
		if err != nil {
			lomcommon.LogError("Error setting up plugin : %s, Skipping ...", pluginname)
		}
	}

	goroutinetracker.Start("StartPluginManager"+lomcommon.GetUUID(),
		pluginManager.Run(nil))

	goroutinetracker.WaitAll()

	// Should never reach here

	return nil
}
