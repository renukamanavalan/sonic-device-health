package engine

import (
    "bytes"
    "flag"
    . "lib/lomcommon"
    . "lib/lomipc"
    "os"
    "os/signal"
    "path/filepath"
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


func runLoop(tx *LoMTransport, chTrack chan int) {
    /*
     * Wait for
     *      signal to refresh
     *      request from client
     *      Internal timer for outstanding request's timeout processing
     */
    if tx == nil {
        LogPanic("Internal error: Nil LoMTransport")
    }

    chAbortLog := make(chan interface{}, 1)
    LogPeriodicInit(chAbortLog)

    chSignal := make(chan os.Signal, 3)
    chAlert := make(chan *LoMRequestInt, 1)
    chSeqHandler := make(chan int64, 2)

    /* Write abort is done once. It is best effort; To let send not block, make it buffered. */
    chAbort := make(chan interface{}, 1)

    signal.Notify(chSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

    readRequest(tx, chAlert, chAbort)

    server := GetServerReqHandler()

    InitSeqHandler(chSeqHandler)

    chTrack <- 0
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
    chTrack <- 1

    /* Abort LogPeriodic */
    chAbortLog <- 0
} 


func startUp(progname string, args []string, chTrack chan int) {

    path := ""
    {
        p := ""
        flags := flag.NewFlagSet(progname, flag.ContinueOnError)
        var buf bytes.Buffer
        flags.SetOutput(&buf)

        flags.StringVar(&p, "path", "", "Config files path")

        err := flags.Parse(args)
        if  err != nil {
            LogPanic("Failed to parse (%v); details(%s)", args, buf.String())
        }
        path = p
    }

    if len(path) == 0 {
        if p, err := os.Getwd(); err != nil {
            LogPanic("Failed to get current working dir (%v)", err)
        } else {
            path = p
        }
    }

    cfgFiles = &ConfigFiles_t {
        GlobalFl: filepath.Join(path, "globals.conf.json"),
        ActionsFl: filepath.Join(path, "actions.conf.json"),
        BindingsFl: filepath.Join(path, "bindings.conf.json"),
    }

    if _, err := InitConfigMgr(cfgFiles); err != nil {
        LogPanic("Failed to read config; (%v)", *cfgFiles)
    }
   
    InitRegistrations()
    tx, err := ServerInit()
    if err != nil {
        LogPanic("Failed to call ServerInit")
    }

    go runLoop(tx, chTrack)
}

func main() {
    ch := make(chan int, 2)  
    startUp(os.Args[0], os.Args[1:], ch)

    <- ch
    LogDebug("Loop started")

    <- ch
    LogDebug("Loop ended")


    LogInfo("Engine exiting...")
}

