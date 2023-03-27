Engine is the actions binding enforcer that honors publishing, timeouts, and action failures.
This is completely driven by actions & Bindings config.

Plugins could be impelemented using different programming languages.
The pluginMgr handles the plugins and communicate server requests/responses appropriately.

Plugin managers: Communicate with Engine using lib/lomipc package APIs
RFE: Other type of clients like ConfigMgr may talk to engine too, via this lib package.

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


At high level:
    1.  Each client must register itself before calling any other API. Likewwise, de-register
        client should be the last call by client and no more request can be made until after
        next register.

    2.  A client is expected to register all actions it manages. This is the only way
        engine realizes the action's reachability and engine handles until de-register
        of the action

    3.  Engine sends request to an action upon registration, if it is first action of
        any sequence. This request is sent with no timeout.

    4.  A sequence instance is created only upon receiving successful response from the first
        action upon successful response. If first action's response states failure,
        the request is re-raised for that request.

    5.  Upon sequence creation, the request is raised for second action and upon its
        response move to next and so on, until the sequence completes.
            a.  At anytime only one sequence can be active.
            b.  If first response arrives, when a sequence is already active, it is
                created and kept pending, with active timer running to watch its timeout.

    6.  A sequence may timeout, in two scenarios
            a.  The current active request's timeout triggered (implying the action
                has not responded before timeout)
                
                OR

            b.  The overall sequence timeout expired.

    7.  A sequence completes in any of the following scenarios
            a. All actions of the sequence are invoked and all responded successful
            b. An action in the sequence sent "failed" response
            c. A timeout occurred
            d. Active request for this sequence got de-registered

    8.  On sequence complete
            a.  The first action of the sequence is re-published with result code
                reflecting the completion state of the sequence 
            b.  The sequence is removed
            c.  NOTE: In case of timeout, the current request of the sequence could
                be still active and this request is not removed from the list.
            

    9.  Sequence completion follow up:
            a.  The first request of the sequence is re-raised
            b.  If any pendig sequence, the one with highest pri is resumed.

    9.  Active request:
            Once a request is raised to an action, it is not removed until 
            the corresponding response is received or the action is de-registered. This
            is true, even if this request's or corresponding sequence's timeout expired.
    
            This is because it implies that the corresponding action/plugin is still
            busy with it and by design another request can't be raised until it
            completes, however long.

            PluginMgr tracks timed requests and raise periodic error logging until
            action responds or action has to be de-registered for some reason.

            If a long standing request is part of a new/next active sequence, this sequence
            gets blocked, until this request completes, as it can't be re-raised until
            completion of last invoke.

APIs at high level:
Client Register:
    Fail if already registered with error.
    Create a local entry for this client.
    All actions registered by a client is internally tagged to its registration.

    
Client Deregister:
    No-op if no entry exist.
    For each action of this client, call action-deregister.
    Remove this client entry.
    Engine will not communicate with this client, until next register.
    
Action Register:
    Fail with error if any of the following is true
        corresponding client is not registered yet.
        This action is already registered (irrespective of which client)
    Register as active action with tag to its client registration.
    If it is first action in any sequence,
        raise request to the corresponding client addressed to this action


Action Deregister:
    Fail if corresponding client is not registered with error.
    Fail if not registered by this client with error.
    if action-state == in-progress,
        remove it (as no response is expected either).
        If part of any active sequence, fail the sequence.
        NOTE: Sequence is created only upon response from the first action.
        So if first action is being de-registered, there will not be any
        sequence to abort. Even for non-first actiobb
    Remove from active action list.


On Server response:
    Validate and publish it.
    If failed:
        if first action, re-raise the request.
        Else abort sequence, if present and follow seq complete process

    If succeeded:
        if first action in sequence, create an active sequence,
        else:
            if next action is not currently active (in-progress) raise the request,
            else if no more next action, follow seq complete process


On seq completion:
    1. Publish first action with state=complete and appropriate result
    2. If any pending sequence, resume one  with highest pri. Among ones
       from same pri, any is picked.


        
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


