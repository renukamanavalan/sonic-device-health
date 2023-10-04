package libtest

import (
    . "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

/*
 * BIG NOTE:  Let this be a last test suite.
 * After shutdown, no API will succeed
 * There is no way to revert shutdown -- One way to exit
 */
var pubSubShutdownSuite = testSuite_t{
    id:          "pubSubShutdownSuite",
    description: "Test pub sub for request & response - Good run",
    tests: []testEntry_t{
        testEntry_t{
            ApiIDRunPubSubProxy,
            []Param_t{Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil}},
            []result_t{
                result_t{"chPrxyClose-E", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR, /*Expect nil error */
            },
            "test to exit on sys shutdown",
        },
        testEntry_t{
            ApiIDRunPubSubProxy,
            []Param_t{Param_t{"chType_E", nil, nil}},
            []result_t{NIL_ANY, NON_NIL_ERROR},
            "Fail duplicate",
        },
        testEntry_t{ /* Get sub channel for events from engine only. */
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{"chType_E", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"prod_E", tele.CHANNEL_PRODUCER_ENGINE, nil},
                EMPTY_STRING,
            },
            []result_t{
                result_t{"chRead-E", nil, validateNonNil},     /* Save in cache */
                result_t{"chSubClose-E", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "test to exit on sys shutdown",
        },
        testEntry_t{
            ApiIDGetPubChannel, /* Simulate publish from engine */
            []Param_t{
                Param_t{"chType_E", nil, nil}, /* pub for events */
                Param_t{"prod_E", nil, nil},   /* from engine */
                EMPTY_STRING,
            },
            []result_t{
                result_t{"chWrite-E", nil, validateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "test to exit on sys shutdown",
        },

        /* Handler terminated by unsolicited response BEGIN in LState_ReadReq state*/
        testEntry_t{ /* Test handler shutdown in stat = LState_ReadReq */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []result_t{
                result_t{"chSerReq-0", nil, validateNonNil}, /* chan for incoming req */
                result_t{"chSerRes-0", nil, validateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Take to LState_ReadReq - unsolicited response test",
        },
        testEntry_t{ /* unsolicited response while handler still blocked in sending req */
            ApiIDSendClientResponse,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Get from cache */
                Param_t{ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []result_t{NIL_ERROR},
            "Write res while handler is blocked in read req state.",
        },
        PAUSE1,
        /* Handler terminated by unsolicited response END in LState_ReadReq state*/

        /* Handler terminated by unsolicited response BEGIN in LState_WriteReq state*/
        testEntry_t{ /* Test handler shutdown in stat = LState_WriteReq */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []result_t{
                result_t{"chSerReq-0", nil, validateNonNil}, /* chan for incoming req */
                result_t{"chSerRes-0", nil, validateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Handler to exit on unsolicited response in LState_WriteReq",
        },
        testEntry_t{ /* SNeak in duplicate failure */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []result_t{NIL_ANY, NIL_ANY, NON_NIL_ERROR},
            "Duplicate req to fail",
        },
        testEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request:Hello world"), nil},
            },
            []result_t{
                result_t{"chClientRes-0", nil, validateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "take server to LState_WriteReq - unsolicited response test",
        },
        PAUSE1, /* Wait for go routine to read client req */
        testEntry_t{ /* unsolicited response while handler still blocked in sending req */
            ApiIDSendClientResponse,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Get from cache */
                Param_t{ANONYMOUS, tele.ServerRes_t("resp: ok"), nil},
                Param_t{ANONYMOUS, 1, nil},
            },
            []result_t{NIL_ERROR},
            "Write response when req is yet to be sent.",
        },
        testEntry_t{
            ApiIDCloseRequestChannel, /* explicit request to close for req channel */
            []Param_t{Param_t{"chType_1", nil, nil}},
            []result_t{NIL_ERROR},
            "Close channel created for client requests.",
        },
        PAUSE1, /* Wait for async shutdown */
        /* Handler terminated by unsolicited response BEGIN in LState_WriteReq state*/

        /* Handler terminated by closed res chan BEGIN in LState_ReadReq state*/
        testEntry_t{ /* Test handler shutdown in stat = LState_ReadReq */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []result_t{
                result_t{"chSerReq-0", nil, validateNonNil}, /* chan for incoming req */
                result_t{"chSerRes-0", nil, validateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Take to LState_ReadReq for closed res chan",
        },
        testEntry_t{ /* unsolicited response while handler still blocked in sending req */
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Get from cache */
            },
            []result_t{NIL_ERROR},
            "res channel closed when waiting for response",
        },
        PAUSE1,
        /* Handler terminated by closed res chan END in LState_ReadReq state*/

        /* Handler terminated by closed res BEGIN in LState_WriteReq state*/
        testEntry_t{ /* Test handler shutdown in stat = LState_WriteReq */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []result_t{
                result_t{"chSerReq-0", nil, validateNonNil}, /* chan for incoming req */
                result_t{"chSerRes-0", nil, validateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Handler to exit on closed res chan in LState_WriteReq",
        },
        testEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request:Hello world"), nil},
            },
            []result_t{
                result_t{"chClientRes-0", nil, validateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "take server to LState_WriteReq for closed test",
        },
        PAUSE1, /* Wait for go routine to read client req */
        testEntry_t{ /* unsolicited response while handler still blocked in sending req */
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Get from cache */
            },
            []result_t{NIL_ERROR},
            "res channel closed when waiting to write req",
        },
        testEntry_t{
            ApiIDCloseRequestChannel, /* explicit request to close for req channel */
            []Param_t{Param_t{"chType_1", nil, nil}},
            []result_t{NIL_ERROR},
            "Close channel created for client requests.",
        },
        PAUSE1,
        /* Handler terminated by closed res END in LState_WriteReq state*/

        /* Handler terminated by closed res channel BEGIN in LState_ReadRes state*/
        testEntry_t{ /* Test handler shutdown in stat = LState_ReadReq */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_2", tele.CHANNEL_TYPE_ECHO, nil}},
            []result_t{
                result_t{"chSerReq-2", nil, validateNonNil}, /* chan for incoming req */
                result_t{"chSerRes-2", nil, validateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Take to LState_ReadRes for closed res test",
        },
        testEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_2", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_2", tele.ClientReq_t("requestX:Hello world"), nil},
            },
            []result_t{
                result_t{"chClientRes-0", nil, validateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "take server to LState_WriteReq for closed test",
        },
        testEntry_t{
            ApiIDReadClientRequest,
            []Param_t{
                Param_t{"chSerReq-2", nil, nil},   /* Get from cache */
                Param_t{ANONYMOUS, 1, nil}, /* valid timeout */
            },
            []result_t{
                result_t{"req_2", nil, nil}, /* Validate against cache val for req_0 */
                NIL_ERROR,
            },
            "read req to put handler in read res state during closed test",
        },
        testEntry_t{ /* close the chan */
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chSerRes-2", nil, nil}, /* Get from cache */
            },
            []result_t{NIL_ERROR},
            "res channel closed when waiting for response",
        },
        testEntry_t{
            ApiIDCloseRequestChannel, /* explicit request to close for req channel */
            []Param_t{Param_t{"chType_2", nil, nil}},
            []result_t{NIL_ERROR},
            "Close channel created for client requests.",
        },
        PAUSE1,
        /* Handler terminated by closed res channel END in LState_ReadRes state*/

        testEntry_t{ /* Test handler in sys shutdown in LState_WriteReq */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []result_t{
                result_t{"chSerReq-0", nil, validateNonNil}, /* chan for incoming req */
                result_t{"chSerRes-0", nil, validateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "test to exit on sys shutdown LState_WriteReq",
        },
        testEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request:Hello world"), nil},
            },
            []result_t{
                result_t{"chClientRes-0", nil, validateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "take server to LState_WriteReq",
        },
        testEntry_t{ /* Create two more to pend to test closing their wr chans on exit */
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request1:Hello Mars"), nil},
            },
            []result_t{
                result_t{"chClientRes-0", nil, validateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "pend in list as first",
        },
        testEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request2:Hello Venus"), nil},
            },
            []result_t{
                result_t{"chClientRes-0", nil, validateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "pend in list as second",
        },
        testEntry_t{ /* Test handler shutdown in stat = LState_ReadRes */
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_2", tele.CHANNEL_TYPE_SCS, nil}},
            []result_t{
                result_t{"chSerReq-1", nil, validateNonNil}, /* chan for incoming req */
                result_t{"chSerRes-1", nil, validateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Register server handler to process requests and provide responses.",
        },
        testEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_2", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_1", tele.ClientReq_t("requestSCS:Hello Mars"), nil},
            },
            []result_t{
                result_t{"chClientRes-0", nil, validateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "Send req to handler",
        },
        testEntry_t{
            ApiIDReadClientRequest,
            []Param_t{
                Param_t{"chSerReq-1", nil, nil},   /* Get chRead_0 from cache */
                Param_t{ANONYMOUS, 1, nil}, /* valid timeout */
            },
            []result_t{
                result_t{"req_1", nil, nil}, /* Validate against cache val for req_1 */
                NIL_ERROR,
            },
            "read req to put handler in read res state during shutdown",
        },
        testEntry_t{ /* Test handler shutdown in stat = LState_ReadReq */
            ApiIDRegisterServerReqHandler, /* No request from client */
            []Param_t{Param_t{"chType_3", tele.CHANNEL_TYPE_TEST_REQ, nil}},
            []result_t{
                result_t{"chSerReq-2", nil, validateNonNil}, /* chan for incoming req */
                result_t{"chSerRes-2", nil, validateNonNil}, /* chan for outgoing res */
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
        testEntry_t{
            ApiIDDoSysShutdown,
            []Param_t{}, /* Missed args */
            []result_t{NON_NIL_ERROR},
            "insufficient args",
        },
        testEntry_t{
            ApiIDDoSysShutdown,
            []Param_t{ /* Incorrect arg type */
                Param_t{ANONYMOUS, false, nil},
            },
            []result_t{NON_NIL_ERROR},
            "incorrect arg",
        },
        testEntry_t{
            ApiIDDoSysShutdown,
            []Param_t{ /* Timeout as 2 secs */
                Param_t{ANONYMOUS, 2, nil},
            },
            []result_t{NIL_ERROR},
            "Initiate system shutdown",
        },
        testEntry_t{
            ApiIDGetPubChannel, /* Simulate publish from engine */
            []Param_t{
                Param_t{"chType_E", nil, nil}, /* pub for events */
                Param_t{"prod_E", nil, nil},   /* from engine */
                EMPTY_STRING,
            },
            []result_t{NIL_ANY, NON_NIL_ERROR},
            "Fail to get after sys shutdown",
        },
        testEntry_t{
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{"chType_E", nil, nil},
                Param_t{"prod_E", nil, nil},
                EMPTY_STRING,
            },
            []result_t{NIL_ANY, NIL_ANY, NON_NIL_ERROR},
            "Fail to get after sys shutdown",
        },
        testEntry_t{
            ApiIDRunPubSubProxy,
            []Param_t{Param_t{"chType_E", nil, nil}},
            []result_t{NIL_ANY, NON_NIL_ERROR},
            "Fail to get after sys shutdown",
        },
        testEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request:Hello world"), nil},
            },
            []result_t{NIL_ANY, NON_NIL_ERROR},
            "Fail to send after sys shutdown",
        },
        testEntry_t{
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_EVENTS, nil}},
            []result_t{NIL_ANY, NIL_ANY, NON_NIL_ERROR},
            "Fail to run after sys shutdown",
        },
        testEntry_t{
            ApiIDInitSysShutdown,
            []Param_t{ Param_t{ANONYMOUS, 2, nil} }, /* redundant arg */
            []result_t{NON_NIL_ERROR},
            "excess args to fail",
        },
        testEntry_t{
            ApiIDInitSysShutdown,
            []Param_t{}, /* no args required */
            []result_t{NIL_ERROR},
            "Initialized for a clean state, just in case for subsequent tests.",
        },
        PAUSE1,
        TELE_IDLE_CHECK,
    },
}
