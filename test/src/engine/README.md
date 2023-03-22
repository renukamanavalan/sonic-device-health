Engine is the core brain behind that manages all plugins invoked by one or more plugin managers.
But the feed to the btain is bindings.conf

Plugins could be impelemented using different programming languages.
The pluginMgr handles the plugins and communicate server requests/responses appropriately.

Plugin managers: Communicate with Engine using lib/lomipc package APIs
RFE: Other type of clients like ConfigMgr may talk to engine too, vis this package.

Refer lib/Readme.md for details.

PluginMgr is one of the clients. The pluginMgr can run as multiple process instances and as 
well written in different programming languages. The PluginMgr written in Go can directly use
this package. We may likely provide c-binding to PluginMgr to plugin i/f, which will open up
languages to use for plugins.

Engine functionality:
1. Reads bindings.conf on startup and upon every SIGHUP
2. Bindings identify action sequence.
3. Accept client/action registrations/de-registrations from clients
4. Accept Action-response from clients for requests sent earlier.
5. Accept heartbeats from clients


Flow:
Client Register:
    Fail if already registered with error.
    Create a local entry for this client.
    All actions registered by a client is internally tagged to its registration.

    
Client Deregister:
    Fail if no entry exist.
    For each action of this client:
        if action-state == in-progress:
            Follow process to abort the sequence.
        remove action from active action list.

    
Action Register:
    Fail if corresponding client is not registered with error.
    Fail if already registered by any client with error.
    Register as active action with tag to its client registration.
    If it is first action in any sequence,
        raise request to the corresponding client addressed to this action


Action Deregister:
    Fail if corresponding client is not registered with error.
    Fail if not registered by its client with error.
    if action-state == in-progress:
        Follow process to abort the sequence.
    Remove from active action list.


On Server response:
    If failed:
        call abort sequence
    If succeeded:
        if only action in sequence:
            publish with state=complete result indicating no mitigation
            raise  
        else:
            if first action:
                Publish action with success & state=InProg
            else
                Publish action as succeedded

            if not last action in sequence:
                If time taken >= mitigation timeout:
                    abort the sequence
                send-requst for next action in sequence.
            if last action:
                re-publish first action with state = "complete" and success
                Raise request to first action.


Abort sequence:
    First action is commonly detection. If not first, abort this action and send
    complete for the first action, which had something detected. If first, it just
`   means that we don't detect any more, so send de-activated.
    
    If not first action in sequence:
        Aborted event for this action with appropriate reason.
        Aborted event for the first action of the sequence with state="complete"
    else:
        Check failure frequency.
        If failed more than N times in M seconds:
            Mark it failed.
            Send failure log every O seconds, until de-registered.

        
Raise Request:
    If first request in sequence:
        Send Action request with *no* context and no timeout set either.
        First actiopn is always detect
    else:
        Send Request to next action.
        If it is the second action in sequence:
            Raise a timer thread, that republish action-1 with state=InProg and time left
            to complete.
            Countdown starts from the time set for entire mitigation, at this point (Raise request
            to second in sequence) as start of mitigation.


SystemHeartbeat:
    Have a forever routine that sends heartbeat
    Each heartbeat lists all actions that notified since last heartbeat


NotifyHeartbeat:
    Request from client to notify engine on heartbeat sent by an action.
    Engine collects and reports on its heartbeats as list of actions that
    notified since last heartbeat by engine.


