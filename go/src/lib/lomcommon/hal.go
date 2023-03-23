package lomcommon

import (
    "encoding/json"
    "fmt"
)


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
    LogInfo(s)
    // TODO: Call event publish
    return s
}

