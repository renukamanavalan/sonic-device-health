package libtest

import (
    "fmt"

    zmq "github.com/pebbe/zmq4"
    cmn "lom/src/lib/lomcommon"
    script "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
)

var tctx *zmq.Context
var tsock *zmq.Socket

func fail_ctrl_port(name string, val any) (retV any, err error) {
    retV = val

    if tctx != nil || tsock != nil {
        err = cmn.LogError("Check test code. Expect nil")
    }
    tctx, err = zmq.NewContext()

    if err == nil {
        tsock, err = tctx.NewSocket(zmq.XSUB)
    }
    if err == nil {
        chType, ok := val.(tele.ChannelType_t)
        if !ok {
            err = cmn.LogError("expect tele.ChannelType_t != (%T)", val)
        } else {
            addr := fmt.Sprintf(tele.ZMQ_ADDRESS, tele.ZMQ_PROXY_CTRL_PORT+int(chType))
            err = tsock.Bind(addr)
        }
    }
    return
}

func fail_response_port(name string, val any) (retV any, err error) {
    retV = val

    if tctx != nil || tsock != nil {
        err = cmn.LogError("Check test code. Expect nil")
    }
    tctx, err = zmq.NewContext()

    if err == nil {
        tsock, err = tctx.NewSocket(zmq.REP)
    }
    if err == nil {
        chType, ok := val.(tele.ChannelType_t)
        if !ok {
            err = cmn.LogError("expect tele.ChannelType_t != (%T)", val)
        } else {
            addr := fmt.Sprintf(tele.ZMQ_ADDRESS, tele.ZMQ_REQ_REP_START_PORT+int(chType))
            err = tsock.Bind(addr)
        }
    }
    return
}

func cleanup_ctx_port(name string, val any) (ret any, err error) {
    if tsock != nil {
        tsock.Close()
        tsock = nil
    }
    if tctx != nil {
        cmn.LogDebug("Terminating test context")
        tctx.Term()
        tctx = nil
        cmn.LogDebug("Terminated test context")
    }
    return
}

/*
 * BIG NOTE:  Let this be a last test suite.
 * After shutdown, no API will succeed
 * There is no way to revert shutdown -- One way to exit
 */
var pubSubBindFail = testSuite_t{
    id:          "pubSubBindFailSuite",
    description: "Test pub sub for request & response - Good run",
    tests: []testEntry_t{
        testEntry_t{ /* Pre-bind the address to simulate failure */
            script.ApiIDRunPubSubProxy,
            []script.Param_t{
                script.Param_t{"chType_E", tele.CHANNEL_TYPE_EVENTS, fail_ctrl_port},
            },
            []result_t{NIL_ANY, NON_NIL_ERROR},
            "Xsub bind failure",
        },
        testEntry_t{
            script.ApiIDRunPubSubProxy,
            []script.Param_t{script.Param_t{script.ANONYMOUS, nil, cleanup_ctx_port}},
            []result_t{NIL_ANY, NON_NIL_ERROR},
            "Intentional failure to call cleanup",
        },
        testEntry_t{ /* Test handler shutdown in stat = LState_WriteReq */
            script.ApiIDRegisterServerReqHandler,
            []script.Param_t{script.Param_t{"chType_1", tele.CHANNEL_TYPE_ECHO, fail_response_port}},
            []result_t{NIL_ANY, NIL_ANY, NON_NIL_ERROR},
            "Duplicate req to fail",
        },
        testEntry_t{
            script.ApiIDRunPubSubProxy,
            []script.Param_t{script.Param_t{script.ANONYMOUS, nil, cleanup_ctx_port}},
            []result_t{NIL_ANY, NON_NIL_ERROR},
            "Intentional failure to call cleanup",
        },
        TELE_IDLE_CHECK,
    },
}
