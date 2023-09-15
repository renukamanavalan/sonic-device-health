package lomtelemetry


import (
)


type ChannelType_t int

const (
    CHANNEL_TYPE_EVENTS ChannelType_t = iota
    CHANNEL_TYPE_COUNTERS
    CHANNEL_TYPE_REDBUTTON
    CHANNEL_TYPE_ECHO
    CHANNEL_TYPE_SCS
    CHANNEL_TYPE_NA
)


type channelMode_t int

const (
    CHANNEL_PUBLISHER channelMode_t = iota
    CHANNEL_SUBSCRIBER
    CHANNEL_REQUEST
    CHANNEL_RESPONSE
    CHANNEL_PROXY_CTRL_PUB
    CHANNEL_PROXY_CTRL_SUB
)

type ChannelProducer_t int
const (
    CHANNEL_PRODUCER_ENGINE ChannelPublisher_t = iota
    CHANNEL_PRODUCER_PLMGR
    CHANNEL_PRODUCER_PLUGIN
)

const (
    CHANNEL_PRODUCER_STR_ENGINE = "Engine"
    CHANNEL_PRODUCER_STR_PLMGR  = "PluginMg
    CHANNEL_PRODUCER_STR_PLUGIN = "Plugin/%s"
)


type JsonString_t string
type ClientReq_t JsonString_t 
type ServerRes_t JsonString_t 
type ClientRes_t struct {
    res ServerRes_t 
    err error   /* Error while processing request */
}


