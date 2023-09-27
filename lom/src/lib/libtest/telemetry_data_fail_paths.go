package libtest

import (
    script "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)


var pubSubFailSuite = testSuite_t{
    id:          "pubSubFailSuite",
    description: "For corner & failure cases",
    tests: []testEntry_t{
        testEntry_t{
            script.ApiIDGetPubChannel, /* Try publish to fail */
            []script.Param_t{
                script.Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}, /* pub for counters */
                script.Param_t{"prod_PM", tele.CHANNEL_PRODUCER_PLMGR, nil},  /* from Plugin Mgr */
                script.Param_t{"PMgr-1", "", nil},   /* missing suffix */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Failed to get Pub channel for missing suffix",
        },
        testEntry_t{
            script.ApiIDGetPubChannel, /* Get a valid pub channel */
            []script.Param_t{
                script.Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}, /* pub for counters */
                script.Param_t{"prod_PM", tele.CHANNEL_PRODUCER_PLMGR, nil},  /* from Plugin Mgr */
                script.Param_t{"PMgr-1", "test", nil},   /* non-empty suffix */
            },
            []result_t{
                result_t{"chWrite-0", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get a channel",
        },
        testEntry_t{
            script.ApiIDGetPubChannel, /* Try again for same type */
            []script.Param_t{
                script.Param_t{"chType_C", nil, nil},           /* Get from cache*/
                script.Param_t{"prod_PM", nil, nil},            /* from cache */
                script.Param_t{"PMgr-1", nil, nil},             /* Same as above */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Fail as duplicate channel",
        },
        testEntry_t{ /* Get sub channel for events from engine only. */
            script.ApiIDGetSubChannel,
            []script.Param_t{
                script.Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil},
                script.Param_t{"prod_E", tele.CHANNEL_PRODUCER_ENGINE, nil},
                EMPTY_STRING,
            },
            []result_t{
                result_t{"chRead-E", nil, validateNonNil},     /* Save in cache */
                result_t{"chSubClose-E", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get sub channel for events from Engine",
        },
        testEntry_t{ /* Get sub channel for events from engine only. */
            script.ApiIDGetSubChannel,      /* Get Sub channel for same params as above */
            []script.Param_t{
                script.Param_t{"chType_E", nil, nil},
                script.Param_t{"prod_E", nil, nil},
                EMPTY_STRING,
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil},   /* Nil on failure */
                result_t{script.ANONYMOUS, nil, validateNil},   /* Nil on failure */
                NON_NIL_ERROR,
            },
            "Get sub channel for same type to fail",
        },
        testEntry_t{
            script.ApiIDCloseChannel,
            []script.Param_t{
                script.Param_t{"chWrite-0", nil, nil},      /* Close pub chan opened above */
                script.Param_t{"chSubClose-E", nil, nil},   /* Close sub chan opened above */
            },
            []result_t{NIL_ERROR},
            "Close channels",
        },
        testEntry_t{
            script.ApiIDSendClientRequest,                  /* Request for incorrect type */
            []script.Param_t{
                script.Param_t{"chType_1", tele.CHANNEL_TYPE_EVENTS, nil}, /* Fetch chType_1 from cache */
                script.Param_t{"req_0", tele.ClientReq_t("request-1:Hello world"), nil},
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* chan to read response */
                NON_NIL_ERROR,
            },
            "Expect request to fail due to incorrect request type",
        },
        testEntry_t{
            script.ApiIDRegisterServerReqHandler,
            []script.Param_t{script.Param_t{"chType_1", tele.CHANNEL_TYPE_EVENTS, nil}},
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil},    /* No chan on error */
                result_t{script.ANONYMOUS, nil, validateNil},    /* No chan on error */
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

var scriptAPIValidate = testSuite_t{
    id:          "ScriptAPIValidation",         /* All the below are for failure only */
    description: "For corner & failure cases",
    tests: []testEntry_t{
        testEntry_t{
            script.ApiIDGetPubChannel, 
            []script.Param_t{
                script.Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}, /* pub for counters */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        testEntry_t{
            script.ApiIDGetPubChannel, /* Get a valid pub channel */
            []script.Param_t{
                script.Param_t{"prod_PM", tele.CHANNEL_PRODUCER_PLMGR, nil},  /* from Plugin Mgr */
                script.Param_t{"PMgr-1", "test", nil},   /* non-empty suffix */
                script.Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}, /* pub for counters */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect first arg",
        },
        testEntry_t{
            script.ApiIDGetPubChannel, /* Get a valid pub channel */
            []script.Param_t{
                script.Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}, /* pub for counters */
                script.Param_t{"prod_PM", true, nil},  /* from Plugin Mgr */
                script.Param_t{"PMgr-1", "test", nil},   /* non-empty suffix */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect second arg",
        },
        testEntry_t{
            script.ApiIDGetPubChannel,
            []script.Param_t{
                script.Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}, /* pub for counters */
                script.Param_t{"prod_PM", tele.CHANNEL_PRODUCER_PLMGR, nil},  /* from Plugin Mgr */
                script.Param_t{"PMgr-1", 11, nil},   /* incorrect type suffix */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect type for third arg",
        },
        testEntry_t{ 
            script.ApiIDGetSubChannel,
            []script.Param_t{
                script.Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil},
                EMPTY_STRING,
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        testEntry_t{ 
            script.ApiIDGetSubChannel,
            []script.Param_t{
                script.Param_t{"prod_E", tele.CHANNEL_PRODUCER_ENGINE, nil},
                script.Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil},
                EMPTY_STRING,
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect first arg",
        },
        testEntry_t{ 
            script.ApiIDGetSubChannel,
            []script.Param_t{
                script.Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil},
                EMPTY_STRING,
                script.Param_t{"prod_E", tele.CHANNEL_PRODUCER_ENGINE, nil},
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect second arg",
        },
        testEntry_t{ 
            script.ApiIDGetSubChannel,
            []script.Param_t{
                script.Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil},
                script.Param_t{"prod_E", tele.CHANNEL_PRODUCER_ENGINE, nil},
                script.Param_t{script.ANONYMOUS, true, nil},
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect third arg",
        },
        testEntry_t{
            script.ApiIDRunPubSubProxy,
            []script.Param_t{},
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        testEntry_t{
            script.ApiIDRunPubSubProxy,
            []script.Param_t{script.Param_t{"chType_1", tele.CHANNEL_PRODUCER_ENGINE, nil}},
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Rub pubsub proxy, required to bind publishers & subscribers",
        },
        testEntry_t{
            script.ApiIDSendClientRequest,
            []script.Param_t{
                script.Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        testEntry_t{
            script.ApiIDSendClientRequest,
            []script.Param_t{
                script.Param_t{"req_0", tele.ClientReq_t("request-2:Hello world"), nil},
                script.Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect first arg",
        },
        testEntry_t{
            script.ApiIDSendClientRequest,
            []script.Param_t{
                script.Param_t{"chType_1", tele.CHANNEL_TYPE_EVENTS, nil},
                script.Param_t{"req_0", 11, nil},
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect second arg",
        },
        testEntry_t{
            script.ApiIDReadClientResponse,                 /* Client read its response */
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, 1, nil},   /* timeout = 1 second */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        testEntry_t{
            script.ApiIDReadClientResponse,                 /* Client read its response */
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, 1, nil},   /* timeout = 1 second */
                script.Param_t{"chClientRes-0", nil, nil},  /* Get chan from cache */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect first arg",
        },
        testEntry_t{
            script.ApiIDReadClientResponse,                 /* Client read its response */
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, chTmpRes, nil},  /* Get chan from cache */
                script.Param_t{script.ANONYMOUS, 1, nil},   /* timeout = 1 second */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "nil first arg",
        },
        testEntry_t{
            script.ApiIDSendClientRequest,  /* Good one to get channel */
            []script.Param_t{
                script.Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil},
                script.Param_t{"req_0", tele.ClientReq_t("request-3:Hello world"), nil},
            },
            []result_t{
                result_t{"chClientRes-0", nil, validateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "Send a request as if from client",
        },
        testEntry_t{
            script.ApiIDReadClientResponse,                 /* Client read its response */
            []script.Param_t{
                script.Param_t{"chClientRes-0", nil, nil},  /* Get chan from cache */
                script.Param_t{script.ANONYMOUS, "rere", nil},   /* timeout not int */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "incorrect second arg",
        },
        testEntry_t{
            script.ApiIDReadClientResponse,                 /* Client read its response */
            []script.Param_t{
                script.Param_t{"chClientRes-0", nil, nil},  /* Get chan from cache */
                script.Param_t{script.ANONYMOUS, 1, nil},   /* timeout 1 sec */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Read response times out",
        },
        testEntry_t{
            script.ApiIDCloseRequestChannel,                /* explicit request to close for req channel */
            []script.Param_t{
                script.Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
            },
            []result_t{
                NIL_ERROR,
            },
            "Close channel created for client requests.",
        },
        PAUSE2,                 /* Allow req channel to close */
        testEntry_t{
            script.ApiIDReadClientResponse,                 /* Client read its response */
            []script.Param_t{
                script.Param_t{"chClientRes-0", nil, nil},  /* Get chan from cache */
                script.Param_t{script.ANONYMOUS, 1, nil},   /* timeout 1 sec */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Read from closed chan for response",
        },
        testEntry_t{
            script.ApiIDReadClientRequest,           
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, 1, nil},  
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        testEntry_t{
            script.ApiIDReadClientRequest,                 
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, 1, nil},  
                script.Param_t{script.ANONYMOUS, nil, nil},    
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Incorrect first arg",
        },
        testEntry_t{
            script.ApiIDReadClientRequest,                 
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, chTmpReq, nil},    
                script.Param_t{script.ANONYMOUS, 1, nil},  
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Nil first arg",
        },
        testEntry_t{                /* Get a proper one to get valid handle for use */
            script.ApiIDRegisterServerReqHandler,
            []script.Param_t{script.Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []result_t{
                result_t{"chSerReq-0", nil, validateNonNil},    /* chan for incoming req */
                result_t{"chSerRes-0", nil, validateNonNil},    /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Register server handler to process requests and provide responses.",
        },
        testEntry_t{
            script.ApiIDReadClientRequest,      
            []script.Param_t{
                script.Param_t{"chSerReq-0", nil, nil},         /* Get chRead_0 from cache */
                script.Param_t{script.ANONYMOUS, true, nil},    /* Invalid timeout */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Nil return*/
                NON_NIL_ERROR,
            },
            "Fail with invalid timeout",
        },
        testEntry_t{
            script.ApiIDReadClientRequest,      
            []script.Param_t{
                script.Param_t{"chSerReq-0", nil, nil},     /* Get chRead_0 from cache */
                script.Param_t{script.ANONYMOUS, 1, nil},   /* valid timeout */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* no result */
                NON_NIL_ERROR,
            },
            "Read fails with timeout once",
        },
        testEntry_t{
            script.ApiIDCloseChannel,
            []script.Param_t{
                script.Param_t{"chSerRes-0", nil, nil},     /* Close server handler */
            },
            []result_t{NIL_ERROR},
            "Close server request handler via closing this channel.",
        },
        PAUSE2,
        testEntry_t{
            script.ApiIDReadClientRequest,      
            []script.Param_t{
                script.Param_t{"chSerReq-0", nil, nil},     /* Get from cache */
                script.Param_t{script.ANONYMOUS, 1, nil},   /* valid timeout */
            },
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil}, /* Read fails with timeout */
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


var scriptAPIValidate_2 = testSuite_t{
    id:          "ScriptAPIValidation",         /* All the below are for failure only */
    description: "For corner & failure cases",
    tests: []testEntry_t{
        testEntry_t{
            script.ApiIDSendClientResponse,
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                script.Param_t{script.ANONYMOUS, 1, nil},
            },
            []result_t{ NON_NIL_ERROR },
            "fewer args",
        },
        testEntry_t{
            script.ApiIDSendClientResponse,             
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, true, nil},
                script.Param_t{script.ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},  
                script.Param_t{script.ANONYMOUS, 1, nil},
            },
            []result_t{ NON_NIL_ERROR },
            "incorrect first arg",
        },
        testEntry_t{
            script.ApiIDSendClientResponse,             
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, chSerResNil, nil},   
                script.Param_t{script.ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                script.Param_t{script.ANONYMOUS, 1, nil},  
            },
            []result_t{ NON_NIL_ERROR },
            "Nil first arg",
        },
        testEntry_t{
            script.ApiIDRegisterServerReqHandler,
            []script.Param_t{},
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil},    
                result_t{script.ANONYMOUS, nil, validateNil},   
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        testEntry_t{
            script.ApiIDRegisterServerReqHandler,
            []script.Param_t{script.Param_t{script.ANONYMOUS, 7, nil}},
            []result_t{
                result_t{script.ANONYMOUS, nil, validateNil},    
                result_t{script.ANONYMOUS, nil, validateNil},   
                NON_NIL_ERROR,
            },
            "Incorrect arg",
        },
        testEntry_t{
            script.ApiIDRegisterServerReqHandler,
            []script.Param_t{script.Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []result_t{
                result_t{"chSerReq-0", nil, validateNonNil},    /* chan for incoming req */
                result_t{"chSerRes-0", nil, validateNonNil},    /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Register server handler to succeed",
        },
        testEntry_t{
            script.ApiIDSendClientResponse,             
            []script.Param_t{
                script.Param_t{"chSerRes-0", nil, nil},         /* Get from cache */
                script.Param_t{script.ANONYMOUS, 11, nil},
                script.Param_t{script.ANONYMOUS, 1, nil},  
            },
            []result_t{ NON_NIL_ERROR },
            "incorrect second arg",
        },
        testEntry_t{
            script.ApiIDSendClientResponse,             
            []script.Param_t{
                script.Param_t{"chSerRes-0", nil, nil},         /* Get from cache */
                script.Param_t{script.ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                script.Param_t{script.ANONYMOUS, "dd", nil},  
            },
            []result_t{ NON_NIL_ERROR },
            "incorrect third arg",
        },
        testEntry_t{
            script.ApiIDCloseChannel,
            []script.Param_t{
                script.Param_t{"chSerRes-0", nil, nil},     /* Close server handler */
            },
            []result_t{NIL_ERROR},
            "Close server request handler via closing this channel.",
        },
        testEntry_t{
            script.ApiIDSendClientResponse,             
            []script.Param_t{
                script.Param_t{"chSerRes-x", chSerResWrNonNil, nil},   
                script.Param_t{script.ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                script.Param_t{script.ANONYMOUS, 1, nil},  
            },
            []result_t{ NON_NIL_ERROR },
            "sendClientresponse times out",
        },
        testEntry_t{
            script.ApiIDCloseChannel,
            []script.Param_t{
                script.Param_t{"chSerRes-x", nil, nil},     /* Close chSerResNonNil chan */
            },
            []result_t{NIL_ERROR},
            "close locally created channel",
        },
        testEntry_t{
            script.ApiIDWriteJsonStringsChannel,
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, nil, nil},
                script.Param_t{script.ANONYMOUS, 1, nil},
            },
            []result_t{ NON_NIL_ERROR },
            "fewer args",
        },
        testEntry_t{
            script.ApiIDWriteJsonStringsChannel,
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, 11, nil},           
                script.Param_t{script.ANONYMOUS, nil, nil},
                script.Param_t{script.ANONYMOUS, 1, nil},        
            },
            []result_t{ NON_NIL_ERROR },
            "Incorrect first arg",
        },
        testEntry_t{
            script.ApiIDWriteJsonStringsChannel,
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, chNilJson, nil},        /* Nil chan */
                script.Param_t{script.ANONYMOUS, nil, nil},
                script.Param_t{script.ANONYMOUS, 1, nil},        
            },
            []result_t{ NON_NIL_ERROR },
            "Nil ch provided",
        },
        testEntry_t{
            script.ApiIDWriteJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"validChJson", chWrNonNilJson, nil}, /* Use valid chan */
                script.Param_t{script.ANONYMOUS, true, nil},        /* Incorrect arg */
                script.Param_t{script.ANONYMOUS, 1, nil},           /* timeout = 1 second */
            },
            []result_t{ NON_NIL_ERROR },
            "Incorrect second arg",
        },
        testEntry_t{
            script.ApiIDWriteJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"validChJson", nil, nil},            /* Use valid chan from cache */
                script.Param_t{script.ANONYMOUS, []tele.JsonString_t{}, nil}, /* valid arg */
                script.Param_t{script.ANONYMOUS, "1", nil},         /* incorrect type */
            },
            []result_t{ NON_NIL_ERROR },
            "Incorrect third arg",
        },
        testEntry_t{
            script.ApiIDWriteJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"validChJson", nil, nil},  /* Use valid chan from cache */
                script.Param_t{script.ANONYMOUS, []tele.JsonString_t{tele.JsonString_t("hello")}, nil},
                script.Param_t{script.ANONYMOUS, 1, nil}, /* timeout = 1sec */
            },
            []result_t{ NON_NIL_ERROR },
            "timeout to occur",
        },
        testEntry_t{
            script.ApiIDReadJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"validChRdJson", chRdNonNilJson, nil},/* Use valid chan */
                script.Param_t{script.ANONYMOUS, 1, nil},
                script.Param_t{script.ANONYMOUS, 1, nil},           /* timeout = 1sec */
            },
            []result_t{ NIL_ANY, NON_NIL_ERROR },
            "timeout to occur",
        },
        testEntry_t{
            script.ApiIDCloseChannel,
            []script.Param_t{
                script.Param_t{"validChJson", nil, nil}, /* Get chWrite_0 from cache */
            },
            []result_t{NIL_ERROR},
            "Close pub chennel",
        },
        testEntry_t{
            script.ApiIDReadJsonStringsChannel,
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, nil, nil},
                script.Param_t{script.ANONYMOUS, 1, nil},
            },
            []result_t{ NIL_ANY, NON_NIL_ERROR },
            "fewer args",
        },
        testEntry_t{
            script.ApiIDReadJsonStringsChannel,
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, 11, nil},           
                script.Param_t{script.ANONYMOUS, nil, nil},
                script.Param_t{script.ANONYMOUS, 1, nil},        
            },
            []result_t{ NIL_ANY, NON_NIL_ERROR },
            "Incorrect first arg",
        },
        testEntry_t{
            script.ApiIDReadJsonStringsChannel,
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, chNilJson, nil},        /* Nil chan */
                script.Param_t{script.ANONYMOUS, nil, nil},
                script.Param_t{script.ANONYMOUS, 1, nil},        
            },
            []result_t{ NIL_ANY, NON_NIL_ERROR },
            "Nil ch provided",
        },
        testEntry_t{
            script.ApiIDReadJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"validChRdJson", nil, nil},          /* Use valid chan from cache */
                script.Param_t{script.ANONYMOUS, true, nil},        /* Incorrect arg. expect cnt */
                script.Param_t{script.ANONYMOUS, 1, nil},           /* timeout = 1 second */
            },
            []result_t{ NIL_ANY, NON_NIL_ERROR },
            "Incorrect second arg",
        },
        testEntry_t{
            script.ApiIDReadJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"validChRdJson", nil, nil},          /* Use valid chan from cache */
                script.Param_t{script.ANONYMOUS, 1, nil},           /* valid arg */
                script.Param_t{script.ANONYMOUS, "1", nil},         /* incorrect type */
            },
            []result_t{ NIL_ANY, NON_NIL_ERROR },
            "Incorrect third arg",
        },
        testEntry_t{
            script.ApiIDReadJsonStringsChannel,
            []script.Param_t{
                script.Param_t{"validChRdJson", nil, nil},          /* Use valid chan from cache */
                script.Param_t{script.ANONYMOUS, 1, nil},
                script.Param_t{script.ANONYMOUS, 1, nil},           /* timeout = 1sec */
            },
            []result_t{ NIL_ANY, NON_NIL_ERROR },
            "channel closed",
        },
        testEntry_t{
            script.ApiIDCloseRequestChannel,
            []script.Param_t{},                                     /* Missing arg */
            []result_t{
                NON_NIL_ERROR,
            },
            "fewer args",
        },
        testEntry_t{
            script.ApiIDCloseRequestChannel,
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, true, nil},        /* Incorrect type */
            },
            []result_t{
                NON_NIL_ERROR,
            },
            "incorrect first arg",
        },
        testEntry_t{
            script.ApiIDCloseChannel,
            []script.Param_t{
                script.Param_t{script.ANONYMOUS, chNonNilJson, nil},
            },
            []result_t{NON_NIL_ERROR},
            "Unknown ch type",
        },
        testEntry_t{ 
            script.ApiIDPause,
            []script.Param_t{},
            []result_t{NON_NIL_ERROR},
            "fewer args",
        },
        testEntry_t{
            script.ApiIDPause,
            []script.Param_t{script.Param_t{script.ANONYMOUS, "2", nil}},
            []result_t{NON_NIL_ERROR},
            "Incorrect arg",
        },
        testEntry_t{
            script.ApiIDIsTelemetryIdle,
            []script.Param_t{script.Param_t{script.ANONYMOUS, "2", nil}},
            []result_t{
                result_t {script.ANONYMOUS, false, nil},
                NON_NIL_ERROR,
            },
            "redundant args",
        },
        TELE_IDLE_CHECK,
    },
}
