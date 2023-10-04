package lomtelemetry

import (
    cmn "lom/src/lib/lomcommon"
)

type ChannelType_t int

const (
    CHANNEL_TYPE_EVENTS ChannelType_t = iota
    CHANNEL_TYPE_COUNTERS
    CHANNEL_TYPE_REDBUTTON
    CHANNEL_TYPE_ECHO
    CHANNEL_TYPE_SCS
    CHANNEL_TYPE_TEST_REQ
)

func ToChannelType(int v) (ChannelType_t, err) {
    if v <= CHANNEL_TYPE_TEST_REQ {
        return ChannelType_t(v), nil
    } 
    return CHANNEL_TYPE_EVENTS, lom.LogError("Invalid value for chType (%d)", v)
}

var CHANNEL_TYPE_STR = map[ChannelType_t]string{
    CHANNEL_TYPE_EVENTS:    "CHANNEL_TYPE_EVENTS",
    CHANNEL_TYPE_COUNTERS:  "CHANNEL_TYPE_COUNTERS",
    CHANNEL_TYPE_REDBUTTON: "CHANNEL_TYPE_REDBUTTON",
    CHANNEL_TYPE_ECHO:      "CHANNEL_TYPE_ECHO",
    CHANNEL_TYPE_SCS:       "CHANNEL_TYPE_SCS",
    CHANNEL_TYPE_TEST_REQ:  "CHANNEL_TYPE_TEST_REQ",
}

type channelMode_t int

const (
    CHANNEL_MODE_PUBLISHER channelMode_t = iota
    CHANNEL_MODE_SUBSCRIBER
    CHANNEL_MODE_REQUEST
    CHANNEL_MODE_RESPONSE
    CHANNEL_MODE_PROXY_CTRL_PUB
    CHANNEL_MODE_PROXY_CTRL_SUB
)

type ChannelProducer_t int

const (
    CHANNEL_PRODUCER_ENGINE ChannelProducer_t = iota
    CHANNEL_PRODUCER_PLMGR
    CHANNEL_PRODUCER_PLUGIN
    CHANNEL_PRODUCER_OTHER
    CHANNEL_PRODUCER_EMPTY
)

func ToChannelPropducer(int v) (ChannelPropducer_t, err) {
    if v <= CHANNEL_PRODUCER_OTHER {
        return ChannelProducer_t(v), nil
    } 
    return CHANNEL_PRODUCER_EMPTY, lom.LogError("Invalid value for chProducer")
}

type CHANNEL_PRODUCER_DATA_t struct {
    pattern         string
    suffix_required bool
}

var CHANNEL_PRODUCER_STR = map[ChannelProducer_t]CHANNEL_PRODUCER_DATA_t{
    CHANNEL_PRODUCER_ENGINE: CHANNEL_PRODUCER_DATA_t{"Engine", false},
    CHANNEL_PRODUCER_PLMGR:  CHANNEL_PRODUCER_DATA_t{"PluginMgr/%s", true},
    CHANNEL_PRODUCER_PLUGIN: CHANNEL_PRODUCER_DATA_t{"Plugin/%s", true},
    CHANNEL_PRODUCER_OTHER:  CHANNEL_PRODUCER_DATA_t{"Other/%s", true},
    CHANNEL_PRODUCER_EMPTY:  CHANNEL_PRODUCER_DATA_t{"", false},
}

type JsonString_t string
type ClientReq_t JsonString_t
type ServerRes_t JsonString_t
type ClientRes_t struct {
    Res ServerRes_t
    Err error /* Error while processing request */
}
