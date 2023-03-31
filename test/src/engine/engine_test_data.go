package engine

import (
    "fmt"
    . "lib/lomcommon"
    . "lib/lomipc"
)


/* Only test data */
type testEntry_t struct {
    id          clientAPIID
    clTx        string          /* Which Tx to use*/
    seqId       int             /* The context to use for save/restore results per seq */
    args        []any
    result      []any
    failed      bool            /* True if expected to fail. */
    desc        string
}

func (p *testEntry_t) toStr() string {
    s := ""
    switch p.id {
    case REG_CLIENT:
        s = "REG_CLIENT"
    case REG_ACTION:
        s = "REG_ACTION"
    case DEREG_CLIENT:
        s = "DEREG_CLIENT"
    case DEREG_ACTION:
        s = "DEREG_ACTION"
    case RECV_REQ:
        s = "RECV_REQ"
    case SEND_RES:
        s = "SEND_RES"
    case SHUTDOWN:
        s = "SHUTDOWN"
    case NOTIFY_HB:
        s = "NOTIFY_HB"
    case CHK_ACTIV_REQ:
        s = "CHK_ACTIV_REQ"
    case CHK_REG_ACTIONS:
        s = "CHK_REG_ACTIONS"
    default:
        s = fmt.Sprintf("UNK(%d)", p.id)
    }
    return fmt.Sprintf("%s:%s: args:(%v) res(%v) failed(%v)",
            p.clTx, s, p.args, p.result, p.failed)
}


type testEntriesList_t  map[int]testEntry_t
type testCollectionId_t string
type testCollectionEntry_t struct {
    preSetup    []testCollectionId_t    /* List to run as pre-setup */
    testEntries testEntriesList_t       /* tests to run */
    postCleanup []testCollectionId_t 
    desc        string
}

type testCollections_t  map[testCollectionId_t]*testCollectionEntry_t

/* Order of test runs by ID */
type testRunList_t []testCollectionId_t
var testRunList = testRunList_t {
    "registrations_test", 
    "seq_success",
    "seq_mit_fail",
    "seq_safety_fail",
    "detect_fail",
}

type registrations_t map[string][]string

/* Test scenario expectations */
var expRegistrations = []registrations_t {
    {    /* Map of client vs actions */
        CLIENT_0: []string { "Detect-0", "Safety-chk-0", "Mitigate-0", "Mitigate-2" },
        CLIENT_1: []string { "Detect-1", "Safety-chk-1", "Mitigate-1", "Detect-2", "Safety-chk-2" },
    },
    {    /* Map of client vs actions */
        CLIENT_0: []string { "Detect-0", "Safety-chk-0" },
        CLIENT_1: []string { "Detect-1", "Safety-chk-1", "Mitigate-1" },
    },
    {    /* Map of client vs actions */
        CLIENT_1: []string { "Detect-1", "Safety-chk-1", "Mitigate-1" },
    },
    {    /* Map of client vs actions */ },
}

type activeActionsList_t map[string]ActiveActionInfo_t
var expActiveActions = make([]activeActionsList_t, len(expRegistrations))

func initActive() {
    if  len(expActiveActions[0]) > 0 {
        return
    }

    cfg := GetConfigMgr()

    for i, rl := range expRegistrations {
        expActiveActions[i] = make(activeActionsList_t)
        lst := expActiveActions[i]
        for cl, v := range rl {
            for _, a := range v {
                if _, ok := lst[a]; ok {
                    LogPanic("Duplicate action in expRegistrations[%d] cl(%s) a(%s)", i, cl, a)
                }
                if c, e := cfg.GetActionConfig(a); e != nil {
                    LogPanic("Failed to get action config for (%s)", a)
                } else {
                    lst[a] = ActiveActionInfo_t {
                        Action: a, Client: cl, Timeout: c.Timeout, }
                }
            }
        }
    }
}

var testCollections = make(testCollections_t)

func init() {
    testCollections["registrations_test"] = &testCollectionEntry_t {
        desc: "Register & De-register testing",
        preSetup: []testCollectionId_t{},    /* none */
        testEntries: testEntriesList_t {
            0: {
                id: REG_ACTION,
                clTx: "",
                args: []any{"xyz"},
                failed: true,
                desc: "RegisterAction: Fail as before register client",
            },
            1: {
                id: REG_CLIENT,
                clTx: "iX",
                args: []any{EMPTY_STR},
                failed: true,
                desc: "RegisterClient: Fail for empty name",
            },
            2: {
                id: REG_CLIENT,
                clTx: "Bogus",
                args: []any{CLIENT_0},
                failed: false,
                desc: "RegisterClient to succeed",
            },
            3: {
                id: REG_CLIENT,
                clTx: "Bogus",
                args: []any{CLIENT_0},
                failed: true,
                desc: "register-client: Fail duplicate on same transport",
            },
            4: {
                id: REG_CLIENT,
                clTx: CLIENT_0,             /* re-reg under new Tx. So succeed" */
                args: []any{CLIENT_0},
                failed: false,
                desc: "RegClient re-reg on new tx to succeed",
            },
            5: {
                id: REG_ACTION,
                clTx: CLIENT_0,
                args: []any{""},
                failed: true,
                desc: "RegisterAction fail for empty name",
            },
            6: {
                id: REG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Detect-0"},
                failed: false,
                desc: "RegisterAction client-0/detect-0 succeeds",
            },
            7: {
                id: REG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Detect-0"},
                failed: false,
                desc: "Re-registerAction succeeds",
            },
            8: {
                id: REG_CLIENT,
                clTx: CLIENT_1,
                args: []any{CLIENT_1},
                failed: false,
                desc: "second Client reg to succeed",
            },
            9: {
                id: REG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Detect-0"},
                failed: false,
                desc: "RegAction: Succeed duplicate register on different client",
            },
            10: {
                id: REG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Detect-0"},
                failed: false,
                desc: "Duplicate action register on different client",
            },
            11: {
                id: REG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Mitigate-0"},
                failed: false,
                desc: "action register succeed",
            },
            12: {
                id: REG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Mitigate-2"},
                failed: false,
                desc: "action register succeed",
            },
            13: {
                id: REG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Safety-chk-0"},
                failed: false,
                desc: "action register succeed",
            },
            14: {
                id: REG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Detect-1"},
                failed: false,
                desc: "action register succeed",
            },
            15: {
                id: REG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Safety-chk-1"},
                failed: false,
                desc: "action register succeed",
            },
            16: {
                id: REG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Mitigate-1"},
                failed: false,
                desc: "action register succeed",
            },
            17: {
                id: REG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Detect-2"},
                failed: false,
                desc: "action register succeed",
            },
            18: {
                id: REG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Safety-chk-2"},
                failed: false,
                desc: "action register succeed",
            },
            19: {
                id: REG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Disabled-0"},
                failed: true,
                desc: "action register fail for disabled",
            },
            20: {
                id: CHK_REG_ACTIONS,
                clTx: "",               /* Local verification */
                args: []any{0},         /* index into expRegistrations & expActiveActions */
                desc: "Verify local cache to succeed",
            },
            21: {
                id: DEREG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Detect-2"},
                failed: false,
                desc: "action deregister succeed",
            },
            22: {
                id: DEREG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Safety-chk-2"},
                failed: false,
                desc: "action deregister succeed",
            },
            23: {
                id: DEREG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Mitigate-0"},
                failed: false,
                desc: "action deregister succeed",
            },
            24: {
                id: DEREG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Mitigate-2"},
                failed: false,
                desc: "action deregister succeed",
            },
            25: {
                id: DEREG_ACTION,
                clTx: CLIENT_0,
                args: []any{""},
                desc: "action deregister succeed for empty",
            },
            26: {
                id: DEREG_ACTION,
                clTx: CLIENT_0,
                args: []any{"XXX"},
                desc: "action deregister succeed for non-existing",
            },
            27: {
                id: CHK_REG_ACTIONS,
                clTx: "",               /* Local verification */
                args: []any{1},         /* index into expRegistrations & expActiveActions */
                desc: "Verify local cache to succeed",
            },
            28: {
                id: DEREG_CLIENT,
                clTx: CLIENT_0,
                desc: "action deregister client succeed",
            },
            29: {
                id: CHK_REG_ACTIONS,
                clTx: "",               /* Local verification */
                args: []any{2},         /* index into expRegistrations & expActiveActions */
                desc: "Verify local cache to succeed",
            },
            30: {
                id: DEREG_CLIENT,
                clTx: CLIENT_1,
                desc: "action deregister client succeed",
            },
            31: {
                id: CHK_REG_ACTIONS,
                clTx: "",               /* Local verification */
                args: []any{3},         /* index into expRegistrations & expActiveActions */
                desc: "Verify local cache to succeed",
            },
        },
        postCleanup: []testCollectionId_t{}, /* none */
    }

    testCollections["registrations_cleanup"] = &testCollectionEntry_t {
        /*
         * This is for shared use by other collections for cleanup.
         * Running this individually won't work, as this has de-reg only
         */
        desc: "Post cleanup of all registrations",
        preSetup: []testCollectionId_t{},    /* none */
        testEntries: testEntriesList_t {
            0: {
                id: DEREG_CLIENT,
                clTx: CLIENT_0,
                desc: "action deregister client succeed",
            },
            1: {
                id: DEREG_CLIENT,
                clTx: CLIENT_1,
                desc: "action deregister client succeed",
            },
            2: {
                id: CHK_REG_ACTIONS,
                clTx: "",               /* Local verification */
                args: []any{3},         /* index into expRegistrations & expActiveActions */
                desc: "Verify local cache to succeed",
            },
        },
        postCleanup: []testCollectionId_t{}, /* none */
    }

    testCollections["registrations_setup"] = &testCollectionEntry_t {
        /*
         * This is for shared use by other collections for setup.
         * Running this individually would work, but leave registrations behind as 
         * orphaned, as test code drops all transports at the end of collection
         * run and hence these registrations are never reachable.
         */
        desc: "Pre setup of all registrations",
        preSetup: []testCollectionId_t{},    /* none */
        testEntries: testEntriesList_t {
            100: {
                id: REG_CLIENT,
                clTx: CLIENT_0,             /* re-reg under new Tx. So succeed" */
                args: []any{CLIENT_0},
                failed: false,
                desc: "RegClient to succeed",
            },
            102: {
                id: REG_CLIENT,
                clTx: CLIENT_1,
                args: []any{CLIENT_1},
                failed: false,
                desc: "second Client reg to succeed",
            },
            104: {
                id: REG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Detect-0"},
                failed: false,
                desc: "Reg Action to succeed",
            },
            106: {
                id: REG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Mitigate-0"},
                failed: false,
                desc: "action register succeed",
            },
            108: {
                id: REG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Mitigate-2"},
            
                desc: "action register succeed",
            },
            110: {
                id: REG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Safety-chk-0"},
                failed: false,
                desc: "action register succeed",
            },
            114: {
                id: REG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Detect-1"},
                failed: false,
                desc: "action register succeed",
            },
            116: {
                id: REG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Safety-chk-1"},
                failed: false,
                desc: "action register succeed",
            },
            118: {
                id: REG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Mitigate-1"},
                failed: false,
                desc: "action register succeed",
            },
            120: {
                id: REG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Detect-2"},
                failed: false,
                desc: "action register succeed",
            },
            122: {
                id: REG_ACTION,
                clTx: CLIENT_1,
                args: []any{"Safety-chk-2"},
                failed: false,
                desc: "action register succeed",
            },
            124: {
                id: CHK_REG_ACTIONS,
                clTx: "",               /* Local verification */
                args: []any{0},
                desc: "Verify local cache to succeed",
            },

        },
        postCleanup: []testCollectionId_t{}, /* none */
    }

    testCollections["seq_success"] = &testCollectionEntry_t {
        desc: "A successful Sequence",
        preSetup: []testCollectionId_t{"registrations_setup"},    /* none */
        testEntries: testEntriesList_t {
            /* Requests are expected in the same order as registration */
            140: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 1,               /* Use non-zero, default is 0. Make it explicit */
                result: []any { &ActionRequestData { Action: "Detect-0"} },
                desc: "Read server request for Detect-0",
            },
            142: {
                id: RECV_REQ,
                clTx: CLIENT_1,
                seqId: 2,
                result: []any { &ActionRequestData { Action: "Detect-1"} },
                desc: "Read server request for Detect-1",
            },
            144: {
                id: RECV_REQ,
                clTx: CLIENT_1,
                seqId: 3,
                result: []any { &ActionRequestData { Action: "Detect-2"} },
                desc: "Read server request for Detect-2",
            },
            /* Test one full successful sequence. Detect-0 -> chk-0 -> Mit-0 */
            /* registrations_setup ha already verified initial requests received. */
            150: {
                id: SEND_RES,
                clTx: CLIENT_0,
                seqId: 1,
                args: []any {&ActionResponseData{Action: "Detect-0", AnomalyKey: "Key-Detect-0", Response: "Detect-0 detected",}},
                desc: "Send res for detect0",
            },
            152: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 1,
                result: []any { &ActionRequestData { Action: "Safety-chk-0", Timeout: 1} },
                desc: "Read server request for Safety-check-0",
            },
            154: {
                id: SEND_RES,
                clTx: CLIENT_0,
                seqId: 1,
                args: []any {&ActionResponseData{Action: "Safety-chk-0", Response: "Safety-chk-0 passed",}},
                desc: "Send res for safety-chk-0",
            },
            156: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 1,
                result: []any { &ActionRequestData { Action: "Mitigate-0", Timeout: -1} },
                desc: "Read server request for Safety-check-0",
            },
            158: {
                id: SEND_RES,
                clTx: CLIENT_0,
                seqId: 1,
                args: []any {&ActionResponseData{Action: "Mitigate-0", Response: "Mitigate-0 passed",}},
                desc: "Send res for Mitigate-0",
            },
            160: {
                id: SEQ_COMPLETE,
                seqId: 1,
                desc: "Verify seq complete",
            },
            162: {
                id: NOTIFY_HB,
                clTx: CLIENT_0,
                args: []any {"XYZ", "Detect-0", "Mitigate-0"},
                result: []any {"Detect-0", "Mitigate-0"},
                desc: "Notify heartbeats valid & invalid names",
            },
            164: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 4,               /* Use non-zero, default is 0. Make it explicit */
                result: []any { &ActionRequestData { Action: "Detect-0"} },
                desc: "Read server request for Detect-0",
            },
            166: {
                /* Local engine level verification */
                id: CHK_ACTIV_REQ,
                args: []any {"Detect-0", "Detect-1", "Detect-2"},
                desc: "Verify active requests only for actions in args",
            },
        },
        postCleanup: []testCollectionId_t{"registrations_cleanup"}, /* none */
    }

    testCollections["seq_mit_fail"] = &testCollectionEntry_t {
        desc: "A failed Sequence in last action",
        preSetup: []testCollectionId_t{"registrations_setup"},    /* none */
        testEntries: testEntriesList_t {
            /* Requests are expected in the same order as registration */
            140: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 1,               /* Use non-zero, default is 0. Make it explicit */
                result: []any { &ActionRequestData { Action: "Detect-0"} },
                desc: "Read server request for Detect-0",
            },
            142: {
                id: RECV_REQ,
                clTx: CLIENT_1,
                seqId: 2,
                result: []any { &ActionRequestData { Action: "Detect-1"} },
                desc: "Read server request for Detect-1",
            },
            144: {
                id: RECV_REQ,
                clTx: CLIENT_1,
                seqId: 3,
                result: []any { &ActionRequestData { Action: "Detect-2"} },
                desc: "Read server request for Detect-2",
            },
            /* Test one failed sequence at last action. Detect-0 -> chk-0 -> Mit-0 (fail) */
            /* registrations_setup ha already verified initial requests received. */
            150: {
                id: SEND_RES,
                clTx: CLIENT_0,
                seqId: 1,
                args: []any {&ActionResponseData{Action: "Detect-0", AnomalyKey: "Key-Detect-0", Response: "Detect-0 detected",}},
                desc: "Send res for detect0",
            },
            152: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 1,
                result: []any { &ActionRequestData { Action: "Safety-chk-0", Timeout: 1} },
                desc: "Read server request for Safety-check-0",
            },
            154: {
                id: SEND_RES,
                clTx: CLIENT_0,
                seqId: 1,
                args: []any {&ActionResponseData{Action: "Safety-chk-0", Response: "Safety-chk-0 passed",}},
                desc: "Send res for safety-chk-0",
            },
            156: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 1,
                result: []any { &ActionRequestData { Action: "Mitigate-0", Timeout: -1} },
                desc: "Read server request for Safety-check-0",
            },
            158: {
                id: SEND_RES,
                clTx: CLIENT_0,
                seqId: 1,
                args: []any {&ActionResponseData{Action: "Mitigate-0", ResultCode: 120, ResultStr: "Blah Blah",}},
                desc: "Send res for Mitigate-0",
            },
            160: {
                id: SEQ_COMPLETE,
                args: []any {&ActionResponseData{Action: "Mitigate-0", ResultCode: 120, ResultStr: "Blah Blah",}},
                seqId: 1,
                desc: "Verify seq complete",
            },
            162: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 4,               /* Use non-zero, default is 0. Make it explicit */
                result: []any { &ActionRequestData { Action: "Detect-0"} },
                desc: "Read server request for Detect-0",
            },
            164: {
                /* Local engine level verification */
                id: CHK_ACTIV_REQ,
                args: []any {"Detect-0", "Detect-1", "Detect-2"},
                desc: "Verify active requests only for actions in args",
            },
        },
        postCleanup: []testCollectionId_t{"registrations_cleanup"}, /* none */
    }

    testCollections["seq_safety_fail"] = &testCollectionEntry_t {
        desc: "A failed Sequence in second action",
        preSetup: []testCollectionId_t{"registrations_setup"},    /* none */
        testEntries: testEntriesList_t {
            /* Requests are expected in the same order as registration */
            140: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 1,               /* Use non-zero, default is 0. Make it explicit */
                result: []any { &ActionRequestData { Action: "Detect-0"} },
                desc: "Read server request for Detect-0",
            },
            142: {
                id: RECV_REQ,
                clTx: CLIENT_1,
                seqId: 2,
                result: []any { &ActionRequestData { Action: "Detect-1"} },
                desc: "Read server request for Detect-1",
            },
            144: {
                id: RECV_REQ,
                clTx: CLIENT_1,
                seqId: 3,
                result: []any { &ActionRequestData { Action: "Detect-2"} },
                desc: "Read server request for Detect-2",
            },
            /* Test one failed sequence at last action. Detect-0 -> chk-0 -> Mit-0 (fail) */
            /* registrations_setup ha already verified initial requests received. */
            150: {
                id: SEND_RES,
                clTx: CLIENT_0,
                seqId: 1,
                args: []any {&ActionResponseData{Action: "Detect-0", AnomalyKey: "Key-Detect-0", Response: "Detect-0 detected",}},
                desc: "Send res for detect0",
            },
            152: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 1,
                result: []any { &ActionRequestData { Action: "Safety-chk-0", Timeout: 1} },
                desc: "Read server request for Safety-check-0",
            },
            154: {
                id: SEND_RES,
                clTx: CLIENT_0,
                seqId: 1,
                args: []any {&ActionResponseData{Action: "Safety-chk-0", ResultCode: 120, ResultStr: "Safe not",}},
                desc: "Send res for safety-chk-0",
            },
            160: {
                id: SEQ_COMPLETE,
                args: []any {&ActionResponseData{ResultCode: 120, ResultStr: "Safe not",}},
                seqId: 1,
                desc: "Verify seq complete",
            },
            162: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 4,               /* Use non-zero, default is 0. Make it explicit */
                result: []any { &ActionRequestData { Action: "Detect-0"} },
                desc: "Read server request for Detect-0",
            },
            164: {
                /* Local engine level verification */
                id: CHK_ACTIV_REQ,
                args: []any {"Detect-0", "Detect-1", "Detect-2"},
                desc: "Verify active requests only for actions in args",
            },
        },
        postCleanup: []testCollectionId_t{"registrations_cleanup"}, /* none */
    }

    testCollections["detect_fail"] = &testCollectionEntry_t {
        desc: "First/detection action fails",
        preSetup: []testCollectionId_t{"registrations_setup"},    /* none */
        testEntries: testEntriesList_t {
            /* Requests are expected in the same order as registration */
            140: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 1,               /* Use non-zero, default is 0. Make it explicit */
                result: []any { &ActionRequestData { Action: "Detect-0"} },
                desc: "Read server request for Detect-0",
            },
            142: {
                id: RECV_REQ,
                clTx: CLIENT_1,
                seqId: 2,
                result: []any { &ActionRequestData { Action: "Detect-1"} },
                desc: "Read server request for Detect-1",
            },
            144: {
                id: RECV_REQ,
                clTx: CLIENT_1,
                seqId: 3,
                result: []any { &ActionRequestData { Action: "Detect-2"} },
                desc: "Read server request for Detect-2",
            },
            /* Test one failed sequence at last action. Detect-0 -> chk-0 -> Mit-0 (fail) */
            /* registrations_setup ha already verified initial requests received. */
            150: {
                id: SEND_RES,
                clTx: CLIENT_0,
                seqId: 1,
                args: []any {&ActionResponseData{Action: "Detect-0", ResultCode: 120, ResultStr: "Detect failed",}},
                desc: "Send res for detect0",
            },
            160: {
                id: SEQ_COMPLETE,
                args: []any {&ActionResponseData{ResultCode: 120, ResultStr: "Detect failed",}},
                seqId: 1,
                desc: "Verify seq complete",
            },
            162: {
                id: RECV_REQ,
                clTx: CLIENT_0,
                seqId: 4,               /* Use non-zero, default is 0. Make it explicit */
                result: []any { &ActionRequestData { Action: "Detect-0"} },
                desc: "Read server request for Detect-0",
            },
            164: {
                /* Local engine level verification */
                id: CHK_ACTIV_REQ,
                args: []any {"Detect-0", "Detect-1", "Detect-2"},
                desc: "Verify active requests only for actions in args",
            },
        },
        postCleanup: []testCollectionId_t{"registrations_cleanup"}, /* none */
    }
}
