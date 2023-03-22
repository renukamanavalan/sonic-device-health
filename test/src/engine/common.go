package engine

import (
    . "lib/lomcommon"
    . "lib/lomipc"
    "os"
    "os/signal"
    "syscall"
)

/* Action info is collected from register calls from clients */
type ActiveActionInfo_t {
    Action      string
    Client      string    /* Client that registered this action
    /*
     * Timeout to use if not set in sequence as obtained from actions.conf 
     */
    Timeout     int
    /* For now, Engine does not need any other config, hence not saved */
}

/* Info per client */
type ActiveClientInfo_t struct {

    ClientName  string

    /* List of registered actions by this client */
    Actions     map[string]struct{}

    /*
     * Clients read server's request via RecvServerRequest API.
     * This API sends LoMRequest for type = TypeRecvServerRequest
     * This is sent to LoMTransport.SendToServer.
     * The server is expected to send ServerRequestData via LomResponse
     *
     * Every accepted client connection run in its own Go routine as 
     * managed by HTTP-RPC client. Hence multiple instances of LoMTransport.SendToServer
     * will be running as one per client connection. All these instances
     * pipe requests into single tr.ServerCh channel.
     *
     * Unlike other requests, like RegisterClient, recvServer request can't be
     * served immediately but wait till the engine raise one. Engine would raise
     * upon processing a registerAction or upon processing a response from another
     * action. In other words this need to block
     *
     * So the handler for TypeRecvServerRequest, queues the request with client
     * via GetRegistrations().SendServerRequest(req). This writes the request into
     * pendingReadRequests channel. 
     *
     * Whenever a request to be raised to client (upon register action/process response/
     * ...), that sends the request into pendingWriteRequests, as there may or may not
     * be a read request from client pending.
     *
     * A dedicated Go routine ProcessSendRequests watches both and do the transfer
     * as appropriate.
     *
     * Each request from SendToServer has a request specific channel (LoMRequestInt::
     * ChResponse) for its response and return it via RPC to client's blocking
     * RecvServerRequest.
     */
    /* Pending requests per client */
    pendingWriteRequests chan *ServerRequestData
    pendingReadRequests chan req *LoMRequestInt

    /* To abort the go routine for ProcessSendRequests */
    abortCh chan interface{}
}

func (p *ActiveClientInfo_t) Close() {
    abortCh <- struct{}{}
}

func (p *ActiveClientInfo_t) ProcessSendRequests() {
    /*
     * Requests from server to client may come anytime.
     * Similarly, client's request to read server requests may come anytime
     * Requests from clients could come with timeout
     *
     * This handler watches for both and as well timeout and responds
     *
     * Client de-register call Close method, which sends abort to this
     * routine
     */
    type wait_t struct {
        req *LoMRequestInt
        due int64 
    }
    listWTimeout := make([]*wait_t, 0, 5)   /* Hold client requests for read */
    listNoTimeout := make([]*wait_t, 0, 5)  /* Hold client requests for read */
    serverRequests := make([]*ServerRequestData, 0, 10)
    tout := A_DAY_IN_SECS
    
    for {
        select {
        case clReq := <- pendingReadRequests:
            w := &wait_t { req: clReq }
            if req.TimeoutSecs == 0 {
                listNoTimeout = append(listNoTimeout, w)
            } else {
                tnow = time.Now().Unix()
                w.due = tnow + int64(req.TimeoutSecs)
                listWTimeout = append(listWTimeout, w)
                sort.Slice(listWTimeout, func(i, j int) bool {
                    listWTimeout[i].due < listWTimeout[j].due
                })
            }

        case serReq := <- pendingWriteRequests:
            serverRequests = append(serverRequests, serReq)

        case <- time.After(time.Duration(tout) * time.Second):
            /* bail out */

        case <- p.abortCh:
            LogInfo("Aborting Send requests")
            return
        }
        
        /* Here you come on client/server request or timeout */
        while len(serverRequests) > 0 {
            var r *LoMRequestInt = nil
            if len(listWTimeout) > 0 {
                r = listWTimeout[0]
                listWTimeout = listWTimeout[1:]
            } else if len(listNoTimeout) > 0 {
                r = listNoTimeout[0]
                listNoTimeout = listNoTimeout[1:]
            }
            if r != nil {
                r.ChResponse <- &LoMResponse {0, "", serverRequests[0]
                serverRequests = serverRequests[1:]
            } else {
                break
            }
        }
        if len(listWTimeout) > 0 {
            if tnow >= listWTimeout[0].due {
                tout = 0
            } else {
                tout = listWTimeout[0].due - tnow
            }
        } else {
            tout = A_DAY_IN_SECS
        }

    }
}


/* All registrations & heartbeats from clients. */
type ClientRegistrations_t struct {
    activeActions   map[string]*ActiveActionInfo_t
    activeClients   map[string]*ActiveClientInfo_t
    heartbeatCh     chan string     /* Action name */
}

/* Initialized object; But not exported */
var clientRegistrations *ClientRegistrations_t = nil


func GetRegistrations() *ClientRegistrations_t {
    return clientRegistrations
}


func InitRegistrations() *ClientRegistrations_t {
    clientRegistrations = &ClientRegistrations_t{
                make(map[string]*ActiveActionInfo_t)
                make(map[string]*ActiveClientInfo_t)
                make(chan string)
            }
    go clientRegistrations.PublishHeartbeats()
    return clientRegistrations
}

func (p *ClientRegistrations_t) RegisterClient(name string) error {
    if _, ok := p.ActiveClients[name]; ok {
        LogError("%s: Duplicate client registration; De & re-register", name)
        p.DeregisterClient(name)
    }
    cl := &ActiveClientInfo_t {
        ClientName: name,
        Actions: make(map[string]struct{}),
        pendingWriteRequests: make(chan *ServerRequestData),
        pendingReadRequests: make(chan *LoMRequestInt),
        abortCh: make(chan, interface{}) }
                
    go cl.ProcessSendRequests()
    p.ActiveClients[name] = cl
    return nil
}

func (p *ClientRegistrations_t) RegisterAction(action *ActiveActionInfo_t) error {
    cl, ok := p.ActiveClients[action.Client]
    if !ok {
        return LogError("%s: Missing client registration", action.Client)
    }
    else if r, ok1 := p.ActiveActions[action.Action]; ok1 {
        LogError("%s/%s: Duplicate action registration (%s); De/re-register,",
                action.Client, action.Action, r.Client)
        p.DeregisterAction(action.Action)
    }
    if cfg, err := GetConfigMgr().GetActionConfig(action.Action); err != nil {
        return LogError("%s: Missing action config", action.Action)
    } else if cfg.Disable {
        return LogError("%s: is disabled in config", action.Action)
    } else {
        if action.Timeout == 0 {
            action.Timeout = cfg.Timeout
        }
        cl.Actions[action.Action] = struct{}{}

        /* Make a copy and save */
        p.ActiveActions[action.Action] = &ActiveActionInfo_t{*action}
        return nil
    }
    GetSeqHandler().RaiseRequest(action.Action, true)
}

func (p *ClientRegistrations_t) GetActiveActionInfo(name string) *ActiveActionInfo_t {
    if r, ok := p.ActiveActions[action.Action]; ok {
        /* return a new copy */
        return &ActiveActionInfo_t{*r}
    } else {
        return nil
    }
}


func (p *ClientRegistrations_t) DeregisterClient(name string) {
    if cl, ok := p.ActiveClients[name]; !ok {
        return
    }

    /*
     * Delete client first tpo avoid removing one action at a time, during
     * deregister of its actions.
     */
    cl.Close()
    delete (p.ActiveClients, name)

    for k, _ := range cl.Actions {
        p.DeregisterAction(k)
    }
}

func (p *ClientRegistrations_t) DeregisterAction(actName string) {
    if r, ok := p.ActiveActions[actName]; !ok {
        /* No such action */
        return
    } else if cl, ok := p.ActiveClients[r.Client]; ok {
        delete (cl.Actions, actName)
    }
    delete (p.ActiveActions, actName)
    GetSeqHandler().DropRequest(action.Action, true)
}

func (p *ClientRegistrations_t) NotifyHeartbeats(actName string,
            ts EpochSecs) {
    if r, ok := p.ActiveActions[actName]; ok {
        p.heartbeatCh <- actName
    }
}

func (p *ClientRegistrations_t) PublishHeartbeats() {
    lst := make(map[string]struct{})

    type HBData_t struct {
        Sender      string
        Actions     []string
        Timestamp   int64
    }

    for {
        /* Read inside the loop to help refresh any change */
        hb := GetConfigMgr().GetGlobalCfgInt("ENGINE_HB_INTERVAL")
        select {
        case a := <- p.heartbeatCh:
            lst[a] = struct{}{}
            /* Collect actions */

        case <- time.After(time.Duration(hb) * time.Second):
            hb := &HBData_t {
                Sender: "LoM",
                Actions: make([]string, len(lst),
                Timestamp: time.Now().Unix()
            }
            if len(lst) > 0 {
                /* Publish collected actions, which could be empty */
                i := 0
                for k, _ := range lst {
                    hb.Actions[i] = k
                    i++
                }
                /* Reset collected. */
                lst = make(map[string]struct{})
            }

            /* Publish with or w/o actions, as this is LoM Heartbeat */
            if out, err := json.Marshal(hb); err != nil {
                LogError("Internal error: Failed to marshal HB (%s) (%v)", err, hb)
            } else {
                PublishString(string(out))
            }
        }
    }
}


/* Request to be sent to client as response to client's recvServerRequest */
func (p, *ClientRegistrations_t) AddServerRequest(
            actionName string, req *ServerRequestData) error {
    if a, ok := p.ActiveActions[actName]; !ok {
        return LogError("(%s): Action is not registered yet", actionName)
    } else if cl, ok := p.ActiveClients[a.Client]; !ok {
        return LogError("Internal error: client(%s) for action (%s) not found",
                actionName, a.Client)
    } else {
        cl.pendingWriteRequests <- req
        return nil
    }
}

/* Client's request to read server request via recvServerRequest */
func (p, *ClientRegistrations_t) SendServerRequest(req *LoMRequestInt) error {
    if cl, ok := p.ActiveClients[req.Client]; !ok {
        return LogError("Internal error: client(%s) not found", req.Client)
    }
    cl.pendingReadRequests <- req
    return nil
}



