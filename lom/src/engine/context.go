package engine

import (
    . "lom/src/lib/lomcommon"
    . "lom/src/lib/lomipc"
    "sort"
    "time"
)

/* Action info is collected from register calls from clients */
type ActiveActionInfo_t struct {
    Action      string
    Client      string    /* Client that registered this action
    /*
     * Timeout to use if not set in sequence as obtained from actions.conf 
     */
    Timeout     int
    /* For now, Engine does not need any other config, hence not saved */
}

/* Any less must be defaulted to this */
const MIN_HB_INTERVAL = 2

/*
 * CHannels that hold read requests from client and write requests from server
 * are generally drained quickly. Yet, have a buffer.
 */
const CHAN_REQ_SIZE = 10

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
     * via GetRegistrations().PendServerRequest(req). This writes the request into
     * pendingReadRequests channel. 
     *
     * Whenever a request to be raised to client (upon register action/process response/
     * ...), that sends the request into pendingWriteRequests channel, as there may or may
     * not be a read request from client pending via AddServerRequest.
     *
     * A dedicated Go routine ProcessSendRequests watches both channels and do the transfer
     * as appropriate.
     *
     * Each request from SendToServer has a request specific channel (LoMRequestInt::
     * ChResponse) for its response and return it via RPC to client's blocking
     * RecvServerRequest.
     */
    /* Pending requests per client */
    pendingWriteRequests    chan *ServerRequestData     /* Server to client */
    pendingReadRequests     chan *LoMRequestInt         /* Client's request to read req from server */

    /* To abort the go routine for ProcessSendRequests */
    abortCh chan interface{}
}

func (p *ActiveClientInfo_t) Close() {
    p.abortCh <- struct{}{}
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
    /* 5 & 100 are just initial capacity. this does not block scaling up. */
    listWTimeout := make([]*wait_t, 0, 5)   /* Hold client requests for read sorted by due */
    listNoTimeout := make([]*wait_t, 0, 5)  /* Hold client requests for read */
    serverRequests := make([]*ServerRequestData, 0, 100)
    tout := A_DAY_IN_SECS

    for {
        select {
        case clReq := <- p.pendingReadRequests:
            w := &wait_t { req: clReq }
            if clReq.Req.TimeoutSecs == 0 {
                listNoTimeout = append(listNoTimeout, w)
            } else {
                w.due = time.Now().Unix() + int64(clReq.Req.TimeoutSecs)
                listWTimeout = append(listWTimeout, w)
                sort.Slice(listWTimeout, func(i, j int) bool {
                    return listWTimeout[i].due < listWTimeout[j].due
                })
            }

        case serReq := <- p.pendingWriteRequests:
            serverRequests = append(serverRequests, serReq)

        case <- time.After(time.Duration(tout) * time.Second):
            /* bail out */

        case <- p.abortCh:
            LogInfo("Aborting Send requests")
            return
        }
        
        /* Here you come on client/server request or timeout */
        for len(serverRequests) > 0 {
            var r *LoMRequestInt = nil
            if len(listWTimeout) > 0 {
                r = listWTimeout[0].req
                listWTimeout = listWTimeout[1:]
            } else if len(listNoTimeout) > 0 {
                r = listNoTimeout[0].req
                listNoTimeout = listNoTimeout[1:]
            }
            if r != nil {
                r.ChResponse <- &LoMResponse { 0, "", serverRequests[0] }
                serverRequests = serverRequests[1:]
            } else {
                break
            }
        }

        tnow := time.Now().Unix()
        tout = A_DAY_IN_SECS
        for i := 0; i < len(listWTimeout); i++ {
            if tnow >= listWTimeout[i].due {
                /* Fail the request */
                r := listWTimeout[i].req
                r.ChResponse <- &LoMResponse { int(LoMReqTimeout),
                                GetLoMResponseStr(LoMReqTimeout), nil }
            } else {
                /* Get tout and drop the failed ones */
                tout = 0
                if tnow < listWTimeout[i].due {
                    tout = listWTimeout[i].due - tnow
                }
                listWTimeout = listWTimeout[i:]
                break
            }
        }

    }
}


/*
 * A go func watches, yet it may get blocked by actions like publish
 * To ensure, notify APIs are not blocked, have a buffer.
 * Publish API is generally pretty quick, a value of 10 is more than enough.
 */
const HEARTBEAT_CH_SIZE = 10

/* All registrations & heartbeats from clients. */
type ClientRegistrations_t struct {
    activeActions   map[string]*ActiveActionInfo_t
    activeClients   map[string]*ActiveClientInfo_t
    heartbeatCh     chan string     /* Action name */
}

/* Initialized object; But not exported */
var clientRegistrations *ClientRegistrations_t = nil


func GetRegistrations() *ClientRegistrations_t {
    if clientRegistrations == nil {
        InitRegistrations()
    }
    return clientRegistrations
}


func InitRegistrations() *ClientRegistrations_t {
    clientRegistrations = &ClientRegistrations_t{
                make(map[string]*ActiveActionInfo_t),
                make(map[string]*ActiveClientInfo_t),
                make(chan string, HEARTBEAT_CH_SIZE),
            }
    go clientRegistrations.PublishHeartbeats()
    return clientRegistrations
}

func (p *ClientRegistrations_t) RegisterClient(name string) error {
    if len(name) == 0 {
        return LogError("Expect non empty name")
    }
    if _, ok := p.activeClients[name]; ok {
        LogInfo("%s: Duplicate client registration; De & re-register", name)
        p.DeregisterClient(name)
    }
    cl := &ActiveClientInfo_t {
        ClientName: name,
        Actions: make(map[string]struct{}),
        pendingWriteRequests: make(chan *ServerRequestData, CHAN_REQ_SIZE),
        pendingReadRequests: make(chan *LoMRequestInt, CHAN_REQ_SIZE),
        abortCh: make(chan interface{}, 2) }
                
    p.activeClients[name] = cl
    go cl.ProcessSendRequests()
    return nil
}

func (p *ClientRegistrations_t) RegisterAction(action *ActiveActionInfo_t) error {
    if action == nil {
        return LogError("Expect non nil ActiveActionInfo_t")
    }
    cl, ok := p.activeClients[action.Client]
    if !ok {
        return LogError("%s: Missing client registration", action.Client)
    } else if r, ok1 := p.activeActions[action.Action]; ok1 {
        LogInfo("%s/%s: Duplicate action registration (%s); De/re-register,",
                action.Client, action.Action, r.Client)
        p.DeregisterAction("", action.Action)
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
        info := &ActiveActionInfo_t{}
        *info = *action
        p.activeActions[action.Action] = info
        GetSeqHandler().RaiseRequest(action.Action)
        return nil
    }
}

func (p *ClientRegistrations_t) GetActiveActionInfo(name string) *ActiveActionInfo_t {
    if r, ok := p.activeActions[name]; ok {
        /* return a new copy */
        info := &ActiveActionInfo_t{}
        *info = *r
        return info
    } else {
        return nil
    }
}


func (p *ClientRegistrations_t) DeregisterClient(name string) {
    if len(name) == 0 {
        LogError("Expect non empty name")
    } else if cl, ok := p.activeClients[name]; ok {
        /*
         * Delete client first to avoid removing one action at a time, during
         * deregister of its actions.
         */
        cl.Close()
        delete (p.activeClients, name)

        for k, _ := range cl.Actions {
            p.DeregisterAction(name, k)
        }
    }
}

func (p *ClientRegistrations_t) DeregisterAction(clName, actName string) {
    if len(actName) == 0 {
        LogError("Expect non empty name")
    } else if r, ok := p.activeActions[actName]; ok {
        if (len(clName) != 0) && (clName != r.Client) {
            LogError("Skip de-register as action(%s) client(%s) != given(%s)",
                    actName, r.Client, clName)
        } else {
            if cl, ok := p.activeClients[r.Client]; ok {
                delete (cl.Actions, actName)
            }
            delete (p.activeActions, actName)
            GetSeqHandler().DropRequest(actName)
        }
    }
}

func (p *ClientRegistrations_t) NotifyHeartbeats(actName string,
            ts int64) {
    if _, ok := p.activeActions[actName]; ok {
        p.heartbeatCh <- actName
    }
}

type HB_t struct {
    Actions     []string
    Timestamp   int64
}

type HBData_t struct {
    LoM_Heartbeat HB_t
}


func (p *ClientRegistrations_t) PublishHeartbeats() {
    /* Map will help combine duplicate heartbeats into one */
    lst := make(map[string]struct{})

    for {
        /* Read inside the loop to help refresh any change */
        hb := GetConfigMgr().GetGlobalCfgInt("ENGINE_HB_INTERVAL")
        if hb == 0 {
            hb = MIN_HB_INTERVAL
        }
        select {
        case a := <- p.heartbeatCh:
            lst[a] = struct{}{}
            /* Collect actions */

        case <- time.After(time.Duration(hb) * time.Second):
            hb := &HBData_t { HB_t { make([]string, len(lst)), time.Now().Unix()}}

            if len(lst) > 0 {
                /* Publish collected actions, which could be empty */
                i := 0
                for k, _ := range lst {
                    hb.LoM_Heartbeat.Actions[i] = k
                    i++
                }
                /* Reset collected. */
                lst = make(map[string]struct{})
            }
            PublishEvent(hb)
        }
    }
}


/* Request to be sent to client as response to client's recvServerRequest */
func (p *ClientRegistrations_t) AddServerRequest(
            actionName string, req *ServerRequestData) error {
    if (len(actionName) == 0) || (req == nil) {
        return LogError("Internal error: Nil args (%v) (%v)", actionName, req)
    } else if a, ok := p.activeActions[actionName]; !ok {
        return LogError("(%s): Action is not registered yet", actionName)
    } else if cl, ok := p.activeClients[a.Client]; !ok {
        return LogError("Internal error: client(%s) for action (%s) not found",
                a.Client, actionName)
    } else {
        if len(cl.pendingWriteRequests) >= cap(cl.pendingWriteRequests) {
            return LogError("Internal error: pendingWriteRequests is full (%d)",
                    len(cl.pendingWriteRequests))
        }
        cl.pendingWriteRequests <- req
        return nil
    }
}

/* Client's request to read server request via recvServerRequest */
func (p *ClientRegistrations_t) PendServerRequest(req *LoMRequestInt) error {
    if req == nil {
        return LogError("Internal error: Nil req")
    } else if cl, ok := p.activeClients[req.Req.Client]; !ok {
        return LogError("Internal error: client(%s) not found", req.Req.Client)
    } else if len(cl.pendingReadRequests) >= cap(cl.pendingReadRequests) {
        return LogError("Internal error: pendingReadRequests is full (%d)",
                len(cl.pendingReadRequests))
    } else {
        cl.pendingReadRequests <- req
        return nil
    }
}

