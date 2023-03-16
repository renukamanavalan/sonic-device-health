package lomcommon

import (
    "encoding/json"
    "fmt"
)

func PublishEvent(m map[string]string) string {
    s := ""
    if b, err := json.Marshal(m); err != nil {
        LogError("Failed to marshal map (%v)", m)
        s = fmt.Sprintf("%v", m)
    } else {
        s = string(b)
    }
    return PublishString(s)
}

func PublishString(s string) string {
    LogInfo(s)
    // TODO: Call event publish
    return s
}

