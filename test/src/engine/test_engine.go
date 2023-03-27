package engine


import (
    "fmt"
    . "lib/lomcommon"
    . "lib/lomipc"
)

const CLIENT_0 = "client-0"
const CLIENT_1 = "client-1"
const CLIENT_2 = "client-2"

/*
 *  Actions.conf
 */
 var actions_conf = `{ "actions": [
        { "name": "Detect-0" },
        { "name": "Safety-chk-0", Timeout: 1},
        { "name": "Mitigate-0", Timeout: 6},
        { "name": "Detect-1" },
        { "name": "Safety-chk-1", Timeout: 7},
        { "name": "Mitigate-1", Timeout: 8},
        { "name": "Detect-2" },
        { "name": "Disabled-0", Disable: false},
        ] }`


var bindings_conf = `{ "bindings": [
    {
        "name": "bind-0", 
        "priority": 0,
        "Timeout": 2,
        "actions": [
            {"name": "Detect-0" },
            {"name": "Safety-chk-0", "sequence": 1 },
            {"name": "Mitigate-0", "sequence": 2 },
        ],
    },
    {
        "name": "bind-1", 
        "priority": 1,
        "Timeout": 19,
        "actions": [
            {"name": "Detect-1" },
            {"name": "Safety-chk-1", "sequence": 1 },
            {"name": "Mitigate-1", "sequence": 2 },
        ],
    },
    {
        "name": "bind-2", 
        "priority": 0,
        "Timeout": 1,
        "actions": [
            {"name": "Detect-2" },
            {"name": "Safety-chk-0", "sequence": 1 },
            {"name": "Safety-chk-2", "sequence": 2 },
            {"name": "Mitigate-2", "sequence": 3 },
        ],
    },
]}


/*
 *  Mock PublishEventAPI 
 *  This test code combines unit test & functional test - Two in one shot
 *
 *  Scenarios:
 *      Register/de-register:
 *          1.  register empty client - Fails
 *          2.  register client CLIENT_0 - Succeeds
 *          2.  re-register client CLIENT_0 - Succeeds
 *          3.  register action with empty name ("") under CLIENT_0 client - fails
 *          4.  register action "Detect-0" under CLIENT_0 client - Succeeds
 *          4.  re-register action "Detect-0" under CLIENT_0 client - Succeeds
 *          x.  register client CLIENT_1            
 *          4.  re-register action "Detect-0" under CLIENT_1 client - fails
 *          x.  register "Safety-chk-0", "Mitigate-0", "Mitigate-2" under CLIENT_0
 *          x.  register ""Detect-1", "Safety-chk-1", "Mitigate-1", "Detect-2" & "Mitigate-2" under CLIENT_1
 *          x.  register "Disabled-0" nder CLIENT_0 client - fails
 *          x.  verify all registrations
 *
 *      Scenarios:
 *      Initial requests
 *          x.  Expect requests for "Detect-0", "Detect-1" & "Detect-2"
 *
 *      One proper sequence
 *          x. "Detect-0" returns good. Expect "Safety-chk-0"; return good; expect"Mitigate-0"; return good
 *              verify publish responses
 *          x. Expect request for detect-0
 *          x. "Detect-0" returns good. Expect "Safety-chk-0"; return good; expect"Mitigate-0"; return fail
 *              verify publish responses
 *          x. "Detect-0" returns good. Expect "Safety-chk-0"; return fail
 *              verify publish responses
 *          x. "Detect-0" returns fail.
 *              verify publish responses
 *          x. "Detect-0" returns good. Expect "Safety-chk-0"; return good; expect"Mitigate-0"; sleep 3s; mmitigate-0  responds; seq timeout
 *              verify publish responses
 *          x. "Detect-0" returns good. Expect "Safety-chk-0"; Sleep forever; req expect to timeout
 *              verify publish responses
 *          x. "Detect-2" & "Detect-1" returns good; But "Safety-chk-0" busy. bind-2 timesout.
 *          x. Expect "Safety-chk-1" call; return good; expect "Mitigate-1"; return good
 *              verify publish responses
 *          x. Trigger "Safety-chk-0" respond
 *          x. "Detect-2" return good; "Safety-chk-0"; good; "Safety-chk-2"; good; "Mitigate-2"; good; seq complete
 *              verify publish responses
 *
 *          x. "Detect-0" good; safety-check-0 sleeps; bind-0 timesout.
 *          x. "Detect-0" good; safety-check-0 not called; bind-0 timesout.
 *          x. De-register safety-check-0 & re-register
 *          x. "Detect-0" good; safety-check-0 good; mitigate-0 good; bind-0 good.
 *              verify publish responses
 *          x.  NotifyHearbeat for "Detect-0"
 *              Verify responnse
 *          x.  NotifyHearbeat for "xyz" non-existing
 *              Verify responnse
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 *
 */


