package engine


/*
 *  Mock PublishEventAPI 
 *  This test code combines unit test & functional test - Two in one shot
 *
 *  Scenarios:
 *      Register/de-register:
 *          1.  register empty client - Fails
 *          2.  register client CLIENT_0 - Succeeds
 *          3.  re-register client CLIENT_0 - Succeeds
 *          4.  register action with empty name ("") under CLIENT_0 client - fails
 *          5.  register action "Detect-0" under CLIENT_0 client - Succeeds
 *          6.  re-register action "Detect-0" under CLIENT_0 client - Succeeds
 *          7.  register client CLIENT_1            
 *          8.  re-register action "Detect-0" under CLIENT_1 client - fails
 *          9.  register "Safety-chk-0", "Mitigate-0", "Mitigate-2" under CLIENT_0
 *          10. register ""Detect-1", "Safety-chk-1", "Mitigate-1", "Detect-2" & "Mitigate-2" under CLIENT_1
 *          11. register "Disabled-0" nder CLIENT_0 client - fails
 *          12. verify all registrations
 *
 *      Scenarios:
 *      Initial requests
 *          1.  Expect requests for "Detect-0", "Detect-1" & "Detect-2"
 *
 *      One proper sequence
 *          2. "Detect-0" returns good. Expect "Safety-chk-0"; return good; expect"Mitigate-0"; return good
 *              verify publish responses
 *          3. Expect request for detect-0
 *          4. "Detect-0" returns good. Expect "Safety-chk-0"; return good; expect"Mitigate-0"; return fail
 *              verify publish responses
 *          5. "Detect-0" returns good. Expect "Safety-chk-0"; return fail
 *              verify publish responses
 *          6. "Detect-0" returns fail.
 *              verify publish responses
 *          7. "Detect-0" returns good. Expect "Safety-chk-0"; return good; expect"Mitigate-0"; sleep 3s; mmitigate-0  responds; seq timeout
 *              verify publish responses
 *          8. "Detect-0" returns good. Expect "Safety-chk-0"; Sleep forever; req expect to timeout
 *              verify publish responses
 *          9. "Detect-2" & "Detect-1" returns good; But "Safety-chk-0" busy. bind-2 timesout.
 *          10.Expect "Safety-chk-1" call; return good; expect "Mitigate-1"; return good
 *              verify publish responses
 *          11.Trigger "Safety-chk-0" respond
 *          12."Detect-2" return good; "Safety-chk-0"; good; "Safety-chk-2"; good; "Mitigate-2"; good; seq complete
 *              verify publish responses
 *
 *          13."Detect-0" good; safety-check-0 sleeps; bind-0 timesout.
 *          14."Detect-0" good; safety-check-0 not called; bind-0 timesout.
 *          15.De-register safety-check-0 & re-register
 *          16."Detect-0" good; safety-check-0 good; mitigate-0 good; bind-0 good.
 *              verify publish responses
 *          17. NotifyHearbeat for "Detect-0"
 *              Verify responnse
 *          18. NotifyHearbeat for "xyz" non-existing
 *              Verify responnse
 *
 */


import (
    . "lib/lomcommon"
    . "lib/lomipc"
    "os"
    "path/filepath"
    "testing"
    "time"
)

const EMPTY_STR= ""
const CLIENT_0 = "client-0"
const CLIENT_1 = "client-1"
const CLIENT_2 = "client-2"

/*
 * During test run, test code keep this chan active. An idle channel for timeout
 * seconds will abort the test
 */
var chTestHeartbeat = make(chan string)

/*
 *  Actions.conf
 */
 var actions_conf = `{ "actions": [
        { "name": "Detect-0" },
        { "name": "Safety-chk-0", "Timeout": 1},
        { "name": "Mitigate-0", "Timeout": 6},
        { "name": "Detect-1" },
        { "name": "Safety-chk-1", "Timeout": 7},
        { "name": "Mitigate-1", "Timeout": 8},
        { "name": "Detect-2" },
        { "name": "Safety-chk-2", "Timeout": 1},
        { "name": "Mitigate-2", "Timeout": 6},
        { "name": "Disabled-0", "Disable": false}
        ] }`


var bindings_conf = `{ "bindings": [
    {
        "name": "bind-0", 
        "priority": 0,
        "Timeout": 2,
        "actions": [
            {"name": "Detect-0" },
            {"name": "Safety-chk-0", "sequence": 1 },
            {"name": "Mitigate-0", "sequence": 2 }
        ]
    },
    {
        "name": "bind-1", 
        "priority": 1,
        "Timeout": 19,
        "actions": [
            {"name": "Detect-1" },
            {"name": "Safety-chk-1", "sequence": 1 },
            {"name": "Mitigate-1", "sequence": 2 }
        ]
    },
    {
        "name": "bind-2", 
        "priority": 0,
        "Timeout": 1,
        "actions": [
            {"name": "Detect-2" },
            {"name": "Safety-chk-0", "sequence": 1 },
            {"name": "Safety-chk-2", "sequence": 2 },
            {"name": "Mitigate-2", "sequence": 3 }
        ]
    }
]}`


/*
 * A bunch of APIs from client transport or internal to engine to be called with varying
 * args and expected results
 */

type clientAPIID int
const (
    REG_CLIENT = clientAPIID(iota)
    REG_ACTION
    DEREG_CLIENT
    DEREG_ACTION
    RECV_REQ
    SEND_RES
    SHUTDOWN
    NOTIFY_HB
    CHK_ACTIV_REQ
    CHK_REG_ACTIONS
)

type testEntry_t struct {
    id          clientAPIID
    args        []any
    result      []any
    failed      bool            /* True if expected to fail. */
    desc        string
}


type testEntriesList_t  []testEntry_t

var testEntriesList = testEntriesList_t {
    {
        id: REG_ACTION,
        args: []any{"xyz"},
        failed: true,
        desc: "Call RegisterAction before register client",
    },
    {
        id: REG_CLIENT,
        args: []any{EMPTY_STR},
        failed: true,
        desc: "Empty string for client in regclient, to fail",
    },
    {
        id: REG_CLIENT,
        args: []any{CLIENT_0},
        failed: false,
        desc: "Client reg to succeed",
    },
}

const CFGPATH = "/tmp"

func createFile(t *testing.T, name string, s string) {
    fl := filepath.Join(CFGPATH, name)

    if len(s) == 0 {
        s = "{}"
    }
    if f, err := os.Create(fl); err != nil {
        t.Fatalf("Failed to create file (%s)", fl)
    } else {
        if _, err := f.WriteString(s); err != nil {
            t.Fatalf("Failed to write file (%s)", fl)
        }
        f.Close()
    }
    chTestHeartbeat <- "createFile: " + name
}

func initServer(t *testing.T) chan int {
    chTestHeartbeat <- "Start: initServer"
    defer func() {
        chTestHeartbeat <- "End: initServer"
    }()

    ch := make(chan int, 2)     /* Two to take start & end of loop w/o blocking*/
    startUp("test", []string { "-path", CFGPATH }, ch)
    chTestHeartbeat <- "Waiting: initServer"

    select {
    case <- ch:
        break

    case <- time.After(2 * time.Second):
        t.Fatalf("initServer failed")
    }
    return ch
}

type callArgs struct {
    t   *testing.T
    tx  *ClientTx
}

func (p *callArgs) call_register_client(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_register_client"
    defer func() {
        chTestHeartbeat <- "End: call_register_client"
    }()

    if len(te.args) != 1 {
        p.t.Fatalf("Expect only one arg len(%d)", len(te.args))
    }
    a := te.args[0]
    clName, ok := a.(string)
    if !ok {
        p.t.Fatalf("Expect string as arg for client name (%T)", a)
    }
    err := p.tx.RegisterClient(clName)
    if te.failed != (err != nil) {
        p.t.Fatalf("Test index %v: Unexpected behavior. te(%v) err(%v)",
                ti, *te, err)
    }
}

func (p *callArgs) call_register_action(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_register_action"
    defer func() {
        chTestHeartbeat <- "End: call_register_action"
    }()

    if len(te.args) != 1 {
        p.t.Fatalf("Expect only one arg len(%d)", len(te.args))
    }
    a := te.args[0]
    actName, ok := a.(string)
    if !ok {
        p.t.Fatalf("Expect string as arg for action name (%T)", a)
    }
    err := p.tx.RegisterAction(actName)
    if te.failed != (err != nil) {
        p.t.Fatalf("Test index %v: Unexpected behavior. te(%v) err(%v)",
                ti, *te, err)
    }
}

func terminate(t *testing.T, tout int) {
    LogDebug("DROP: Terminate guard called tout=%d", tout)
    for {
        select {
        case m := <- chTestHeartbeat:
            LogDebug("Test HB: (%s)", m)

        case <- time.After(time.Duration(tout) * time.Second):
            LogPanic("Terminating test for no heartbeats for tout=%d", tout)
        }
    }
}

    
func TestRun(t *testing.T) {
    go terminate(t, 5)

    createFile(t, "globals.conf.json", "")
    createFile(t, "actions.conf.json", actions_conf)
    createFile(t, "bindings.conf.json", bindings_conf)

    ch := initServer(t)

    tx := GetClientTx(0)
    if tx == nil {
        t.Fatalf("Failed to get client")
    }

    cArgs := &callArgs{t: t, tx: tx }

    for t_i, t_e := range testEntriesList {
        if len(ch) > 0 {
            t.Fatalf("Server loop exited")
        }
        switch (t_e.id) {
        case REG_CLIENT:
            cArgs.call_register_client(t_i, &t_e)
        case REG_ACTION:
            cArgs.call_register_action(t_i, &t_e)
        default:
            t.Fatalf("Unhandled API ID (%v)", t_e.id)
        }
    }
}

