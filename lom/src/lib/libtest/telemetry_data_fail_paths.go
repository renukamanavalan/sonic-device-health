package libtest

import (
    "errors"
    . "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

var pubSubFailSuite = ScriptSuite_t{
    Id:          "pubSubFailSuite",
    Description: "For corner & failure cases",
    Entries: []ScriptEntry_t{
        ScriptEntry_t{
            ApiIDGetPubChannel, /* Try publish to fail */
            []Param_t{
                Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}, /* pub for counters */
                Param_t{"prod_PM", tele.CHANNEL_PRODUCER_PLMGR, nil}, /* from Plugin Mgr */
                Param_t{"PMgr-1", "", nil},                           /* missing suffix */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Failed to get Pub channel for missing suffix",
        },
        ScriptEntry_t{
            ApiIDGetPubChannel, /* Get a valid pub channel */
            []Param_t{
                Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}, /* pub for counters */
                Param_t{"prod_PM", tele.CHANNEL_PRODUCER_PLMGR, nil}, /* from Plugin Mgr */
                Param_t{"PMgr-1", "test", nil},                       /* non-empty suffix */
            },
            []Result_t{
                Result_t{"chWrite-0", nil, ValidateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get a channel",
        },
        ScriptEntry_t{
            ApiIDIsTelemetryIdle,
            []Param_t{},
            []Result_t{TEST_FOR_FALSE, NIL_ERROR},
            "Check for non idle",
        },
        ScriptEntry_t{
            ApiIDGetPubChannel, /* Try again for same type */
            []Param_t{
                Param_t{"chType_C", nil, nil}, /* Get from cache*/
                Param_t{"prod_PM", nil, nil},  /* from cache */
                Param_t{"PMgr-1", nil, nil},   /* Same as above */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Fail as duplicate channel",
        },
        ScriptEntry_t{ /* Get sub channel for events from engine only. */
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil},
                Param_t{"prod_E", tele.CHANNEL_PRODUCER_ENGINE, nil},
                EMPTY_STRING,
            },
            []Result_t{
                Result_t{"chRead-E", nil, ValidateNonNil},     /* Save in cache */
                Result_t{"chSubClose-E", nil, ValidateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get sub channel for events from Engine",
        },
        ScriptEntry_t{ /* Get sub channel for events from engine only. */
            ApiIDGetSubChannel, /* Get Sub channel for same params as above */
            []Param_t{
                Param_t{"chType_E", nil, nil},
                Param_t{"prod_E", nil, nil},
                EMPTY_STRING,
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil on failure */
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil on failure */
                NON_NIL_ERROR,
            },
            "Get sub channel for same type to fail",
        },
        ScriptEntry_t{
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chWrite-0", nil, nil},    /* Close pub chan opened above */
                Param_t{"chSubClose-E", nil, nil}, /* Close sub chan opened above */
            },
            []Result_t{NIL_ERROR},
            "Close channels",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest, /* Request for incorrect type */
            []Param_t{
                Param_t{"chType_1", tele.CHANNEL_TYPE_EVENTS, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request-1:Hello world"), nil},
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* chan to read response */
                NON_NIL_ERROR,
            },
            "Expect request to fail due to incorrect request type",
        },
        ScriptEntry_t{
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_EVENTS, nil}},
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* No chan on error */
                Result_t{ANONYMOUS, nil, ValidateNil}, /* No chan on error */
                NON_NIL_ERROR, /*Expect non nil error */
            },
            "Expect request to fail due to incorrect request type",
        },
        PAUSE2,
        TELE_IDLE_CHECK,
    },
}

var chTmpRes <-chan *tele.ClientRes_t
var chTmpReq <-chan *tele.ClientReq_t

var scriptAPIValidate = ScriptSuite_t{
    Id:          "ScriptAPIValidation", /* All the below are for failure only */
    Description: "For corner & failure cases",
    Entries: []ScriptEntry_t{
        ScriptEntry_t{
            ApiIDGetPubChannel,
            []Param_t{
                Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}, /* pub for counters */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        ScriptEntry_t{
            ApiIDGetPubChannel, /* Get a valid pub channel */
            []Param_t{
                Param_t{"prod_PM", tele.CHANNEL_PRODUCER_PLMGR, nil}, /* from Plugin Mgr */
                Param_t{"PMgr-1", "test", nil},                       /* non-empty suffix */
                Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}, /* pub for counters */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect first arg type",
        },
        ScriptEntry_t{
            ApiIDGetPubChannel, /* Get a valid pub channel */
            []Param_t{
                Param_t{ANONYMOUS, tele.CHANNEL_TYPE_ECHO, nil}, /* incorrect val*/
                Param_t{"prod_PM", tele.CHANNEL_PRODUCER_PLMGR, nil},   /* from Plugin Mgr */
                Param_t{"PMgr-1", "test", nil},                         /* non-empty suffix */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect first arg value",
        },
        ScriptEntry_t{
            ApiIDGetPubChannel, /* Get a valid pub channel */
            []Param_t{
                Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}, /* pub for counters */
                Param_t{"prod_PM", true, nil},                        /* from Plugin Mgr */
                Param_t{"PMgr-1", "test", nil},                       /* non-empty suffix */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect second arg",
        },
        ScriptEntry_t{
            ApiIDGetPubChannel,
            []Param_t{
                Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}, /* pub for counters */
                Param_t{"prod_PM", tele.CHANNEL_PRODUCER_PLMGR, nil}, /* from Plugin Mgr */
                Param_t{"PMgr-1", 11, nil},                           /* incorrect type suffix */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect type for third arg",
        },
        ScriptEntry_t{
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil},
                EMPTY_STRING,
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        ScriptEntry_t{
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{ANONYMOUS, tele.CHANNEL_TYPE_ECHO, nil},
                Param_t{"prod_E", tele.CHANNEL_PRODUCER_ENGINE, nil},
                EMPTY_STRING,
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect first arg value",
        },
        ScriptEntry_t{
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{"prod_E", tele.CHANNEL_PRODUCER_ENGINE, nil},
                Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil},
                EMPTY_STRING,
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect first arg type",
        },
        ScriptEntry_t{
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil},
                EMPTY_STRING,
                Param_t{"prod_E", tele.CHANNEL_PRODUCER_ENGINE, nil},
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect second arg",
        },
        ScriptEntry_t{
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil},
                Param_t{"prod_E", tele.CHANNEL_PRODUCER_ENGINE, nil},
                Param_t{ANONYMOUS, true, nil},
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect third arg",
        },
        ScriptEntry_t{
            ApiIDRunPubSubProxy,
            []Param_t{},
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        ScriptEntry_t{
            ApiIDRunPubSubProxy,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect chtype",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"req_0", tele.ClientReq_t("request-2:Hello world"), nil},
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect first arg",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", tele.CHANNEL_TYPE_EVENTS, nil},
                Param_t{"req_0", 11, nil},
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect second arg",
        },
        ScriptEntry_t{
            ApiIDReadClientResponse, /* Client read its response */
            []Param_t{
                Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        ScriptEntry_t{
            ApiIDReadClientResponse, /* Client read its response */
            []Param_t{
                Param_t{ANONYMOUS, 1, nil},  /* timeout = 1 second */
                Param_t{"chClientRes-0", nil, nil}, /* Get chan from cache */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect first arg",
        },
        ScriptEntry_t{
            ApiIDReadClientResponse, /* Client read its response */
            []Param_t{
                Param_t{ANONYMOUS, chTmpRes, nil}, /* Get chan from cache */
                Param_t{ANONYMOUS, 1, nil},        /* timeout = 1 second */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "nil first arg",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest, /* Good one to get channel */
            []Param_t{
                Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil},
                Param_t{"req_0", tele.ClientReq_t("request-3:Hello world"), nil},
            },
            []Result_t{
                Result_t{"chClientRes-0", nil, ValidateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "Send a request as if from client",
        },
        ScriptEntry_t{
            ApiIDReadClientResponse, /* Client read its response */
            []Param_t{
                Param_t{"chClientRes-0", nil, nil},     /* Get chan from cache */
                Param_t{ANONYMOUS, "rere", nil}, /* timeout not int */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "incorrect second arg timeout rere",
        },
        ScriptEntry_t{
            ApiIDReadClientResponse, /* Client read its response */
            []Param_t{
                Param_t{"chClientRes-0", nil, nil}, /* Get chan from cache */
                Param_t{ANONYMOUS, 1, nil},  /* timeout 1 sec */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Read response times out - 1",
        },
        ScriptEntry_t{
            ApiIDCloseRequestChannel, /* explicit request to close for req channel */
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
            },
            []Result_t{
                NIL_ERROR,
            },
            "Close channel created for client requests.",
        },
        ScriptEntry_t{
            ApiIDCloseRequestChannel, /* explicit request to close for req channel */
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
            },
            []Result_t{NON_NIL_ERROR},
            "Duplicate Close to fail",
        },
        PAUSE2, /* Allow req channel to close */
        ScriptEntry_t{
            ApiIDReadClientResponse, /* Client read its response */
            []Param_t{
                Param_t{"chClientRes-0", nil, nil}, /* Get chan from cache */
                Param_t{ANONYMOUS, 1, nil},  /* timeout 1 sec */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Read from closed chan for response",
        },
        ScriptEntry_t{
            ApiIDReadClientRequest,
            []Param_t{
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        ScriptEntry_t{
            ApiIDReadClientRequest,
            []Param_t{
                Param_t{ANONYMOUS, 1, nil},
                Param_t{ANONYMOUS, nil, nil},
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect first arg",
        },
        ScriptEntry_t{
            ApiIDReadClientRequest,
            []Param_t{
                Param_t{ANONYMOUS, chTmpReq, nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Nil first arg",
        },
        ScriptEntry_t{ /* Get a proper one to get valid handle for use */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []Result_t{
                Result_t{"chSerReq-0", nil, ValidateNonNil}, /* chan for incoming req */
                Result_t{"chSerRes-0", nil, ValidateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Register server handler to process requests and provide responses.",
        },
        ScriptEntry_t{
            ApiIDReadClientRequest,
            []Param_t{
                Param_t{"chSerReq-0", nil, nil},      /* Get chRead_0 from cache */
                Param_t{ANONYMOUS, true, nil}, /* Invalid timeout */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Fail with invalid timeout",
        },
        ScriptEntry_t{
            ApiIDReadClientRequest,
            []Param_t{
                Param_t{"chSerReq-0", nil, nil},   /* Get chRead_0 from cache */
                Param_t{ANONYMOUS, 1, nil}, /* valid timeout */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* no result */
                NON_NIL_ERROR,
            },
            "Read fails with timeout once",
        },
        ScriptEntry_t{
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Close server handler */
            },
            []Result_t{NIL_ERROR},
            "Close server request handler via closing this channel.",
        },
        PAUSE2,
        ScriptEntry_t{
            ApiIDReadClientRequest,
            []Param_t{
                Param_t{"chSerReq-0", nil, nil},   /* Get from cache */
                Param_t{ANONYMOUS, 1, nil}, /* valid timeout */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil}, /* Read fails with timeout */
                NON_NIL_ERROR,
            },
            "Read fails due to closed socket",
        },
        TELE_IDLE_CHECK,
    },
}

var chSerResNil <-chan tele.ServerRes_t
var chSerResNonNil = make(chan tele.ServerRes_t)
var chSerResWrNonNil chan<- tele.ServerRes_t = chSerResNonNil
var chNilJson chan<- tele.JsonString_t
var chNonNilJson = make(chan tele.JsonString_t)
var chWrNonNilJson chan<- tele.JsonString_t = chNonNilJson
var chRdNonNilJson <-chan tele.JsonString_t = chNonNilJson

func getFail(name string, val any) (any, error) {
    return nil, errors.New("Simulated error for test")
}

var scriptAPIValidate_2 = ScriptSuite_t{
    Id:          "ScriptAPIValidation-2", /* All the below are for failure only */
    Description: "For corner & failure cases",
    Entries: []ScriptEntry_t{
        ScriptEntry_t{
            ApiIDSendClientResponse,
            []Param_t{
                Param_t{ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NON_NIL_ERROR},
            "fewer args",
        },
        ScriptEntry_t{
            ApiIDSendClientResponse,
            []Param_t{
                Param_t{ANONYMOUS, true, nil},
                Param_t{ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NON_NIL_ERROR},
            "incorrect first arg",
        },
        ScriptEntry_t{
            ApiIDSendClientResponse,
            []Param_t{
                Param_t{ANONYMOUS, chSerResNil, nil},
                Param_t{ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NON_NIL_ERROR},
            "Nil first arg",
        },
        ScriptEntry_t{
            ApiIDRegisterServerReqHandler,
            []Param_t{},
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil},
                Result_t{ANONYMOUS, nil, ValidateNil},
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        ScriptEntry_t{
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{ANONYMOUS, 7, nil}},
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil},
                Result_t{ANONYMOUS, nil, ValidateNil},
                NON_NIL_ERROR,
            },
            "Incorrect arg",
        },
        ScriptEntry_t{
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []Result_t{
                Result_t{"chSerReq-0", nil, ValidateNonNil}, /* chan for incoming req */
                Result_t{"chSerRes-0", nil, ValidateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Register server handler to succeed",
        },
        ScriptEntry_t{
            ApiIDSendClientResponse,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Get from cache */
                Param_t{ANONYMOUS, 11, nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NON_NIL_ERROR},
            "incorrect second arg",
        },
        ScriptEntry_t{
            ApiIDSendClientResponse,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Get from cache */
                Param_t{ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                Param_t{ANONYMOUS, "dd", nil},
            },
            []Result_t{NON_NIL_ERROR},
            "incorrect third arg",
        },
        ScriptEntry_t{
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Close server handler */
            },
            []Result_t{NIL_ERROR},
            "Close server request handler via closing this channel.",
        },
        ScriptEntry_t{
            ApiIDSendClientResponse,
            []Param_t{
                Param_t{"chSerRes-x", chSerResWrNonNil, nil},
                Param_t{ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NON_NIL_ERROR},
            "sendClientresponse times out",
        },
        ScriptEntry_t{
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chSerRes-x", nil, nil}, /* Close chSerResNonNil chan */
            },
            []Result_t{NIL_ERROR},
            "close locally created channel",
        },
        ScriptEntry_t{
            ApiIDWriteJsonStringsChannel,
            []Param_t{
                Param_t{ANONYMOUS, nil, nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NON_NIL_ERROR},
            "fewer args",
        },
        ScriptEntry_t{
            ApiIDWriteJsonStringsChannel,
            []Param_t{
                Param_t{ANONYMOUS, 11, nil},
                Param_t{ANONYMOUS, nil, nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NON_NIL_ERROR},
            "Incorrect first arg",
        },
        ScriptEntry_t{
            ApiIDWriteJsonStringsChannel,
            []Param_t{
                Param_t{ANONYMOUS, chNilJson, nil}, /* Nil chan */
                Param_t{ANONYMOUS, nil, nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NON_NIL_ERROR},
            "Nil ch provided",
        },
        ScriptEntry_t{
            ApiIDWriteJsonStringsChannel,
            []Param_t{
                Param_t{"validChJson", chWrNonNilJson, nil}, /* Use valid chan */
                Param_t{ANONYMOUS, true, nil},        /* Incorrect arg */
                Param_t{ANONYMOUS, 1, nil},           /* timeout = 1 second */
            },
            []Result_t{NON_NIL_ERROR},
            "Incorrect second arg",
        },
        ScriptEntry_t{
            ApiIDWriteJsonStringsChannel,
            []Param_t{
                Param_t{"validChJson", nil, nil},                      /* Use valid chan from cache */
                Param_t{ANONYMOUS, []tele.JsonString_t{}, nil}, /* valid arg */
                Param_t{ANONYMOUS, "1", nil},                   /* incorrect type */
            },
            []Result_t{NON_NIL_ERROR},
            "Incorrect third arg",
        },
        ScriptEntry_t{
            ApiIDWriteJsonStringsChannel,
            []Param_t{
                Param_t{"validChJson", nil, nil}, /* Use valid chan from cache */
                Param_t{ANONYMOUS, []tele.JsonString_t{tele.JsonString_t("hello")}, nil},
                Param_t{ANONYMOUS, 1, nil}, /* timeout = 1sec */
            },
            []Result_t{NON_NIL_ERROR},
            "timeout to occur",
        },
        ScriptEntry_t{
            ApiIDReadJsonStringsChannel,
            []Param_t{
                Param_t{"validChRdJson", chRdNonNilJson, nil}, /* Use valid chan */
                Param_t{ANONYMOUS, 1, nil},
                Param_t{ANONYMOUS, 1, nil}, /* timeout = 1sec */
            },
            []Result_t{NIL_ANY, NON_NIL_ERROR},
            "timeout to occur",
        },
        ScriptEntry_t{
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"validChJson", nil, nil}, /* Get chWrite_0 from cache */
            },
            []Result_t{NIL_ERROR},
            "Close pub chennel",
        },
        ScriptEntry_t{
            ApiIDReadJsonStringsChannel,
            []Param_t{
                Param_t{ANONYMOUS, nil, nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NIL_ANY, NON_NIL_ERROR},
            "fewer args",
        },
        ScriptEntry_t{
            ApiIDReadJsonStringsChannel,
            []Param_t{
                Param_t{ANONYMOUS, 11, nil},
                Param_t{ANONYMOUS, nil, nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NIL_ANY, NON_NIL_ERROR},
            "Incorrect first arg",
        },
        ScriptEntry_t{
            ApiIDReadJsonStringsChannel,
            []Param_t{
                Param_t{ANONYMOUS, chNilJson, nil}, /* Nil chan */
                Param_t{ANONYMOUS, nil, nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NIL_ANY, NON_NIL_ERROR},
            "Nil ch provided",
        },
        ScriptEntry_t{
            ApiIDReadJsonStringsChannel,
            []Param_t{
                Param_t{"validChRdJson", nil, nil},   /* Use valid chan from cache */
                Param_t{ANONYMOUS, true, nil}, /* Incorrect arg. expect cnt */
                Param_t{ANONYMOUS, 1, nil},    /* timeout = 1 second */
            },
            []Result_t{NIL_ANY, NON_NIL_ERROR},
            "Incorrect second arg",
        },
        ScriptEntry_t{
            ApiIDReadJsonStringsChannel,
            []Param_t{
                Param_t{"validChRdJson", nil, nil},  /* Use valid chan from cache */
                Param_t{ANONYMOUS, 1, nil},   /* valid arg */
                Param_t{ANONYMOUS, "1", nil}, /* incorrect type */
            },
            []Result_t{NIL_ANY, NON_NIL_ERROR},
            "Incorrect third arg",
        },
        ScriptEntry_t{
            ApiIDReadJsonStringsChannel,
            []Param_t{
                Param_t{"validChRdJson", nil, nil}, /* Use valid chan from cache */
                Param_t{ANONYMOUS, 1, nil},
                Param_t{ANONYMOUS, 1, nil}, /* timeout = 1sec */
            },
            []Result_t{NIL_ANY, NON_NIL_ERROR},
            "channel closed",
        },
        ScriptEntry_t{
            ApiIDCloseRequestChannel,
            []Param_t{}, /* Missing arg */
            []Result_t{
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        ScriptEntry_t{
            ApiIDCloseRequestChannel,
            []Param_t{
                Param_t{ANONYMOUS, true, nil}, /* Incorrect type */
            },
            []Result_t{
                NON_NIL_ERROR,
            },
            "incorrect first arg",
        },
        ScriptEntry_t{
            ApiIDCloseChannel,
            []Param_t{
                Param_t{ANONYMOUS, chNonNilJson, nil},
            },
            []Result_t{NON_NIL_ERROR},
            "Unknown ch type",
        },
        ScriptEntry_t{
            ApiIDPause,
            []Param_t{},
            []Result_t{NON_NIL_ERROR},
            "fewer args",
        },
        ScriptEntry_t{
            ApiIDPause,
            []Param_t{Param_t{ANONYMOUS, "2", nil}},
            []Result_t{NON_NIL_ERROR},
            "Incorrect arg",
        },
        ScriptEntry_t{
            ApiIDPause,
            []Param_t{Param_t{ANONYMOUS, nil, getFail}},
            []Result_t{NON_NIL_ERROR},
            "Failed to get val",
        },
        ScriptEntry_t{
            ApiIDIsTelemetryIdle,
            []Param_t{Param_t{ANONYMOUS, "2", nil}},
            []Result_t{
                Result_t{ANONYMOUS, false, nil},
                NON_NIL_ERROR,
            },
            "redundant args",
        },
        TELE_IDLE_CHECK,
    },
}
