package lomtelemetry


import (
    cmn "lom/src/lib/lomcommon"
)


func getTopic(producer ChannelProducer_t, pluginName string) (string, error) {
    switch chProducer {
    case CHANNEL_PRODUCER_ENGINE:
        return CHANNEL_PRODUCER_STR_ENGINE, nil

    case CHANNEL_PRODUCER_PLMGR:
        return CHANNEL_PRODUCER_STR_PLMGR, nil

    case CHANNEL_PRODUCER_PLUGIN:
        if pluginName == "" {
            return nil, cmn.LogError("Missing plugin Name")
        }
        return fmt.Sprintf(CHANNEL_PRODUCER_STR_PLUGIN, pluginName), nil

    case CHANNEL_PRODUCER_ANY:
        return CHANNEL_PRODUCER_STR_ANY

    default:
        return nil, cmn.LogError("Unknown channel producer(%v)", chProducer)
    }
}

/*
 * GetPubChannel
 *
 * Get channel for publishing.
 * The channel will be closed upon system shutdown (cmn.DoSysShutdown)
 *
 * Input:
 *  chtype      - Type of data to be published
 *  producer    - Is this engine, pluginMgr or plugin
 *  pluginName  - If plugin, your name
 *
 * Output: None
 *
 * Return:
 *  chData      - Input data channel for publishing. All data written
 *                into this channel by anyone are published.
 *  err - Any error
 *
 */
func GetPubChannel(chtype ChannelType_t, producer ChannelProducer_t,
        pluginName string) (ch chan<- string, err error) {

    defer func() {
        if err != nil {
            close(ch)
            ch = nil
        }
    }()

    ch = make(chan string)
    prefix, e:= getTopic(producer, pluginName)
    if (e != nil) || (prefix == "") {
        err = cmn.LogError("Failed to get pub prefix (%v)", e)
    } else if e := return openChannel(CHANNEL_PUBLISHER, chtype, prefix, ch); e != nil {
        err = cmn.LogError("Failed to get pub channel (%v)", e)
    }
    return
}


/*
 * GetSubChannel
 *
 * Get channel for subscribing
 *
 * Input:
 *  chtype - Type of data to receive.
 *  producer - If interested from a specific producer, specify, else receive from all.
 *  pluginName - If producer is a plugin, its name
 *
 * Output: None
 *
 * Return:
 *  chData - Channel to read data from subscription channel
 *  err - Any error
 *
 */

func GetSubChannel(chtype ChannelType_t, receiveFrom ChannelProducer_t,
        pluginName string) (ch <-chan string, err error) {
    defer func() {
        if err != nil {
            close(ch)
            ch = nil
        }
    }()

    ch = make(chan string)
    if prefix, e := getTopic(receiveFrom, pluginName); err != nil {
        err = cmn.LogError("Failed to get sub filter (%v)", e)
    } else if e := openChannel(CHANNEL_SUBSCRIBER, chtype, prefix, ch); e != nil {
        err = cmn.LogError("Failed to get sub channel (%v)", e)
    }
    return 
}



/*
 * A proxy to bind publishers & subscribers
 * This enables publishers & subscribers to be unaware of each other
 * Helps asynchronous activtion of publishers & subscribers.
 *
 * This routine's sole job to connect publishers & subscribers in full
 * mesh. Hence this is a *blocking* routine that run forever until abort
 * request arrives. This has no special business logic.
 *
 * The main subscriber can choose to run this proxy.
 *
 * Each channel type has its own channels, hence start proxy for each.
 *
 * Input:
 *  chType
 *      Different channel for different data types. Hence need a proxy
 *      per type.
 *  chAbort
 *      Provides a channel to listen to for system level abort.
 *      On any data
 *
 * Output:
 *  None
 *
 * Return:
 *  err - Non nil, in case of failure
 */
func RunPubSubProxy(chType ChannelType_t, chAbort <-chan int) error {
    runPubSubProxy(chType, chAbort)
}


/*
 * Send a request and get channel for receiving response asynchronously
 *
 * Input:
 *  req - Request to send
 *
 * Output:
 *  None
 *
 * Return:
 *  ch -    Channel to get response from. Caller to close the channel upon
 *          receiving response.
 *  err -   Non nil, in case of failure
 */
func ClientReqHandler(req *ClientReq_t) (ch <-chan *ClientRes_t, err error) {

    ch = make(chan *ClientRes_t)
    if e := ProcessRequest(req, ch); e != nil {
        err = cmn.LogError("Failed to process client req err(%v) req(+%v)", e, req)
        close(ch)
        ch = nil
    } 
    return
}


/*
 * Initializes a handler for processing requests.
 * A handler is for a specific request type.
 *
 * Input:
 *  reqType - Type of request it handles.
 *
 * Output:
 *  None
 *
 * Return:
 *  chReq - Input channel to read client requests.
 *  chRes - Output channel to write server's response
 *  err - Non nil error implies failure
 */
serverHandlers = map[ChannelReqType_t]bool

func ServerReqHandler(reqType ChannelReqType_t) (chReq <-chan *ServerReq_t, 
            chRes chan<- *ServerRes_t, err error) {
    if _, ok := serverHandlers[reqType]; ok {
        err = cmn.LogError("Already handler registered for %d", reqType)
        return
    }

    chReq = make(chan *ServerReq_t)
    chRes = make(chan *ServerRes_t)
    if err = initRequestHandler(chReq, chRes); err != nil {
        err = cmn.LogError("Failed to setup request handler for (%d) err(%v)", reqType, err)
        return
    }
    serverHandlers[reqType] = true
}
