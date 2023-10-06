package libtest

import (
    . "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

/*
 * Register all long running handlers
 * Raise sys shutdown and call for idle check
 */
var pubSubShutdownSuite = ScriptSuite_t{
    Id:          "pubSubShutdownSuite",
    Description: "Test pub sub for request & response - Good run",
    Entries: []ScriptEntry_t{
        ScriptEntry_t{
            ApiIDRunPubSubProxy,
            []Param_t{Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil}},
            []Result_t{
                Result_t{"chPrxyClose-E", nil, ValidateNonNil}, /* Save in cache */
                NIL_ERROR, /*Expect nil error */
            },
            "test to exit on sys shutdown",
        },
        ScriptEntry_t{
            ApiIDRunPubSubProxy,
            []Param_t{Param_t{"chType_E", nil, nil}},
            []Result_t{NIL_ANY, NON_NIL_ERROR},
            "Fail duplicate",
        },
        ScriptEntry_t{ /* Get sub channel for events from engine only. */
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{"chType_E", nil, nil}, /* Fetch from cache */
                Param_t{"prod_E", tele.CHANNEL_PRODUCER_ENGINE, nil},
                EMPTY_STRING,
            },
            []Result_t{
                Result_t{"chRead-E", nil, ValidateNonNil},     /* Save in cache */
                Result_t{"chSubClose-E", nil, ValidateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "test to exit on sys shutdown",
        },
        ScriptEntry_t{
            ApiIDGetPubChannel, /* Simulate publish from engine */
            []Param_t{
                Param_t{"chType_E", nil, nil}, /* pub for events */
                Param_t{"prod_E", nil, nil},   /* from engine */
                EMPTY_STRING,
            },
            []Result_t{
                Result_t{"chWrite-E", nil, ValidateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "test to exit on sys shutdown",
        },

        /* Handler terminated by unsolicited response BEGIN in LState_ReadReq state*/
        ScriptEntry_t{ /* Test handler shutdown in stat = LState_ReadReq */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []Result_t{
                Result_t{"chSerReq-0", nil, ValidateNonNil}, /* chan for incoming req */
                Result_t{"chSerRes-0", nil, ValidateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Take to LState_ReadReq - unsolicited response test",
        },
        ScriptEntry_t{ /* unsolicited response while handler still blocked in sending req */
            ApiIDSendClientResponse,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Get from cache */
                Param_t{ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NIL_ERROR},
            "Write res while handler is blocked in read req state.",
        },
        PAUSE1,
        /* Handler terminated by unsolicited response END in LState_ReadReq state*/
        /* On any failure to terminate will fail next RegisterServerReqHandler call for same type */

        /* Handler terminated by unsolicited response BEGIN in LState_WriteReq state*/
        ScriptEntry_t{ /* Test handler shutdown in stat = LState_WriteReq */
            ApiIDRegisterServerReqHandler,
            /* Get from last register req which must have terminated */
            []Param_t{Param_t{"chType_1", nil, nil}},
            []Result_t{
                Result_t{"chSerReq-0", nil, ValidateNonNil}, /* chan for incoming req */
                Result_t{"chSerRes-0", nil, ValidateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Handler to exit on unsolicited response in LState_WriteReq",
        },
        ScriptEntry_t{ /* SNeak in duplicate failure */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []Result_t{NIL_ANY, NIL_ANY, NON_NIL_ERROR},
            "Duplicate req to fail",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request:Hello world"), nil},
            },
            []Result_t{
                Result_t{"chClientRes-0", nil, ValidateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "take server to LState_WriteReq - unsolicited response test",
        },
        PAUSE1, /* Wait for go routine to read client req */
        ScriptEntry_t{ /* unsolicited response while handler still blocked in sending req */
            ApiIDSendClientResponse,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Get from cache */
                Param_t{ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []Result_t{NIL_ERROR},
            "Write response when req is yet to be sent.",
        },
        ScriptEntry_t{
            ApiIDCloseRequestChannel, /* explicit request to close for req channel */
            []Param_t{Param_t{"chType_1", nil, nil}},
            []Result_t{NIL_ERROR},
            "Close channel created for client requests - LState_WriteReq",
        },
        PAUSE1, /* Wait for async shutdown */
        /* Handler terminated by unsolicited response BEGIN in LState_WriteReq state*/

        /* Handler terminated by closed res chan BEGIN in LState_ReadReq state*/
        ScriptEntry_t{ /* Test handler shutdown in stat = LState_ReadReq */
            /* Get from last register req which must have terminated */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []Result_t{
                Result_t{"chSerReq-0", nil, ValidateNonNil}, /* chan for incoming req */
                Result_t{"chSerRes-0", nil, ValidateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Take to LState_ReadReq for closed res chan",
        },
        ScriptEntry_t{ /* unsolicited response while handler still blocked in sending req */
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Get from cache */
            },
            []Result_t{NIL_ERROR},
            "res channel closed when waiting for response",
        },
        PAUSE1,
        /* Handler terminated by closed res chan END in LState_ReadReq state*/

        /* Handler terminated by closed res BEGIN in LState_WriteReq state*/
        ScriptEntry_t{ /* Test handler shutdown in stat = LState_WriteReq */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []Result_t{
                Result_t{"chSerReq-0", nil, ValidateNonNil}, /* chan for incoming req */
                Result_t{"chSerRes-0", nil, ValidateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Handler to exit on closed res chan in LState_WriteReq",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request:Hello world"), nil},
            },
            []Result_t{
                Result_t{"chClientRes-0", nil, ValidateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "take server to LState_WriteReq for closed test",
        },
        PAUSE1, /* Wait for go routine to read client req */
        ScriptEntry_t{ /* unsolicited response while handler still blocked in sending req */
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Get from cache */
            },
            []Result_t{NIL_ERROR},
            "res channel closed when waiting to write req",
        },
        ScriptEntry_t{
            ApiIDCloseRequestChannel, /* explicit request to close for req channel */
            []Param_t{Param_t{"chType_1", nil, nil}},
            []Result_t{NIL_ERROR},
            "Close channel created for client requests.",
        },
        PAUSE1,
        /* Handler terminated by closed res END in LState_WriteReq state*/

        /* Handler terminated by closed res channel BEGIN in LState_ReadRes state*/
        ScriptEntry_t{ /* Test handler shutdown in stat = LState_ReadReq */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_2", tele.CHANNEL_TYPE_ECHO, nil}},
            []Result_t{
                Result_t{"chSerReq-2", nil, ValidateNonNil}, /* chan for incoming req */
                Result_t{"chSerRes-2", nil, ValidateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Take to LState_ReadRes for closed res test",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_2", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_2", tele.ClientReq_t("requestX:Hello world"), nil},
            },
            []Result_t{
                Result_t{"chClientRes-0", nil, ValidateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "take server to LState_ReadRes for closed test",
        },
        ScriptEntry_t{
            ApiIDReadClientRequest,
            []Param_t{
                Param_t{"chSerReq-2", nil, nil},   /* Get from cache */
                Param_t{ANONYMOUS, 1, nil}, /* valid timeout */
            },
            []Result_t{
                Result_t{"req_2", nil, nil}, /* Validate against cache val for req_0 */
                NIL_ERROR,
            },
            "read req to put handler in read res state during closed test",
        },
        ScriptEntry_t{ /* close the chan */
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chSerRes-2", nil, nil}, /* Get from cache */
            },
            []Result_t{NIL_ERROR},
            "res channel closed when waiting for response",
        },
        ScriptEntry_t{
            ApiIDCloseRequestChannel, /* explicit request to close for req channel */
            []Param_t{Param_t{"chType_2", nil, nil}},
            []Result_t{NIL_ERROR},
            "Close channel created for client requests.",
        },
        PAUSE1,
        /* Handler terminated by closed res channel END in LState_ReadRes state*/

        ScriptEntry_t{ /* Test handler in sys shutdown in LState_WriteReq */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []Result_t{
                Result_t{"chSerReq-0", nil, ValidateNonNil}, /* chan for incoming req */
                Result_t{"chSerRes-0", nil, ValidateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "test to exit on sys shutdown LState_WriteReq",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request:Hello world"), nil},
            },
            []Result_t{
                Result_t{"chClientRes-0", nil, ValidateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "take server to LState_WriteReq",
        },
        ScriptEntry_t{ /* Create two more to pend to test closing their wr chans on exit */
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request1:Hello Mars"), nil},
            },
            []Result_t{
                Result_t{"chClientRes-0", nil, ValidateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "pend in list as first",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request2:Hello Venus"), nil},
            },
            []Result_t{
                Result_t{"chClientRes-0", nil, ValidateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "pend in list as second",
        },
        ScriptEntry_t{ /* Test handler shutdown in stat = LState_ReadRes */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_2", tele.CHANNEL_TYPE_SCS, nil}},
            []Result_t{
                Result_t{"chSerReq-1", nil, ValidateNonNil}, /* chan for incoming req */
                Result_t{"chSerRes-1", nil, ValidateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Register server handler to process requests and provide responses.",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_2", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_1", tele.ClientReq_t("requestSCS:Hello Mars"), nil},
            },
            []Result_t{
                Result_t{"chClientRes-0", nil, ValidateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "Send req to handler",
        },
        ScriptEntry_t{
            ApiIDReadClientRequest,
            []Param_t{
                Param_t{"chSerReq-1", nil, nil},   /* Get chRead_0 from cache */
                Param_t{ANONYMOUS, 1, nil}, /* valid timeout */
            },
            []Result_t{
                Result_t{"req_1", nil, nil}, /* Validate against cache val for req_1 */
                NIL_ERROR,
            },
            "read req to put handler in read res state during shutdown",
        },
        ScriptEntry_t{ /* Test handler shutdown in stat = LState_ReadReq */
            ApiIDRegisterServerReqHandler, /* No request from client */
            []Param_t{Param_t{"chType_3", tele.CHANNEL_TYPE_TEST_REQ, nil}},
            []Result_t{
                Result_t{"chSerReq-2", nil, ValidateNonNil}, /* chan for incoming req */
                Result_t{"chSerRes-2", nil, ValidateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Register server handler to process requests and provide responses.",
        },
        /* Now we have a pub-sub proxy and channels for publisher, subscriber,
         * client request & server handler.
         *
         * All the above register for system shutdown.
         *
         * Initiate system shutdown. Ensure everyone go down
         */
        ScriptEntry_t{
            ApiIDDoSysShutdown,
            []Param_t{}, /* Missed args */
            []Result_t{NON_NIL_ERROR},
            "insufficient args",
        },
        ScriptEntry_t{
            ApiIDDoSysShutdown,
            []Param_t{ /* Incorrect arg type */
                Param_t{ANONYMOUS, false, nil},
            },
            []Result_t{NON_NIL_ERROR},
            "incorrect arg",
        },
        ScriptEntry_t{
            ApiIDDoSysShutdown,
            []Param_t{ /* Timeout as 2 secs */
                Param_t{ANONYMOUS, 2, nil},
            },
            []Result_t{NIL_ERROR},
            "Initiate system shutdown",
        },
        TELE_IDLE_CHECK,        /* Expect all active handlers to go down */
        ScriptEntry_t{
            ApiIDGetPubChannel, /* Simulate publish from engine */
            []Param_t{
                Param_t{"chType_E", nil, nil}, /* pub for events */
                Param_t{"prod_E", nil, nil},   /* from engine */
                EMPTY_STRING,
            },
            []Result_t{NIL_ANY, NON_NIL_ERROR},
            "Fail to get after sys shutdown",
        },
        ScriptEntry_t{
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{"chType_E", nil, nil},
                Param_t{"prod_E", nil, nil},
                EMPTY_STRING,
            },
            []Result_t{NIL_ANY, NIL_ANY, NON_NIL_ERROR},
            "Fail to get after sys shutdown",
        },
        ScriptEntry_t{
            ApiIDRunPubSubProxy,
            []Param_t{Param_t{"chType_E", nil, nil}},
            []Result_t{NIL_ANY, NON_NIL_ERROR},
            "Fail to get after sys shutdown",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request:Hello world"), nil},
            },
            []Result_t{NIL_ANY, NON_NIL_ERROR},
            "Fail to send after sys shutdown",
        },
        ScriptEntry_t{
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_EVENTS, nil}},
            []Result_t{NIL_ANY, NIL_ANY, NON_NIL_ERROR},
            "Fail to run after sys shutdown",
        },
        PAUSE1,
        TELE_IDLE_CHECK,
        ScriptEntry_t{
            ApiIDInitSysShutdown,
            []Param_t{ Param_t{ANONYMOUS, 2, nil} }, /* redundant arg */
            []Result_t{NON_NIL_ERROR},
            "excess args to fail",
        },
        ScriptEntry_t{
            ApiIDInitSysShutdown,
            []Param_t{}, /* no args required */
            []Result_t{NIL_ERROR},
            "Initialized for a clean state, just in case for subsequent tests.",
        },
    },
}
