package toolsmain

import (
    "bufio"
    "flag"
    "fmt"
    "log/syslog"
    cmn "lom/src/lib/lomcommon"
    script "lom/src/lib/lomscripted"
    tele "lom/src/lib/lomtelemetry"
    teletools "lom/src/tools/toolstelemetry"
    "os"
)

type CmdFn_t func() error

type VerbMap_T map[string]CmdFn_t

const STDIN = "stdin"
const STDOUT = "stdout"
const Hours_24 = (24 * 3600)

const DEF_LOG_LVL = 3 /* 3 ERR, 6 INFO & 7 DEBUG */

var teleVerbs = VerbMap_T{
    "publish":   telePublish,
    "subscribe": teleSubscribe,
    "request":   teleReq,
    "respond":   teleRes,
    "runProxy":  teleRunProxy,
}

type ModVerbMap_T map[string]VerbMap_T

var moduleVerbs = ModVerbMap_T{
    "telemetry": teleVerbs,
}

var (
    chTypeP           = flag.Int("chType", 0, fmt.Sprint(tele.CHANNEL_TYPE_STR)+" Default: 0")
    chProducerP       = flag.Int("chProducer", int(tele.CHANNEL_PRODUCER_OTHER), "Channel Producer. Default: other")
    chProducerSuffixP = flag.String("name", "cliTool", "Channel Producer. Suffix needed except for engine")
    inFileP           = flag.String("input", STDIN, "Input file. Use stdin for console")
    outFileP          = flag.String("output", STDOUT, "Input file. use stdout")
    cmdP              = flag.String("cmd", "publish", "cmd: publish/subscribe/request/respond/runProxy. Default: publish")
    moduleP           = flag.String("mod", "telemetry", "module options: telemetry/config/...; default: telemetry")
    toutP             = flag.Int("timeout", Hours_24, "Timeout used")
    cntP              = flag.Int("count", 1000, "Count used where needed")
    logLvlP           = flag.Int("loglvl", DEF_LOG_LVL, "Log level. Levels inuse: 3-Error 6-info, 7-debug")

    inReader  *bufio.Reader
    outWriter *bufio.Writer
)

func validateArgs() (err error) {
    if *chTypeP >= int(tele.CHANNEL_TYPE_CNT) {
        err = cmn.LogError("Invalid value for chTytpe (%v)", *chTypeP)
    } else if *chProducerP >= int(tele.CHANNEL_PRODUCER_CNT) {
        err = cmn.LogError("Invalid value for chProducer (%v)", *chProducerP)
    }
    return
}

func telePublish() error {
    if suite, err := teletools.GetPubSuite(tele.ChannelType_t(*chTypeP),
        tele.ChannelProducer_t(*chProducerP), *chProducerSuffixP, "cli", inReader); err != nil {
        return err
    } else {
        prStr, _ := tele.GetProdStr(tele.ChannelProducer_t(*chProducerP), *chProducerSuffixP)
        cmn.LogInfo("Running Publish for chType(%s) chProducer(%s) for file(%s)",
            tele.CHANNEL_TYPE_STR[tele.ChannelType_t(*chTypeP)], prStr, *inFileP)
        return script.RunOneScriptSuite(suite)
    }
}

func teleSubscribe() error {
    if suite, err := teletools.GetSubSuite(tele.ChannelType_t(*chTypeP),
        tele.ChannelProducer_t(*chProducerP), *chProducerSuffixP, "cli", outWriter); err != nil {
        return err
    } else {
        prStr, _ := tele.GetProdStr(tele.ChannelProducer_t(*chProducerP), *chProducerSuffixP)
        cmn.LogInfo("Running Subscribe for chType(%s) chProducer(%s) for file(%s)",
            tele.CHANNEL_TYPE_STR[tele.ChannelType_t(*chTypeP)], prStr, *outFileP)
        return script.RunOneScriptSuite(suite)
    }
}

func teleReq() error {
    if suite, err := teletools.GetReqSuite(tele.ChannelType_t(*chTypeP), *cntP,
        outWriter, inReader, *toutP); err != nil {
        return err
    } else {
        cmn.LogInfo("Running request client for chType(%s) cnt(%d) for file in:(%s) out:(%s) timeout(%d)",
            tele.CHANNEL_TYPE_STR[tele.ChannelType_t(*chTypeP)], *cntP, *inFileP, *outFileP, *toutP)
        return script.RunOneScriptSuite(suite)
    }
}

func teleRes() error {
    if suite, err := teletools.GetResSuite(tele.ChannelType_t(*chTypeP), *cntP,
        outWriter, inReader, *toutP); err != nil {
        return err
    } else {
        cmn.LogInfo("Running server responder for chType(%s) cnt(%d) for file in:(%s) out:(%s) timeout(%d)",
            tele.CHANNEL_TYPE_STR[tele.ChannelType_t(*chTypeP)], *cntP, *inFileP, *outFileP, *toutP)
        return script.RunOneScriptSuite(suite)
    }
}

func teleRunProxy() (err error) {
    if suite, err := teletools.GetProxySuite(tele.ChannelType_t(*chTypeP), *toutP); err != nil {
        return err
    } else {
        cmn.LogInfo("Running Proxy for chType(%s) for timeout(%d)",
            tele.CHANNEL_TYPE_STR[tele.ChannelType_t(*chTypeP)], *toutP)
        return script.RunOneScriptSuite(suite)
    }
    return
}

func GetFileReader(fl string) (err error) {
    var fp *os.File
    switch {
    case fl == "":
        /* No file to open */
        return
    case fl == STDIN:
        fp = os.Stdin
        cmn.LogDebug("Read from stdin ...")
    default:
        if fp, err = os.Open(fl); err != nil {
            return
        }
    }
    inReader = bufio.NewReader(fp)
    return
}

func GetFileWriter(fl string) (err error) {
    var fp *os.File
    switch {
    case fl == "":
        /* No file to open */
        return
    case fl == STDOUT:
        fp = os.Stdout
        cmn.LogDebug("Write into stdout...")
    default:
        if fp, err = os.Create(fl); err != nil {
            return
        }
    }
    outWriter = bufio.NewWriter(fp)
    return
}

func RunMain() {
    flag.Parse()
    failMsg := ""
    printUsage := true
    var err error

    defer func() {
        if failMsg != "" {
            fmt.Println(failMsg)
            if printUsage {
                flag.Usage()
            }
        }
    }()

    cmn.SetLogLevel(syslog.Priority(*logLvlP))
    if modEntry, ok := moduleVerbs[*moduleP]; !ok {
        failMsg = fmt.Sprintf("Unknown module: %s", *moduleP)
    } else if fn, ok := modEntry[*cmdP]; !ok {
        failMsg = fmt.Sprintf("module(%s): Unknown cmd(%s)", *moduleP, *cmdP)
    } else if err = GetFileReader(*inFileP); err != nil {
        failMsg = fmt.Sprintf("Failed to open (%s) (%v)", *inFileP, err)
    } else if err = GetFileWriter(*outFileP); err != nil {
        failMsg = fmt.Sprintf("Failed to create (%s) (%v)", *outFileP, err)
    } else if err = validateArgs(); err != nil {
        failMsg = fmt.Sprintf("Failed to validate (%v)", err)
    } else if err := fn(); err != nil {
        printUsage = false
        failMsg = fmt.Sprintf("module(%s): cmd(%s) failed with err(%v)", *moduleP, *cmdP, err)
    } else {
        fmt.Println("All good")
    }
}
