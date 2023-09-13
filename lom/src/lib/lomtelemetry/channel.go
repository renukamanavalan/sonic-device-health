package lomtelemetry


import (
)


type ChannelType_t int

const (
    CHANNEL_TYPE_EVENTS ChannelType_t = iota
    CHANNEL_TYPE_COUNTERS
    CHANNEL_TYPE_REDBUTTON
    CHANNEL_TYPE_NA
)


type channelMode_t int

const (
    CHANNEL_PUBLISHER channelMode_t = iota,
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
    CHANNEL_PRODUCER_ANY
)

const (
    CHANNEL_PRODUCER_STR_ENGINE = "Engine"
    CHANNEL_PRODUCER_STR_PLMGR  = "PluginMg
    CHANNEL_PRODUCER_STR_PLUGIN = "Plugin/%s"
    CHANNEL_PRODUCER_STR_ANY = ""
)


type ChannelReqType_t string
const (
    CHANNEL_REQ_ECHO = "ECHO"
    CHANNEL_REQ_SCS  = "SCS"
)

type ClientReq_t struct {
    ReqType ChannelReqType_t
    Data    any
}

type ClientRes_t struct {
    Req     channelReqResData_t
    ResCode int
    ResStr  string
    Res     channelReqResData_t
}

type ServerReq_t struct {
    Id      int     // A running identifier to match req & response
    req     ClientReq_t
}

type ServerRes_t struct {
    Id      int     // A running identifier to match req & response
    res     ClientRes_t
}


