// Client to return counters
//
package client

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "sync"
    "time"

    "gopkg.in/yaml.v2"

    lpb "lom/src/gnmi/proto"
    "github.com/Workiva/go-datastructures/queue"
    gnmipb "github.com/openconfig/gnmi/proto/gnmi"

    cmn "lom/usr/lib/lomcommon"
    tele "lom/usr/lib/lomtelemetry"
)


/* Path parameters - Refer "doc/gNMI_Info.txt" */
/* Send counter updates every UPD_FREQ seconds */
const PARAM_UPD_FREQ = "updfreq"
const PARAM_UPD_FREQ_DEFAULT = 10 /* Default 10 secs.*/
const PARAM_UPD_FREQ_MAX     = 60 /* max = 1 min */
const PARAM_UPD_FREQ_MIN     = 5  /* max = 5 secs */

/* On change only - Implies send updates every PARAM_UPD_FREQ seconds, only if there is change */
const PARAM_ON_CHANGE = "onchange"
const PARAM_ON_CHANGE_DEFAULT = true    /* Default: Only upon change */

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
var droppedData = map[string]int{}

type LoMDataClient struct {
    prefix      *gnmipb.Path
    path        *gnmipb.Path

    /* Params */
    pq_max      int
    updFreq     int
    onChg       bool

    /* Runstream params */
    q           *queue.PriorityQueue

    // Internal data from subscribing for counters published.
    chData      <-chan JsonString_t // Chan to read published data from other LoM
                                    // components
    chClose     chan<- int          // Way to close the internal subscription chan
    chType      tele.ChannelType_t
    chTypeStr   string
}


func NewLoMDataClient(paths []*gnmipb.Path, prefix *gnmipb.Path, logLevel int) (Client, error) {
    var c LoMDataClient
    var err error
    
    c.chType := tele.CHANNEL_TYPE_EVENTS

    cmn.SetLogLevel(logLevel)

    c.prefix = prefix
    // Only one path is expected. Take the last if many
    c.path = paths[len(paths)-1]

    /* Caller already filtered for events/counters */
    if prefix.GetTarget() == "COUNTERS" {
        c.chType = tele.CHANNEL_TYPE_COUNTERS
    } else if prefix.GetTarget() != "EVENTS" {
        return nil, cmn.LogError("Unexpected target=(%s)", prefix.GetTarget())
    }
    
    c.chTypeStr = tele.CHANNEL_TYPE_STR[c.chType]

    for _, e := range c.path.GetElem() {
        keys := e.GetKey()
        for k, v := range keys {
            if (k == PARAM_UPD_FREQ) {
                c.updFreq = validatedVal(v, PARAM_UPD_FREQ_MAX, PARAM_UPD_FREQ_MIN,
                            PARAM_UPD_FREQ_DEFAULT, k)
            } else if (k == PARAM_QSIZE) {
                c.pq_max = validatedVal(v, PARAM_QSIZE_MAX, PARAM_QSIZE_MIN,
                            PARAM_QSIZE_DEFAULT, k)
            } else if (k == PARAM_ON_CHANGE) {
                c.onChg = !strings.ToLower(v) == "false"
            }
        }
    }

    LogDebug("%s Subscribe params (UpdFreq=%d PQ-Max=%d OnChange=%v)",
            c.chTypeStr, c.updFreq, c.pq_max, c.onChg)

    /* Init subscriber with cache use and defined time out */
    if c.chData, c.chClose, err = tele.GetSubChannel(chType, CHANNEL_PRODUCER_EMPTY, ""); err != nil {
        cmn.LogError("Failed to create LoMDataClient for (%s) due to (%v)", c.chTypeStr, err)
        return nil, err
    }
    cmn.LogDebug("NewLoMDataClient constructed. logLevel=%d", logLevel)

    return &c, nil
}


// String returns the target the client is querying.
func (c *LoMDataClient) String() string {
    return fmt.Sprintf("LoMDataClient Prefix %v", c.prefix.GetTarget())
}

sendData(c *LoMDataClient, sndData JsonString_t) {
    if (q.Len() >= c.pq_max) {
        droppedData[c.chTypeStr]++
        cmn.LogError("Dropped %s Total dropped: %v", c.chTypeStr, droppedData)
        return
    }
    var fvp map[string]interface{}
    json.Unmarshal([]byte(sndData), &fvp)

    if jv, err := json.Marshal(fvp); err != nil {
        cmn.LogCritical("Invalid event string: %v", sndData)
        return
    } 

    if c.chType == CHANNEL_TYPE_COUNTERS {
        fvp["DROPPED"] = droppedData
    }

    tv := &gnmipb.TypedValue {
        Value: &gnmipb.TypedValue_JsonIetfVal {
            JsonIetfVal: jv,
        }}
    lpbv := &lpb.Value{
        Prefix:    c.prefix,
        Path:      c.path,
        Timestamp: time.Now().UnixMilli(),
        Val:  tv,
    }

    if err = c.q.Put(Value{lpbv}); err != nil {
        cmn.LogError("Queue error:  %v", err)
    }
}


func (c *LoMDataClient) StreamRun(q *queue.PriorityQueue, stop chan struct{}, wg *sync.WaitGroup, subscribe *gnmipb.SubscriptionList) {
    /* caller has already added to wg; so just done is enough. */
    defer wg.Done()

    c.q = q

    for {
        select {
        case ev := <-c.chData:
            if !strings.HasPrefix(ev, TEST_DATA) {
                sendData(c, ev)
            }

        case <- chStop:
            close(c.chClose)
            return
        }
    }
}


func (c *LoMDataClient) OnceRun(q *queue.PriorityQueue, once chan struct{}, wg *sync.WaitGroup, subscribe *gnmipb.SubscriptionList) {
    return
}

func (c *LoMDataClient) PollRun(q *queue.PriorityQueue, poll chan struct{}, wg *sync.WaitGroup, subscribe *gnmipb.SubscriptionList) {
    return
}

func (c *LoMDataClient) Get(w *sync.WaitGroup) ([]*lpb.Value, error) {
    return nil, cmn.LogError("NotImpl")
}

func (c *LoMDataClient) Set(delete []*gnmipb.Path, replace []*gnmipb.Update, update []*gnmipb.Update) error {
    return cmn.LogError("NotImpl")
}

func (c *LoMDataClient) Capabilities() []gnmipb.ModelData {
    return nil
}

func (c *LoMDataClient) FailedSend() {
    return
}

func (c *LoMDataClient) SentOne(*Value) {
    return
}

