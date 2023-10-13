package libtest

import (
    . "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

/*
 * Data driven test FW.
 *
 * A test entry {
 *  Identifies API by API ID
 *  Each arg is represented by param_t struct
 *  Each return value is expressed by Result_t struct
 *
 * Named param or result entity is saved in cache.
 * Subseqent param/result could refer value from the cache.
 * A cache is per test suite
 *
 * A test suite is a collection of tests.
 *
 */

/* Test Data for telemetry */

var pubSubSuite = ScriptSuite_t{
    Id:          "pubSubSuite",
    Description: "Test pub sub for events - Good run",
    Entries: []ScriptEntry_t{
        ScriptEntry_t{
            ApiIDRunPubSubProxy,
            []Param_t{Param_t{"chType_1", tele.CHANNEL_TYPE_EVENTS, nil}},
            []Result_t{
                Result_t{"chPrxyClose-0", nil, ValidateNonNil}, /* Save in cache */
                NIL_ERROR, /*Expect nil error */
            },
            "Rub pubsub proxy, required to bind publishers & subscribers",
        },
        ScriptEntry_t{
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"prod_0", tele.CHANNEL_PRODUCER_EMPTY, nil},
                EMPTY_STRING,
            },
            []Result_t{
                Result_t{"chRead-0", nil, ValidateNonNil},     /* Save in cache */
                Result_t{"chSubClose-0", nil, ValidateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get sub channel for same type as proxy above",
        },
        ScriptEntry_t{
            ApiIDGetPubChannel,
            []Param_t{
                Param_t{"chType_1", nil, nil}, /* Fetch chType_1 from cache */
                Param_t{"prod_1", tele.CHANNEL_PRODUCER_ENGINE, nil},
                EMPTY_STRING,
            },
            []Result_t{
                Result_t{"chWrite-0", nil, ValidateNonNil}, /* Save in cache */
                Result_t{ANONYMOUS, nil, ValidateNil},
            },
            "Get pub channel for same type as proxy above",
        },
        ScriptEntry_t{
            ApiIDWriteJsonStringsChannel,
            []Param_t{
                Param_t{"chWrite-0", nil, nil}, /* Use chan from cache */
                Param_t{"pub_0", []tele.JsonString_t{
                    tele.JsonString_t("Hello World!")}, nil}, /* Save written data in cache */
                Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil},
            }, /*Expect nil error */
            "Write into pub channel created above",
        },
        ScriptEntry_t{
            ApiIDReadJsonStringsChannel,
            []Param_t{
                Param_t{"chRead-0", nil, nil}, /* Get chRead_0 from cache */
                Param_t{ANONYMOUS, 1, nil},    /* read cnt = 1 */
                Param_t{ANONYMOUS, 1, nil},    /* timeout = 1 second */
            },
            []Result_t{
                Result_t{"pub_0", nil, nil}, /* Validate against cache val for pub_0 */
                Result_t{ANONYMOUS, nil, ValidateNil},
            },
            "read from sub channel created above",
        },
        ScriptEntry_t{
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chWrite-0", nil, nil}, /* Get chWrite_0 from cache */
            },
            []Result_t{NIL_ERROR},
            "Close pub chennel",
        },
        ScriptEntry_t{
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chSubClose-0", nil, nil}, /* Get from cache */
            },
            []Result_t{NIL_ERROR},
            "Close sub chennel",
        },
        ScriptEntry_t{
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chPrxyClose-0", nil, nil}, /* Get from cache */
            },
            []Result_t{NIL_ERROR},
            "Close proxy chennel",
        },
        PAUSE2,
        TELE_IDLE_CHECK,
    },
}

var pubSubMultiSuite = ScriptSuite_t{
    Id:          "pubSubMultiSuite",
    Description: "Test multi pub sub for events - Good run",
    Entries: []ScriptEntry_t{
        ScriptEntry_t{
            ApiIDRunPubSubProxy,
            []Param_t{Param_t{"chType_C", tele.CHANNEL_TYPE_COUNTERS, nil}},
            []Result_t{
                Result_t{"chPrxyClose-C", nil, ValidateNonNil}, /* Save in cache */
                NIL_ERROR, /*Expect nil error */
            },
            "Rub pubsub proxy, required to bind publishers & subscribers",
        },
        ScriptEntry_t{
            ApiIDRunPubSubProxy,
            []Param_t{Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, nil}},
            []Result_t{
                Result_t{"chPrxyClose-E", nil, ValidateNonNil}, /* Save in cache */
                NIL_ERROR, /*Expect nil error */
            },
            "Rub pubsub proxy, required to bind publishers & subscribers",
        },
        ScriptEntry_t{ /* Get sub channel for events from engine only. */
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{"chType_E", nil, nil}, /* Fetch chType_1 from cache */
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
        ScriptEntry_t{ /* Get sub channel for counters from a plugin-mgr instance */
            ApiIDGetSubChannel,
            []Param_t{
                Param_t{"chType_C", nil, nil}, /* Fetch from cache */
                Param_t{"prod_PM", tele.CHANNEL_PRODUCER_PLMGR, nil},
                Param_t{"PMgr-1", "inst-1", nil},
            },
            []Result_t{
                Result_t{"chRead-C", nil, ValidateNonNil},     /* Save in cache */
                Result_t{"chSubClose-C", nil, ValidateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get sub channel for events from Engine",
        },
        ScriptEntry_t{
            ApiIDGetPubChannel, /* Simulate publish from plugin-mgr instance */
            []Param_t{
                Param_t{"chType_C", nil, nil}, /* pub for counters */
                Param_t{"prod_PM", nil, nil},  /* from Plugin Mgr */
                Param_t{"PMgr-1", nil, nil},   /* instance-1 */
            },
            []Result_t{
                Result_t{"chWrite-C", nil, ValidateNonNil}, /* Save in cache */
                NIL_ERROR,
            },
            "Get pub channel for counters as if from Plugin Mgr",
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
            "Get pub channel for counters as if from Plugin Mgr",
        },
        ScriptEntry_t{
            ApiIDWriteJsonStringsChannel,
            []Param_t{
                Param_t{"chWrite-E", nil, nil}, /* Use chan from cache */
                Param_t{"pub_E", []tele.JsonString_t{
                    tele.JsonString_t("Hello World!")}, nil}, /* Save written data in cache */
                Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil},
            }, /*Expect nil error */
            "Write into pub channel created above",
        },
        ScriptEntry_t{
            ApiIDWriteJsonStringsChannel,
            []Param_t{
                Param_t{"chWrite-C", nil, nil}, /* Use chan from cache */
                Param_t{"pub_C", []tele.JsonString_t{
                    tele.JsonString_t("Some counters")}, nil}, /* Save written data in cache */
                Param_t{ANONYMOUS, 1, nil}, /* timeout = 1 second */
            },
            []Result_t{
                Result_t{ANONYMOUS, nil, ValidateNil},
            }, /*Expect nil error */
            "Write into pub channel created above",
        },
        ScriptEntry_t{
            ApiIDReadJsonStringsChannel,
            []Param_t{
                Param_t{"chRead-C", nil, nil}, /* read counters */
                Param_t{ANONYMOUS, 1, nil},    /* read cnt = 1 */
                Param_t{ANONYMOUS, 1, nil},    /* timeout = 1 second */
            },
            []Result_t{
                Result_t{"pub_C", nil, nil}, /* Validate against cache val for pub_C */
                Result_t{ANONYMOUS, nil, ValidateNil},
            },
            "read from sub channel created above",
        },
        ScriptEntry_t{
            ApiIDReadJsonStringsChannel,
            []Param_t{
                Param_t{"chRead-E", nil, nil}, /* read counters */
                Param_t{ANONYMOUS, 1, nil},    /* read cnt = 1 */
                Param_t{ANONYMOUS, 1, nil},    /* timeout = 1 second */
            },
            []Result_t{
                Result_t{"pub_E", nil, nil}, /* Validate against cache val for pub_E */
                Result_t{ANONYMOUS, nil, ValidateNil},
            },
            "read from sub channel created above",
        },
        ScriptEntry_t{
            ApiIDCloseChannel,
            []Param_t{
                Param_t{"chPrxyClose-C", nil, nil},
                Param_t{"chPrxyClose-E", nil, nil},
                Param_t{"chSubClose-E", nil, nil},
                Param_t{"chSubClose-C", nil, nil},
                Param_t{"chWrite-C", nil, nil},
                Param_t{"chWrite-E", nil, nil},
            },
            []Result_t{NIL_ERROR},
            "Close pub chennel",
        },
        PAUSE2,
        TELE_IDLE_CHECK,
    },
}

var testTelemetrySuites = []*ScriptSuite_t{
    &pubSubSuite,
    &pubSubMultiSuite,
    &pubSubFnSuite,
    &pubSubReqRepSuite,
    &pubSubFailSuite,
    &scriptAPIValidate,
    &scriptAPIValidate_2,
    &pubSubBindFail,
    &pubSubShutdownSuite, /* KEEP this as last suite as it invokes irreversible shutdown */
}
