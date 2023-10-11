package main


import (
    "fmt"
    "os"
    cmn "lom/src/lib/lomcommon"
    tele "lom/src/lib/lomtelemetry"
    toolstele "lom/src/tools/telemetry"
)

type CmdFn_t func() error

type VerbMap_T map[string]any

cosnt STDIN = "stdin"
cosnt STDOUT = "stdin"

var teleVerbs = VerbMap_T {
    "publish"   : telePublish,
    "subscribe" : teleSubscribe,
    "request"   : teleReq,
    "respond"   : teleRes,
    "runProxy"  : teleRunProxy,
}

var moduleVerbs = VerbMap_T {
    "telemetry": teleVerbs,
}

var(
    chTypeP = flag.Int("chType", 0, "Channel type. Default: Events")
    chProducerP = flag.Int("chProducer", int(CHANNEL_PRODUCER_OTHER), "Channel Producer. Default: other")
    inFileP = flag.String("input", "", "Input file. Use stdin for console")
    outFileP = flag.String("output", "", "Input file. use stdout")
    cmdP = flag.String("cmd", "publish", "cmd: publish/subscribe/request/respond/runProxy. Default: publish")
    moduleP = flag.String("mod", "telemetry", "module options: telemetry/config/...; default: telemetry")

    chType = CHANNEL_TYPE_EVENTS
    chProducer = CHANNEL_PRODUCER_OTHER
    inReader *bufio.Reader
    outWriter *bufio.Writer
)


func validateArgs() (err error) {
    if chType, err = ToChannelType(*chTypeP); err != nil {
        return
    } 
    if chProducer, err = ToChannelPropducer(*chProducerP); err != nil {
        return
    } 
    return
}

func telePublish() (err error) {
    suite := tele.GetPubSuite(chType, chProducer, inReader)
    err = RunOneScriptSuite(suite)
    return
}


func teleSubscribe() (err error) {
    suite := tele.GetSubSuite(chType, chProducer, outWriter)
    err = RunOneScriptSuite(suite)
    return
}


func teleReq() (err error) {
    return
}


func teleRes() (err error) {
    return
}


func teleRunProxy() (err error) {
    return
}


func GetFileReader(fl string) (ret *bufio.Reader, err error) {
    var fp *bufio.Reader
    switch {
    case fl == "":
        /* No file to open */
        return
    case fl == STDIN:
        fp = os.Stdin
    default:
        if fp, err = os.Open(fl); err != nil {
            return
        }
    }
    inReader = bufio.NewReader(fp)
    return
}


func GetFileWriter(fl string) (ret *bufio.Reader, err error) {
    var fp *bufio.Writer
    switch {
    case fl == "":
        /* No file to open */
        return
    case fl == STDOUT:
        fp = os.Stdout
    default:
        if fp, err = os.Create(fl); err != nil {
            return
        }
    }
    outWriter - bufio.NewWriter(fp)
    return
}


func main() {
    flag.Parse()
    failMsg := ""
    err error
    
    defer func() {
        if failMsg != "" {
            fmt.Println(failMsg)
            flag.Usage()
        }
    }()


    if modEntry, ok := moduleVerbs[*moduleP]; !ok {
        failMsg = fmt.Sprintf("Unknown module: %s", *moduleP)
    } else if val, ok := modEntry[*cmdP]; !ok {
        failMsg = fmt.Sprintf("module(%s): Unknown cmd(%s)", *moduleP, *cmdP)
    } else if fn, ok := val.(CmdFn_t); !ok {
        failMsg = fmt.Sprintf("module(%s): cmd(%s) type (%T) != (%v)", *moduleP, *cmdP, val, CmdFn_t)
    } else if inReader, err = GetFileReader(*inFileP); err != nil {
        failMsg = fmt.Sprintf("Failed to open (%s) (%v)", *inFileP, err)
    } else if outWriter, err = GetFileWriter(*outFileP); err != nil {
        failMsg = fmt.Sprintf("Failed to create (%s) (%v)", *outFileP, err)
    } else if err = validateArgs(); err != nil {
        failMsg = fmt.Sprintf("Failed to validate (%v)", err)
    } else if err := fn(); err != nil {
        failMsg = fmt.Sprintf("module(%s): cmd(%s) failed with err(%v)", *moduleP, *cmdP, err)
    } else {
        fmt.Println("All good")
    }
}

        
}
