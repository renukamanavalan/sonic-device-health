// Client to return counters & events
package client

import (
    "encoding/json"
    "fmt"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/Workiva/go-datastructures/queue"
    gnmipb "github.com/openconfig/gnmi/proto/gnmi"
    lpb "lom/src/gnmi/proto"

    cmn "lom/src/lib/lomcommon"
    tele "lom/src/lib/lomtelemetry"
)

/* Path parameters - Refer "doc/gNMI_Info.txt" */
/* Send counter updates every UPD_FREQ seconds */
const PARAM_UPD_FREQ = "updfreq"
const PARAM_UPD_FREQ_DEFAULT = 10 /* Default 10 secs.*/
const PARAM_UPD_FREQ_MAX = 60     /* max = 1 min */
const PARAM_UPD_FREQ_MIN = 5      /* max = 5 secs */

/* On change only - Implies send updates every PARAM_UPD_FREQ seconds, only if there is change */
const PARAM_ON_CHANGE = "onchange"
const PARAM_ON_CHANGE_DEFAULT = true /* Default: Only upon change */

const PARAM_QSIZE = "qsize"
const PARAM_QSIZE_DEFAULT = 10240 // Def size for pending events in PQ.
const PARAM_QSIZE_MIN = 1024      // Min size for pending events in PQ.
const PARAM_QSIZE_MAX = 102400    // Max size for pending events in PQ.

const TEST_DATA = "{\"lom:test-"

/*
 * Anytime, we fail to send data, the dropped counter is incremented
 * This cumulative info is logged
 * Added to publshed counters
 * This data is collected over the gNMI server instance run time and
 * not impacted by client connections' lifetime.
 */
type DroppedDataType struct {
    mu   sync.Mutex
    data map[string]int
}

func (d *DroppedDataType) add(dType string, cnt int) {
    if cnt > 0 {
        d.mu.Lock()
        defer d.mu.Unlock()

        d.data[dType] += cnt
    }
}

func (d *DroppedDataType) inc(dType string) {
    d.add(dType, 1)
}

var droppedData = DroppedDataType{data: make(map[string]int)}

type LoMDataClient struct {
    id string

    prefix *gnmipb.Path
    path   *gnmipb.Path

    /* Params */
    pq_max  int
    updFreq int
    onChg   bool

    /* Queue to send response. */
    q *queue.PriorityQueue

    /* telemetry - count of messages sent */
    sentCnt int64

    /* Server call back on each successful send */
    lastReportedSentIndex int64

    /*
     * Internal data from subscribing for counters published.
     * chData - Chan to read published data from other LoM components.
     * chClose - Way to close the internal subscription chan
     */
    chData    <-chan tele.JsonString_t
    chClose   chan<- int
    chType    tele.ChannelType_t
    chTypeStr string
}

func NewLoMDataClient(path *gnmipb.Path, prefix *gnmipb.Path) (Client, error) {
    var c LoMDataClient
    var err error

    switch prefix.GetTarget() {
    case "EVENTS":
        c.chType = tele.CHANNEL_TYPE_EVENTS
    case "COUNTERS":
        c.chType = tele.CHANNEL_TYPE_COUNTERS
    default:
        return nil, cmn.LogError("Unexpected target=(%s)", prefix.GetTarget())
    }
    c.id = time.Now().Format("2006-01-02T15:04:05.000")
    c.chTypeStr = tele.CHANNEL_TYPE_STR[c.chType]

    c.prefix = prefix
    // Only one path is expected. Take the last if many
    c.path = path
    c.pq_max = PARAM_QSIZE_DEFAULT

    for _, e := range c.path.GetElem() {
        keys := e.GetKey()
        for k, v := range keys {
            if k == PARAM_UPD_FREQ {
                c.updFreq = cmn.ValidatedVal(v, PARAM_UPD_FREQ_MAX, PARAM_UPD_FREQ_MIN,
                    PARAM_UPD_FREQ_DEFAULT, k)
            } else if k == PARAM_QSIZE {
                c.pq_max = cmn.ValidatedVal(v, PARAM_QSIZE_MAX, PARAM_QSIZE_MIN,
                    PARAM_QSIZE_DEFAULT, k)
            } else if k == PARAM_ON_CHANGE {
                c.onChg, _ = strconv.ParseBool(v)
            }
        }
    }

    cmn.LogDebug("%s Subscribe params (UpdFreq=%d PQ-Max=%d OnChange=%v)",
        c.chTypeStr, c.updFreq, c.pq_max, c.onChg)

    /* Init subscriber with cache use and defined time out */
    if c.chData, c.chClose, err = tele.GetSubChannel(c.chType,
        tele.CHANNEL_PRODUCER_EMPTY, "", c.id); err != nil {
        cmn.LogError("Failed to create LoMDataClient for (%s) due to (%v)", c.chTypeStr, err)
        return nil, err
    }
    cmn.LogDebug("NewLoMDataClient constructed.")

    return &c, nil
}

// String returns the target the client is querying.
func (c *LoMDataClient) String() string {
    return fmt.Sprintf("LoMDataClient Prefix %v", c.prefix.GetTarget())
}

func sendData(c *LoMDataClient, sndData tele.JsonString_t) {
    sent := false

    defer func() {
        if !sent {
            droppedData.inc(c.chTypeStr)
            cmn.LogError("Dropped %s Total droppedx: %v", c.chTypeStr, droppedData)
        }
    }()

    if c.q.Len() >= c.pq_max {
        cmn.LogError("q:(%d/%d): Dropped %s Total dropped: %v",
            c.q.Len(), c.pq_max, c.chTypeStr, droppedData)
        return
    }

    fvp := map[string]any{}
    s := string(sndData)
    if err := json.Unmarshal([]byte(s), &fvp); err != nil {
        cmn.LogCritical("Invalid event message (%T) (%v)", s, s)
        cmn.LogCritical("Invalid event message err: (%v)", err)
        return
    }

    if c.chType == tele.CHANNEL_TYPE_COUNTERS {
        fvp["DROPPED"] = droppedData.data
    }

    if jv, err := json.Marshal(fvp); err != nil {
        cmn.LogCritical("Invalid event string: %v", sndData)
        return
    } else {
        tv := &gnmipb.TypedValue{
            Value: &gnmipb.TypedValue_JsonIetfVal{
                JsonIetfVal: jv,
            }}
        lpbv := &lpb.Value{
            Prefix:    c.prefix,
            Path:      c.path,
            Timestamp: time.Now().UnixMilli(),
            Val:       tv,
            SendIndex: c.sentCnt + 1,
        }

        if err = c.q.Put(Value{lpbv}); err != nil {
            cmn.LogError("Queue error:  %v", err)
            return
        }
    }
    cmn.LogInfo("sendData: (%s)", sndData)
    c.sentCnt++
    sent = true
}

func (c *LoMDataClient) StreamRun(q *queue.PriorityQueue, stop chan struct{}, wg *sync.WaitGroup, subscribe *gnmipb.SubscriptionList) {
    /* caller has already added to wg; so just done is enough. */
    defer wg.Done()

    c.q = q

    for {
        select {
        case ev := <-c.chData:
            if !strings.HasPrefix(string(ev), TEST_DATA) {
                sendData(c, ev)
            }

        case <-stop:
            close(c.chClose)
            return
        }
    }
}

func (c *LoMDataClient) FailedSend() {
    droppedData.inc(c.chTypeStr)
    cmn.LogError("Dropped %s Total droppedx: %v", c.chTypeStr, droppedData)
}

func (c *LoMDataClient) SentOne(v *Value) {

    diff := v.SendIndex - c.lastReportedSentIndex - 1
    if diff < 0 {
        cmn.LogError("Internal indices issue sentIndex(%v) lastSentIndex(%v) sentCnt(%v)",
            v.SendIndex, c.lastReportedSentIndex, c.sentCnt)
    } else if diff > 0 {
        droppedData.add(c.chTypeStr, int(diff))
    }
}

func (c *LoMDataClient) Close() error {
    return nil
}
