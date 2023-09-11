package lomtelemetry


import (
    cmn "lom/src/lib/lomcommon"
)


func getTopic(producer ChannelProducer, pluginName string) (string, error) {
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
 * Get channel for publishing
 *
 * Input:
 *  chtype - Type of data to be published
 *  producer - Is this engine, pluginMgr or plugin
 *  pluginName - If plugin, your name
 *
 * Output: None
 *
 * Return:
 *  chTelemetry - A receiver for subsequent write
 *  err - Any error
 *
 */
func GetPubChannel(chtype ChannelType, producer ChannelProducer,
        pluginName string) (any, error) {
    prefix, err := getPrefix(producer, pluginName)
    if (err != nil) || (prefix == "") {
        return nil, cmn.LogError("Failed to get pub prefix (%v)", err)
    } else if ch, err := return getHandle(CHANNEL_PUBLISHER, chtype, prefix); err != nil {
        return nil, cmn.LogError("Failed to get pub channel (%v)", err)
    }
    return ch, nil
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
 *  chTelemetry - A receiver for subsequent read
 *  err - Any error
 *
 */
func GetSubChannel(chtype ChannelType, receiveFrom ChannelProducer,
        pluginName string) (any, error) {
    if prefix, err := getPrefix(receiveFrom, pluginName); err != nil {
        return nil, cmn.LogError("Failed to get sub filter (%v)", err)
    } else if ch, err := getChannel(CHANNEL_SUBSCRIBER, chtype, prefix); err != nil {
        return nil, cmn.LogError("Failed to get sub channel (%v)", err)
    }
    return ch, nil
}


/*
 * This could be used to send request and receive response with timeout.
 * Requests types are defined and data format is provided.
 *
 * Request types can be Echo, SCS, ...
 */
func GetReqChannel() (any, err) {
    /* Req/rep is for any supported request type, hence no specific channel type */
    return getChannel(CHANNEL_REQUEST, CHANNEL_TYPE_END, "")
}


/*
 * This could be used to receive request and send response within timeout.
 * Requests types are defined and data format is provided.
 *
 * TODO
 * Request types can be Echo, SCS, ...
 */
func GetResChannel() (any, err) {
    return getChannel(CHANNEL_RESPONSE, CHANNEL_TYPE_REQ_REP, "")
    return nil, LogError("Not Implemented yet")
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
 *  None
 */
func RunPubSubProxy(chType ChannelType, chAbort <-chan int) error {
    runPubSubProxy(chType, chAbort)
}


func WriteChannel(handle any, data any) error {
    buff, err := json.Marshal(v)
    if err != nil {
        return cmn.LogError("Failed to marshal err(%v)  v(%+v)", err, v)
    }
    return WriteHandle(handle, string(buff))
}


func ReadChannel(handle any, data any) error {
    dataStr, err := ReadHandle(handle)
    if err != nil {
        return nil
    }
    if err = json.Unmarshal([]byte(dataStr), data); err != nil {
        return cmn.LogError("Failed to unmarshal err(%v)  v(%+v)", err, data[index])
    }
    return nil
}


