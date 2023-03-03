Summary:
    Provides shared common APIs, like log related
    Provides APIs for clients to communicate with Engine
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
            
    
    All LoM code (plugins, pluginMgr, Engine, Config-Mgr, ...) uses this single set of APIs for
    logging. This helps us provide a unified presentation of LoM logs and ability to tweak any
    in one place for entire LoM

TODO:
    Add facility for callers to override syslog provided Writer with any. 
    This is an *advanced* use case and hence we do, when we see a need.


lomipc:
    Provides APIs for clients to use to contact Engine

        ClientTx - Client object that is created once & used for entire session
            All calls below are methods on this object.
            The timeout value for any call can be set in this object 
            Every API below is blocking until timeout.

        RegisterClient:
            Expected to be the first call to engine.
            Create the comm channel with server.
            Send register request to Engine. Wait for server response; Return
            nil or non nil error, depending upon server response as success/failed.
            
        DeregisterClient
            Expected to be the last call to engine.
            Deletes the comm channel with server.
            Send deregister request to Engine. Wait for server response; Return
            nil or non nil error, depending upon server response as success/failed.

        RegisterAction
            Registers a valid action upon loading it.
            Send register request to Engine. Wait for server response; Return
            nil or non nil error, depending upon server response as success/failed.

        DeregisterAction
            Deregisters a valid action upon any fatal error or upon need to reload it.
            Send deregister request to Engine. Wait for server response; Return
            nil or non nil error, depending upon server response as success/failed.

        RecvServerRequest
            Call for any server request. 
            The request can be ActionRequest to any action or a shutdown request.
            Send the request to read it; Wait for server response; Return a tuple
            (read-request, error). When error is non nil, the requset is nil.

        SendServerResponse
            Any response from plugin for request from engine is returned back to engine.
            Response is sent to server. Wait for server response; Return
            nil or non nil error, depending upon server response as success/failed.

        NotifyHeartbeat
            Send heartbeat from any action to engine.
            Send notification to engine. Wait for server response; Return
            nil or non nil error, depending upon server response as success/failed.

    APIs for making remote write and local read by engine/server
        SendToServer
            An internal API that encodes and send any

        ReadClientRequest
            Internal API for server to read any client request


Unit test:
    Exercise all the above code.
    Try to be data driven where possible
    Get test result as "code coverage" >= 85%

    
