package engine

import (
    "flag"
    . "lib/lomcommon"
    . "lib/lomipc"
    "os"
    "os/signal"
    "syscall"
)

BindingsConfFile = flag.String("b", "/etc/sonic/LoM/bindings.conf.json", 
            "Bindings config file")
ActionsConfFile = flag.String("a", "/etc/sonic/LoM/actions.conf.json",
            "Actions config file")

func readRequest(tx *LoMTransport, chAlert chan interface{}, chAbort chan interface{}) {

    go func() {
        /* In forever read loop until aborted */
        for {
            /* Blocking read until request or error or signal in chAbort */
            req, err := tx.ReadClientRequest(chAbort)

            if req != nil {
                select {
                case chAlert <- req:
                    break
                case <- chAbort:
                    return
            } else {
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

    chAlert := make(chan interface{})

    /* Write abort is done once. It is best effort; To let send not block, make it buffered. */
    chAbort := make(chan interface{}, 1)

    sigHandler(chAlert)

    readRequest(tx, chAlert, chAbort)

    server := GetServerReqHandler()

loop:
    for msg := range chAlert {
        if s, ok := msg.(SigReceived); ok {
            switch(s) {
            case syscall.SIGHUP, syscall.SIGINT:
                /* Reload */
                /* NOTE: Any currently active sequence will not be affected */
                Bindings.Load(BindingsConfFile, ActionsConfigFile)

            case SIG_TERM:
                chAbort <- "Aborted"
                break loop
            }
        } else if s, ok := msg.(*LoMRequestInt) {
            server.processRequest()
        }
    }
} 


func main() {

    flag.Parse()    /* Parse args */

    if _, err := InitConfigMgr((actionsConfFile, BindingsConfFile); err != nil {
        LogPanic("Failed to read config; actions(%s) bindings(%s)", 
                actionsConfFile, BindingsConfFile)
    } else {
        InitRegistrations()
    }

    tx, err := ServerInit()
    if err != nil {
        LogPanic("Failed to call ServerInit")
    }

    runLoop(tx)

    LogInfo("Engine exiting...")
}

