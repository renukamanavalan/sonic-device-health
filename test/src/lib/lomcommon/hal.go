package lomcommon

import (
    "errors"
    "fmt"
    "log"
    "log/syslog"
    "os"
)

func PublishEvent(m map[string]string) {
    s = ""
    if b, err := json.Marshal(m); err != nil {
        LogError("Failed to marshal map (%v)", m)
        s = fmt.Sprintf("%v", m)
    } else {
        s = string(b)
    }
    PublishString(s)
}

func PublishString(s string) {
    LogInfo(s)
    // TODO: Call event publish
}

