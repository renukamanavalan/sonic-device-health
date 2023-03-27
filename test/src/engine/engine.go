package engine

import (
    "flag"
    . "lib/lomcommon"
    . "lib/lomipc"
    "os"
    "os/signal"
    "syscall"
)


var cfgFiles *ConfigFiles_t

func readRequest(tx *LoMTransport, chAlert chan *LoMRequestInt, chAbort chan interface{}) {
    if (tx == nil) || (chAlert == nil) || (chAbort == nil) {
        LogPanic("Internal error: Nil args (%v)(%v)(%v)", tx, chAlert, chAbort)
    }

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
    if tx == nil {
        LogPanic("Internal error: Nil LoMTransport")
    }

    chSignal := make(chan os.Signal, 3)
    chAlert := make(chan *LoMRequestInt, 1)
    chSeqHandler := make(chan int64, 2)

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

        case <- chSeqHandler:
            GetSeqHandler().processTimeout()

        case sig := <- chSignal:
            switch(sig) {
            case syscall.SIGHUP, syscall.SIGINT:
                /*
                 * Reload.
                 * NOTE: Any currently active sequence will not be affected
                 * On any error, continues to use last loaded values. 
                 */
                 InitConfigMgr(cfgFiles)

            case syscall.SIGTERM:
                chAbort <- "Aborted"
                break loop
            }
        }
    }
} 


func main() {
    testMode := flag.Bool("t", false, "Run in test mode")

    {
    globalFl := flag.String("g", "/etc/sonic/LoM/globals.conf.json", "Globals config file")
    actionsFl := flag.String("a", "/etc/sonic/LoM/actions.conf.json", "Actions config file")
    bindingsFl := flag.String("b", "/etc/sonic/LoM/bindings.conf.json", "Bindings config file")
    flag.Parse()    /* Parse args */

    cfgFiles = &ConfigFiles_t {
        GlobalFl: *globalFl,
        ActionsFl: *actionsFl,
        BindingsFl: *bindingsFl }

    if *testMode {
        testSetFiles(cfgFiles)
    }
        

    if _, err := InitConfigMgr(cfgFiles); err != nil {
        LogPanic("Failed to read config; (%v)", *cfgFiles)
    }
   
    chAbortLog := make(chan interface{}, 1)
    InitRegistrations()
    LogPeriodicInit(chAbortLog)

    tx, err := ServerInit()
    if err != nil {
        LogPanic("Failed to call ServerInit")
    }

    if *testMode {
        go testRun()
    }

    runLoop(tx)

    /* Abort LogPeriodic */
    chAbortLog <- 0

    LogInfo("Engine exiting...")
}

