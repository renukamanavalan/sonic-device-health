package lomcommon

import (
    "encoding/json"
    "fmt"
    "log/syslog"
)

/*
 *  Publish string as event
 *
 *  Input:
 *      The given string is logged & published
 *      
 *  Output:
 *      None
 *
 *  Return:
 *      The string that was published.
 *
 */
func PublishString(s string) string {
    LogMessage(syslog.LOG_INFO, s)
    // TODO: Call event publish
    return s
}

/* will be set to appropriate API */
var publishEventAPI func(string) string = PublishString

func SetPublishAPI(f func(string) string) {
    publishEventAPI = f
}


/*
 *  Publish as event
 *
 *  Input:
 *      A map of string vs string. JSonified map will be published.
 *      
 *  Output:
 *      None
 *
 *  Return:
 *      The string that was published. 
 *
 */
func PublishEvent(m any) string {
    s := ""
    if b, err := json.Marshal(m); err != nil {
        LogError("Failed to marshal map (%v)", m)
        s = fmt.Sprintf("%v", m)
    } else {
        s = string(b)
    }
    return publishEventAPI(s)
}

