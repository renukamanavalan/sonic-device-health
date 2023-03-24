package engine

import (
    "flag"
    . "lib/lomcommon"
    . "lib/lomipc"
    "os"
    "os/signal"
    "syscall"
)

var BindingsConfFile = flag.String("b", "/etc/sonic/LoM/bindings.conf.json", 
            "Bindings config file")
var ActionsConfFile = flag.String("a", "/etc/sonic/LoM/actions.conf.json",
            "Actions config file")
var GlobalsConfFile = flag.String("g", "/etc/sonic/LoM/globals.conf.json",
            "Globals config file")

func readRequest(tx *LoMTransport, chAlert chan *LoMRequestInt, chAbort chan interface{}) {

    go func() {
        /* In forever read loop until aborted */
        for {
            /* Blocking read until request or error or signal in chAbort */
            req, err := tx.ReadClientRequest(chAbort)

            if req != nil {
                /* Select can block write into chAlert, so watch chAbort too */
                select {
                case chAlert <- req:
                    break
                case <- chAbort:
                    return
                }
            } else {
                /* Close as no more writes. This will help abort read loop below */
                close(chAlert)
                LogError("Failed to read request. err(%v)", err)
                return
            }
        }
    }()
}


func runLoop(tx *LoMTransport) {
    /*
     * Wait for
     *      signal to refresh
     *      request from client
     *      Internal timer for outstanding request's timeout processing
     */

    chSignal := make(chan os.Signal, 3)
    chAlert := make(chan *LoMRequestInt)
    chSeqHandler := make(chan interface{})

    /* Write abort is done once. It is best effort; To let send not block, make it buffered. */
    chAbort := make(chan interface{}, 1)

    signal.Notify(chSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

    readRequest(tx, chAlert, chAbort)

    server := GetServerReqHandler()

    InitSeqHandler(chSeqHandler)
loop:
    for {
        select {
        case msg := <- chAlert:
            server.processRequest(msg)

        case := <- chSeqHandler:
            GetSeqHandler().processTimeout()

        case sig := <- chSignal:
            switch(sig) {
            case syscall.SIGHUP, syscall.SIGINT:
                /* Reload */
                /* NOTE: Any currently active sequence will not be affected */
                Bindings.Load(BindingsConfFile, ActionsConfigFile)

            case syscall.SIG_TERM:
                chAbort <- "Aborted"
                break loop
            }
        }
    }
} 


func main() {

    flag.Parse()    /* Parse args */

    if _, err := InitConfigMgr(actionsConfFile, BindingsConfFile); err != nil {
        LogPanic("Failed to read config; actions(%s) bindings(%s)", 
                actionsConfFile, BindingsConfFile)
    } else {
        InitRegistrations()
        LogPeriodicInit()
    }

    tx, err := ServerInit()
    if err != nil {
        LogPanic("Failed to call ServerInit")
    }

    runLoop(tx)

    LogInfo("Engine exiting...")
}

