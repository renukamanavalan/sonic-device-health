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
GlobalsConfFile = flag.String("g", "/etc/sonic/LoM/globals.conf.json",
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
            } else {
                /* Close as no more writes. This will help abort read loop below */
                close(chAlert)
                LogError("Failed to read request. err(%v)", err)
                return
            }
        }
    }()
}

type oneShotTimers st5
/* One shot timers */
var oneShotTimers = make(map[int64][]func())
var sortedTimers []int64


func AddOneShotTimer(due int64, f func()) {
    oneShotTimers[due] = append(oneShotTimers[due], f)
    sortedTimers = append(sortedTimers, due)
    sort.Slice(sortedTimers, func(i, j int) bool {
        return sortedTimers[i] < sortedTimers[j]
    })
}

func 

func runLoop(tx *LoMTransport) {
    /*
     * Wait for
     *      signal to refresh
     *      request from client
     *      Internal timer for outstanding request's timeout processing
     */

    chSignal := make(chan os.Signal)
    chAlert := make(chan *LoMRequestInt)

    /* Write abort is done once. It is best effort; To let send not block, make it buffered. */
    chAbort := make(chan interface{}, 1)

    signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

    readRequest(tx, chAlert, chAbort)

    server := GetServerReqHandler()

loop:
    for {
        select {
        case msg := <- chAlert:
            server.processRequest(msg)

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

    if _, err := InitConfigMgr((actionsConfFile, BindingsConfFile); err != nil {
        LogPanic("Failed to read config; actions(%s) bindings(%s)", 
                actionsConfFile, BindingsConfFile)
    } else {
        InitRegistrations()
        LogPeriodicInit()
        InitSeqHandler()
    }

    tx, err := ServerInit()
    if err != nil {
        LogPanic("Failed to call ServerInit")
    }

    runLoop(tx)

    LogInfo("Engine exiting...")
}

