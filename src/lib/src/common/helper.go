package common

import (
    "fmt"
    "log"
    "log/syslog"
    "os"
)

var writers = make(map[syslog.Priority]*syslog.Writer)

var log_level = syslog.LOG_DEBUG

func init() {

    for i := syslog.LOG_EMERG; i <= syslog.LOG_DEBUG; i++ {
        writer, err := syslog.Dial("", "", (i|syslog.LOG_LOCAL7), "")
        if err != nil {
            log.Fatal(err)
        }
        writers[i] = writer
    }

    /*
     * Samples:
     *  fmt.Fprintf(writers[syslog.LOG_WARNING], "This is a daemon warning message")
     *  fmt.Fprintf(writers[syslog.LOG_ERR], "This is a daemon ERROR message")
     *  fmt.Fprintf(writers[syslog.LOG_INFO], "This is a daemon INFO message")
     */
}

func GetLogLevel() syslog.Priority {
    return log_level
}


func SetLogLevel(lvl syslog.Priority) {
    log_level = lvl
}


func LogMessage(lvl syslog.Priority, s string, a ...interface{})  {
    ct_lvl := GetLogLevel()
    if ct_lvl <= lvl {
        fmt.Fprintf(writers[lvl], s, a...)
        if ct_lvl >= syslog.LOG_DEBUG {
            /* Debug messages gets printed out to STDOUT */
            fmt.Printf(s, a...)
            fmt.Println("")
        }
    }
}


func LogPanic(s string, a ...interface{})  {
    LogMessage(syslog.LOG_CRIT, s, a...)
    LogMessage(syslog.LOG_CRIT, "LoM exiting ...")
    os.Exit(-1)
}


func LogError(s string, a ...interface{})  {
    LogMessage(syslog.LOG_ERR, s, a...)
}


func LogWarning(s string, a ...interface{})  {
    LogMessage(syslog.LOG_WARNING, s, a...)
}


func LogInfo(s string, a ...interface{})  {
    LogMessage(syslog.LOG_INFO, s, a...)
}


func LogDebug(s string, a ...interface{})  {
    LogMessage(syslog.LOG_DEBUG, s, a...)
}

