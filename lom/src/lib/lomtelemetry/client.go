package lomtelemetry

import (
    "fmt"
    cmn "lom/src/lib/lomcommon"
)

func getTopic(chProducer ChannelProducer_t, suffix string) (topic string, err error) {
    data, ok := CHANNEL_PRODUCER_STR[chProducer]
    if !ok || (data.suffix_required && (suffix == "")) {
        err = cmn.LogError("producer match(%v) - ok(%v) or missing suffix",
                    chProducer, ok)
    } else if !data.suffix_required {
        topic = data.pattern
    } else {
        topic = fmt.Sprintf(data.pattern, suffix)
    }
    return
}

/*
 * GetPubChannel
 *
 * Get channel for publishing events, counters, red-button, ...
 * Once opened it stays till system shutdown or terminates upon
 * i/p data channel close.
 * NOTE: The Pub channel can be opened only once per process. Any
 *       pre-mature termination will block any further publish.
 *
 * Closing returned channel chData will close the underlying
 * network connection.
 *
 * Input:
 *  chtype      - Type of data to be published
 *  producer    - Is this engine, pluginMgr or plugin
 *  suffix      - Producer suffix. Say plugin-name in case of plugin.
 *
 * Output: None
 *
 * Return:
 *  chData      - Input data channel for publishing. All data written
 *                into this channel by anyone are published.
 *                Expect a JSON string.
 *                Closing this shuts down this pubchannel
 *  err - Any error
 *
 */
func GetPubChannel(chtype ChannelType_t, producer ChannelProducer_t,
    pluginName string) (chData chan<- JsonString_t, err error) {

    ch := make(chan JsonString_t)

    defer func() {
        if err != nil {
            close(ch)
            ch = nil
        }
    }()

    prefix := ""
    if prefix, err = getTopic(producer, pluginName); err != nil {
        /* err is detailed enough */
    } else if err = openPubChannel(chtype, prefix, ch); err != nil {
        err = cmn.LogError("Failed to get pub channel (%v)", err)
    } else {
        chData = ch
    }
    return
}

/*
 * GetSubChannel
 *
 * Get channel for subscribing from other processes.
 * Runs until system shutdown.
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
 *  chClose - Close this channel to close the underlying subscriber network connection.
 *  err - Any error
 *
 */

func GetSubChannel(chtype ChannelType_t, receiveFrom ChannelProducer_t,
    pluginName string) (chData <-chan JsonString_t, chClose chan<- int, err error) {

    ch := make(chan JsonString_t)
    chCl := make(chan int)

    defer func() {
        if err != nil {
            close(ch)
            close(chCl)
        }
    }()

    prefix := ""
    if prefix, err = getTopic(receiveFrom, pluginName); err == nil {
        if err = openSubChannel(chtype, prefix, ch, chCl); err != nil {
            err = cmn.LogError("Failed to get sub channel (%v)", err)
        } else {
            chData = ch
            chClose = chCl
            cmn.LogDebug("CHANNEL_MODE_SUBSCRIBER created for chtype=%d(%s)",
                chtype, CHANNEL_TYPE_STR[chtype])
        }
    }
    return
}

/*
 * A proxy to bind publishers & subscribers
 *
 * The proxy enables loose coupling of publishers & subscribers.
 * Publishers & subscribers may start anytime and can be unware of each other.
 * This also means that only upon this proxy is started, publishers' data
 * will reach corresponding subscribers.
 *
 * This routine's sole job is to connect publishers & subscribers in full
 * mesh. This has no special business logic.
 *
 * Any one main process can choose to run this proxy. Only one instance
 * can be run. Any subsequent requests would fail.
 *
 * As each channel type has its own independent channels, an independent proxy
 * is needed per channel type.
 *
 * It gets shutdown only upon system shutdown.
 *
 * Input:
 *  chType
 *      Different channel for different data types. Hence need a proxy
 *      per type.
 *
 * Output:
 *  None
 *
 * Return:
 *  chClose - Close this channel to stop the proxy
 *  err - Non nil, in case of failure
 *
 */
func RunPubSubProxy(chType ChannelType_t) (chClose chan<- int, err error) {
    chCl := make(chan int)
    err = doRunPubSubProxy(chType, chCl)
    if err == nil {
        chClose = chCl
    }
    return
}

/*
 * Send a request and get channel for receiving response asynchronously
 *
 * Input:
 *  req -   Request to send
 *          An empty request will shutdown request handler running in background.
 *          Send empty request for each channel type to ensure closed.
 *
 * Output:
 *  None
 *
 * Return:
 *  chData -    Channel to get response from. Caller to close the channel upon
 *          receiving response.
 *  err -   Non nil, in case of failure
 */
func SendClientRequest(reqType ChannelType_t, req ClientReq_t) (chData <-chan *ClientRes_t, err error) {

    ch := make(chan *ClientRes_t)
    if e := processRequest(reqType, req, ch); e != nil {
        err = cmn.LogError("Failed to process client req err(%v) req(+%v)", e, req)
        close(ch)
        ch = nil
    } else {
        chData = ch
    }
    return
}


func CloseClientRequest(reqType ChannelType_t) error {
    return closeRequestChannel(reqType)
}

/*
 * Initializes a handler for processing requests.
 * A handler is for a specific request type.
 * Only one handler per request can be registered.
 * Any proc may choose to register a handler.
 *
 * Close returned channel chRes to shut this handler.
 *
 * Input:
 *  reqType - Type of request it handles.
 *
 * Output:
 *  None
 *
 * Return:
 *  chDataReq - Input channel to read client requests.
 *  chRDataes - Output channel to write server's response
 *  err - Non nil error implies failure
 */

func RegisterServerReqHandler(reqType ChannelType_t) (chDataReq <-chan ClientReq_t,
    chDataRes chan<- ServerRes_t, err error) {
    chReq := make(chan ClientReq_t)
    chRes := make(chan ServerRes_t)

    if err = initServerRequestHandler(reqType, chReq, chRes); err != nil {
        /* initServerRequestHandler closes chReq */
        close(chRes)
    } else {
        chDataReq = chReq
        chDataRes = chRes
    }
    return
}

func IsTelemetryIdle() bool {
    return isZMQIdle()
}
