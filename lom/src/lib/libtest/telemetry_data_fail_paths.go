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
                script.Param_t{"req_0", tele.ClientReq_t("request:Hello world"), nil},
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

var pubSubScriptAPIValidate = testSuite_t{
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
                script.Param_t{"req_0", tele.ClientReq_t("request:Hello world"), nil},
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
    },
}
