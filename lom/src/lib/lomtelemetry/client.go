package lomtelemetry


import (
    cmn "lom/src/lib/lomcommon"
)


func getTopic(chProducer ChannelProducer_t, pluginName string) (string, error) {
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

    default:
        return nil, cmn.LogError("Unknown channel producer(%v)", chProducer)
    }
}

/*
 * GetPubChannel
 *
 * Get channel for publishing across processes.
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
 *                Expect a JSON string
 *  err - Any error
 *
 */
func GetPubChannel(chtype ChannelType_t, producer ChannelProducer_t,
        pluginName string) (ch chan<- JsonString_t, err error) {

    defer func() {
        if err != nil {
            close(ch)
            ch = nil
        }
    }()

    ch = make(chan JsonString_t)
    prefix = ""
    if prefix, err = getTopic(producer, pluginName); err != nil {
        /* err is detailed enough */
    } else if err = return openChannel(CHANNEL_PUBLISHER, chtype, prefix, ch); err != nil {
        err = cmn.LogError("Failed to get pub channel (%v)", err)
    }
    return
}


/*
 * GetSubChannel
 *
 * Get channel for subscribing from other processes.
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
        pluginName string) (ch <-chan JsonString_t, err error) {
    defer func() {
        if err != nil {
            close(ch)
            ch = nil
        }
    }()

    ch = make(chan JsonString_t)
    prefix = ""
    if prefix, err = getTopic(receiveFrom, pluginName); err == nil {
        if err := openChannel(CHANNEL_SUBSCRIBER, chtype, prefix, ch); err != nil {
            err = cmn.LogError("Failed to get sub channel (%v)", e)
    }
    return 
}



/*
 * A proxy to bind publishers & subscribers
 * 
 * The proxy enables loose coupling of publishers & subscribers.
 * Publishers & subscribers may start anytime and unware of other 
 * publishers & subscribers.
 * This also means that until this proxy is started all publish data are dropped.
 *
 * This routine's sole job to connect publishers & subscribers in full
 * mesh. This has no special business logic.
 *
 * The main subscriber can choose to run this proxy. Only one instance
 * can be run. Any subsequent requests would fail.
 *
 * Each channel type has its own independent channels, hence need a
 * separate proxy for each.
 *
 * It gets shutdown only upon system shutdown.
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
func RunPubSubProxy(chType ChannelType_t) error {
    return runPubSubProxy(chType)
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
func SendClientRequest(reqType ChannelType_t, req ClientReq_t) (ch <-chan *ClientRes_t, err error) {

    ch = make(chan *ClientRes_t)
    if e := processRequest(req, ch); e != nil {
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

func RegisterServerReqHandler(reqType ChannelType_t) (chReq <-chan ClientReq_t, 
            chRes chan<- ServerRes_t, err error) {
    chReq = make(chan ClientReq_t)
    chRes = make(chan *ServerRes_t)
    defer func() {
        if err != nil {
            close(chReq)
            close(chRes)
            chReq = nil
            chRes = nil
        }
    }()
    return initServerRequestHandler(reqType, chReq, chRes)
}
