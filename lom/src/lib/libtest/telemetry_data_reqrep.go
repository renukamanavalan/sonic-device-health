package libtest

import (
    . "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

var pubSubReqRepSuite = testSuite_t{
    id:          "pubSubReqRepSuite",
    description: "Test pub sub for request & response - Good run",
    tests: []testEntry_t{
        testEntry_t{
            ApiIDRegisterServerReqHandler,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, nil}},
            []result_t{
                result_t{"chSerReq-0", nil, validateNonNil}, /* chan for incoming req */
                result_t{"chSerRes-0", nil, validateNonNil}, /* chan for outgoing res */
                NIL_ERROR, /*Expect nil error */
            },
            "Register server handler to process requests and provide responses.",
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
            "Send a request as if from client",
        },
        /* Simulate Server read req and respond */
        testEntry_t{
            ApiIDReadClientRequest, /* Server read req */
            []Param_t{
                Param_t{"chSerReq-0", nil, nil},   /* Get chRead_0 from cache */
                Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
            },
            []result_t{
                result_t{"req_0", nil, nil}, /* Validate against cache val for req_0 */
                NIL_ERROR,
            },
            "As server read your request",
        },
        testEntry_t{
            ApiIDSendClientResponse, /* Server Write response */
            []Param_t{
                Param_t{"chSerRes-0", nil, nil},                     /* Use chan from cache */
                Param_t{"res-0", tele.ServerRes_t("resp: ok"), nil}, /* Save written data in cache */
                Param_t{ANONYMOUS, 1, nil},                   /* timeout = 1 second */
            },
            []result_t{NIL_ERROR},
            "As server write your response",
        },
        testEntry_t{
            ApiIDReadClientResponse, /* Client read its response */
            []Param_t{
                Param_t{"chClientRes-0", nil, nil}, /* Get chan from cache */
                Param_t{ANONYMOUS, 1, nil},  /* timeout = 1 second */
            },
            []result_t{
                result_t{"res-0", nil, nil}, /* Validate against cache val for res_0 */
                NIL_ERROR,
            },
            "read from sub channel created above",
        },
        testEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_1", tele.ClientReq_t("request:Hello Mars"), nil},
            },
            []result_t{
                result_t{"chClientRes-0", nil, validateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "Send a request as if from client",
        },
        /* Simulate Server read req and respond */
        testEntry_t{
            ApiIDReadClientRequest, /* Server read req */
            []Param_t{
                Param_t{"chSerReq-0", nil, nil},   /* Get chRead_0 from cache */
                Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
            },
            []result_t{
                result_t{"req_1", nil, nil}, /* Validate against cache val for req_1 */
                NIL_ERROR,
            },
            "As server read your request",
        },
        testEntry_t{
            ApiIDSendClientResponse, /* Server Write response */
            []Param_t{
                Param_t{"chSerRes-0", nil, nil},                           /* Use chan from cache */
                Param_t{"res_1", tele.ServerRes_t("resp: Hi Mars!"), nil}, /* Save written data in cache */
                Param_t{ANONYMOUS, 1, nil},                         /* timeout = 1 second */
            },
            []result_t{NIL_ERROR},
            "As server write your response",
        },
        testEntry_t{
            ApiIDReadClientResponse, /* Client read its response */
            []Param_t{
                Param_t{"chClientRes-0", nil, nil}, /* Get chan from cache */
                Param_t{ANONYMOUS, 1, nil},  /* timeout = 1 second */
            },
            []result_t{
                result_t{"res_1", nil, nil}, /* Validate against cache val for res_0 */
                NIL_ERROR,
            },
            "read from sub channel created above",
        },
        testEntry_t{
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chSerRes-0", nil, nil}, /* Close server handler */
            },
            []result_t{NIL_ERROR},
            "Close server request handler via closing this channel.",
        },
        testEntry_t{
            ApiIDCloseRequestChannel, /* explicit request to close for req channel */
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
            },
            []result_t{
                NIL_ERROR,
            },
            "Close channel created for client requests.",
        },
        PAUSE2,
        TELE_IDLE_CHECK,
    },
}
