Summary:
    Provides shared common APIs, like log related
    Provides APIs for clients to communicate with Server.
    The server is referred as Engine as well. 
    The terms server & engine are used interchangeably.
    Unit tests


lomcommon:
    Provides APIs for
        GetLogLevel:
            Returns current log level

        SetLogLevel
            Accepts the log level to set
            Only messages with log level <= set log level will be logged.
            Default log level is syslog.LOG_ERR
            With log level set to syslog.LOG_DEBUG, all messages are printed
            to STDOUT too, to faciliate debugging.
           
        LogPanic
            Log messages with syslog.LOG_CRIT and call for process exit.

        LogError
            Log messages with syslog.LOG_ERR

        LogWarning
            Log messages with syslog.LOG_WARNING

        LogInfo
            Log messages with syslog.LOG_INFO

        LogDebug
            Log messages with syslog.LOG_DEBUG
            
    
    All LoM code (plugins, pluginMgr, Server, Config-Mgr, ...) uses this single set of APIs for
    logging. This helps us provide a unified presentation of LoM logs and ability to tweak any
    in one place for entire LoM

TODO:
    Add facility for callers to override syslog provided Writer with any. 
    This is an *advanced* use case and hence we do, when we see a need.


lomipc:
    Provides APIs for clients to use to contact Server

        ClientTx - Client object that is created once & used for entire session
            All calls below are methods on this object.
            The timeout value for any call can be set in this object 
            Every API below is blocking until timeout.

            type ClientTx struct {}

        RegisterClient:
            Expected to be the first call to server.
            Create the comm channel with server.
            Send register request to Server. Wait for server response; Return
            nil or non nil error, depending upon server response as success/failed.

            func (tx *ClientTx) RegisterClient(client string) error
            
        DeregisterClient
            Expected to be the last call to server.
            Deletes the comm channel with server.
            Send deregister request to Engine. Wait for server response; Return
            nil or non nil error, depending upon server response as success/failed.

            func (tx *ClientTx) DeregisterClient() error 

        RegisterAction
            Registers a valid action upon loading it.
            Send register request to Server. Wait for server response; Return
            nil or non nil error, depending upon server response as success/failed.

            func (tx *ClientTx) RegisterAction(action string) error

        DeregisterAction
            Deregisters a valid action upon any fatal error or upon need to reload it.
            Send deregister request to Server. Wait for server response; Return
            nil or non nil error, depending upon server response as success/failed.

            func (tx *ClientTx) DeregisterAction(action string) error

        RecvServerRequest
            Call for any server request. 
            The request can be ActionRequest to any action or a shutdown request.
            Send the request to read it; Wait for server response; Return a tuple
            (read-request, error). When error is non nil, the requset is nil.

            func (tx *ClientTx) RecvServerRequest() (*ServerRequestData, error)

        SendServerResponse
            Any response from plugin for request from server is returned back to server.
            Response is sent to server. Wait for server response; Return
            nil or non nil error, depending upon server response as success/failed.

            func (tx *ClientTx) SendServerResponse(res *ServerResponseData) 

        NotifyHeartbeat
            Send heartbeat from any action to server.
            Send notification to server. Wait for server response; Return
            nil or non nil error, depending upon server response as success/failed.
    
            func (tx *ClientTx) NotifyHeartbeat(action string, tstamp EpochSecs) error

    APIs for making remote write and local read by server
        SendToServer
            An internal API that encodes and send any

        ReadClientRequest
            Internal API for server to read any client request


Unit test:
    Exercise all the above code.
    Try to be data driven where possible
    Get test result as "code coverage" >= 85%

    
Note: Please refer code for struct details
