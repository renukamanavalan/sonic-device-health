package lomtelemetry


import (
    cmn "lom/src/lib/lomcommon"
    "encoding/json"
)


type ChannelType int

const (
    CHANNEL_TYPE_EVENTS ChannelType = iota
    CHANNEL_TYPE_COUNTERS
    CHANNEL_TYPE_REDBUTTON
    CHANNEL_TYPE_END
)


type channelMode int

const (
    CHANNEL_PUBLISHER channelMode = iota,
    CHANNEL_SUBSCRIBER
    CHANNEL_REQUEST
    CHANNEL_RESPONSE
)

type ChannelProducer int
const (
    CHANNEL_PRODUCER_ENGINE ChannelPublisher = iota
    CHANNEL_PRODUCER_PLMGR
    CHANNEL_PRODUCER_PLUGIN
    CHANNEL_PRODUCER_ANY
)

type ChannelPublisherStr string
const (
    CHANNEL_PRODUCER_STR_ENGINE = "Engine"
    CHANNEL_PRODUCER_STR_PLMGR  = "PluginMg
    CHANNEL_PRODUCER_STR_PLUGIN = "Plugin/%s"
    CHANNEL_PRODUCER_STR_ANY = ""
)


type ChTelemetry struct {
    chMode      channelMode
    chType      ChannelType
    prefix      string      /* Prefixed in writing or used as filter in reader */
    handle      any
}


