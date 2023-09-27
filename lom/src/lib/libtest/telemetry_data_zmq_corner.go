package libtest

import (
    script "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

var pubSubShutdownSuite = testSuite_t{
    id:          "pubSubReqRepSuite",
    description: "Test pub sub for request & response - Good run",
    tests: []testEntry_t{
        testEntry_t{
            script.ApiIDRunPubSubProxy,
            []script.Param_t{script.Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil}},
            []result_t{
                result_t{"chPrxyClose-E", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR, /*Expect nil error */
            },
            "Rub pubsub proxy, required to bind publishers & subscribers",
        },
        testEntry_t{ /* Get sub channel for events from engine only. */
            script.ApiIDGetSubChannel,
            []script.Param_t{
                script.Param_t{"chType_E", nil, nil}, /* Fetch chType_1 from cache */
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
        testEntry_t{
            script.ApiIDGetPubChannel, /* Simulate publish from engine */
            []script.Param_t{
                script.Param_t{"chType_E", nil, nil}, /* pub for events */
                script.Param_t{"prod_E", nil, nil},   /* from engine */
                EMPTY_STRING,
            },
            []result_t{
                result_t{"chWrite-E", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get pub channel for counters as if from Plugin Mgr",
        },
        testEntry_t{
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
            script.ApiIDSendClientRequest,
            []script.Param_t{
                script.Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                script.Param_t{"req_0", tele.ClientReq_t("request:Hello world"), nil},
            },
            []result_t{
                result_t{"chClientRes-0", nil, validateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "Send a request as if from client",
        },
        /* 
         * Now we have a pub-sub proxy and channels for publisher, subscriber,
         * client request & server handler.
         * 
         * All the above register for system shutdown.
         * 
         * Initiate system shutdown. Ensure everyone go down
         */
        testEntry_t{
            script.ApiIDSysShutdown,
            []script.Param_t{},                 /* Missed args */
            []result_t{ NON_NIL_ERROR },
            "insufficient args",
        },
        testEntry_t{
            script.ApiIDSysShutdown,
            []script.Param_t{                   /* Incorrect arg type */
                script.Param_t{script.ANONYMOUS, false, nil},
            },
            []result_t{ NON_NIL_ERROR },
            "incorrect arg",
        },
        testEntry_t{
            script.ApiIDSysShutdown,
            []script.Param_t{           /* Timeout as 2 secs */
                script.Param_t{script.ANONYMOUS, 2, nil},
            },
            []result_t{ NIL_ERROR },
            "Initiate system shutdown",
        },
        PAUSE1,
        TELE_IDLE_CHECK,
    },
}


