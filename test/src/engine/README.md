Engine is the core btain behind that manages all plugins invoked by one or more plugin managers.
Plugins could be impelemented using different programming languages.
The pluginMgr handles the plugins and communicate server requests/responses appropriately.

Plugin managers: Communincate with Engine using lib/lomipc package APIs

Refer lib/Readme.md for details.

Engine speaks with PluginMgr only, which it treats as its clients. Any term "client" can be inferred
as "plugunMgr".

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
    If first action in any sequence,
        send request to the corresponding client addressed to this action


Action Deregister:
    Fail if corresponding client is not registered with error.
    Fail if not registered by its client with error.
    if action-state == in-progress:
        Follow process to abort the sequence.
    Remove from active action list.


On Server response:
    Publish this action with result.
    If failed:
        if not first action:
            send complete for the first action in sequence with faliure code.
            First action in sequence is usually detection and we indicate the completion
            of mitigation sequence as failed.
        else:
            Check failure frequency.
            If failed more than N times in M seconds:
                Mark it failed.
                Send failure log every O seconds, until de-registered.
    If succeeded:
        if not last action:
            Watch the timetaken.
            If time taken >= mitigatip
        
    
    
    Run mitigation sequence.

Abort sequence:
    First action is commonly detection. If not first, abort this action and send
    complete for the first action, which had something detected. If first, it just
`   means that we don't detect any more, so send de-activated.
    
    If not first action in sequence:
        Aborted event for this action with appropriate reason.
        Aborted event for the first action of the sequence with state="complete"
    else:
        Aborted event for the this action of the sequence with state="de-activated"
        
