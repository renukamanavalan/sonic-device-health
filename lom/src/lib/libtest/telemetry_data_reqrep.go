package libtest

import (
    . "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

var pubSubReqRepSuite = ScriptSuite_t{
    Id:          "pubSubReqRepSuite",
    Description: "Test pub sub for request & response - Good run",
    Entries: []ScriptEntry_t{
        ScriptEntry_t{
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
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_0", tele.ClientReq_t("request:Hello world"), nil},
            },
            []Result_t{
                Result_t{"chClientRes-0", nil, ValidateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "Send a request as if from client",
        },
        /* Simulate Server read req and respond */
        ScriptEntry_t{
            ApiIDReadClientRequest, /* Server read req */
            []Param_t{
                Param_t{"chSerReq-0", nil, nil}, /* Get chRead_0 from cache */
                Param_t{ANONYMOUS, 1, nil},      /* timeout = 1 second */
            },
            []Result_t{
                Result_t{"req_0", nil, nil}, /* Validate against cache val for req_0 */
                NIL_ERROR,
            },
            "As server read your request",
        },
        ScriptEntry_t{
            ApiIDSendClientResponse, /* Server Write response */
            []Param_t{
                Param_t{"chSerRes-0", nil, nil},                     /* Use chan from cache */
                Param_t{"res-0", tele.ServerRes_t("resp: ok"), nil}, /* Save written data in cache */
                Param_t{ANONYMOUS, 1, nil},                          /* timeout = 1 second */
            },
            []Result_t{NIL_ERROR},
            "As server write your response",
        },
        ScriptEntry_t{
            ApiIDReadClientResponse, /* Client read its response */
            []Param_t{
                Param_t{"chClientRes-0", nil, nil}, /* Get chan from cache */
                Param_t{ANONYMOUS, 1, nil},         /* timeout = 1 second */
            },
            []Result_t{
                Result_t{"res-0", nil, nil}, /* Validate against cache val for res_0 */
                NIL_ERROR,
            },
            "read from sub channel created above",
        },
        ScriptEntry_t{
            ApiIDSendClientRequest,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"req_1", tele.ClientReq_t("request:Hello Mars"), nil},
            },
            []Result_t{
                Result_t{"chClientRes-0", nil, ValidateNonNil}, /* chan to read response */
                NIL_ERROR,
            },
            "Send a request as if from client",
        },
        /* Simulate Server read req and respond */
        ScriptEntry_t{
            ApiIDReadClientRequest, /* Server read req */
            []Param_t{
                Param_t{"chSerReq-0", nil, nil}, /* Get chRead_0 from cache */
                Param_t{ANONYMOUS, 1, nil},      /* timeout = 1 second */
            },
            []Result_t{
                Result_t{"req_1", nil, nil}, /* Validate against cache val for req_1 */
                NIL_ERROR,
            },
            "As server read your request",
        },
        ScriptEntry_t{
            ApiIDSendClientResponse, /* Server Write response */
            []Param_t{
                Param_t{"chSerRes-0", nil, nil},                           /* Use chan from cache */
                Param_t{"res_1", tele.ServerRes_t("resp: Hi Mars!"), nil}, /* Save written data in cache */
                Param_t{ANONYMOUS, 1, nil},                                /* timeout = 1 second */
            },
            []Result_t{NIL_ERROR},
            "As server write your response",
        },
        ScriptEntry_t{
            ApiIDReadClientResponse, /* Client read its response */
            []Param_t{
                Param_t{"chClientRes-0", nil, nil}, /* Get chan from cache */
                Param_t{ANONYMOUS, 1, nil},         /* timeout = 1 second */
            },
            []Result_t{
                Result_t{"res_1", nil, nil}, /* Validate against cache val for res_0 */
                NIL_ERROR,
            },
            "read from sub channel created above",
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
            ApiIDCloseRequestChannel, /* explicit request to close for req channel */
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
            },
            []Result_t{
                NIL_ERROR,
            },
            "Close channel created for client requests.",
        },
        PAUSE2,
        TELE_IDLE_CHECK,
    },
}
