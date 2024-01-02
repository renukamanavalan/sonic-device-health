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
	linuxproc "github.com/c9s/goprocinfo/linux"
	log "github.com/golang/glog"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"

    cmn "lom/usr/lib/lomcommon"
    tele "lom/usr/lib/lomtelemetry"
)


const EVENTD_PUBLISHER_SOURCE = "{\"sonic-events-eventd"


const TEST_EVENT = "{\"sonic-host:device-test-event"

// Path parameter
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

/*
 * Anytime, we fail to send data, the dropped counter is incremented
 * This cumulative info is logged
 * TODO: Added to publshed counters
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
    wg          *sync.WaitGroup // wait for all sub go routines to finish
    chStop      chan struct{}

    // Internal data from subscribing for counters published.
    chData      <-chan JsonString_t // Chan to read published data from other LoM
                                    // components
    chClose     chan<- int          // Way to close the internal subscription chan
    chTypeStr   string
}


func NewLoMDataClient(paths []*gnmipb.Path, prefix *gnmipb.Path, logLevel int) (Client, error) {
    var c LoMDataClient
    var err error
    
    chType := tele.CHANNEL_TYPE_EVENTS

    cmn.SetLogLevel(logLevel)

    c.prefix = prefix
    // Only one path is expected. Take the last if many
    c.path = paths[len(paths)-1]

    c.pq_max = PARAM_QSIZE_DEFAULT
    c.updFreq = PARAM_UPD_FREQ_DEFAULT
    c.onChg = PARAM_ON_CHANGE_DEFAULT

    if prefix.GetTarget() == "COUNTERS" {
        chType := tele.CHANNEL_TYPE_COUNTERS
    }
    c.chTypeStr = tele.CHANNEL_TYPE_STR[chType]

    for _, e := range c.path.GetElem() {
        keys := e.GetKey()
        for k, v := range keys {
            if (k == PARAM_UPD_FREQ) {
                if val, err := strconv.Atoi(v); err == nil {
                    if (val > PARAM_UPD_FREQ_MAX) {
                        val = PARAM_UPD_FREQ_MAX
                    } else if (val < PARAM_UPD_FREQ_MIN) {
                        val = PARAM_UPD_FREQ_MIN
                    }
                    if val != qval {
                        cmn.LogWarning("Counters update freq %v updated to nearest limit %v",
                            qval, val)
                    }
                    LogDebug("c.heartbeat_interval is set to %d", val)
                    c.updFreq = val
                }
            } else if (k == PARAM_QSIZE) {
                if val, err := strconv.Atoi(v); err == nil {
                    qval := val
                    if (val < PARAM_QSIZE_MIN) {
                        val = PARAM_QSIZE_MIN
                    } else if (val > PARAM_QSIZE_MAX) {
                        val = PARAM_QSIZE_MAX
                    }
                    if val != qval {
                        LogWarning("Events priority Q request %v updated to nearest limit %v",
                            qval, val)
                    }
                    c.pq_max = val
                    LogDebug("Events priority Q max set by qsize param = %v", c.pq_max)
                }
            } else if (k == PARAM_ON_CHANGE) {
                c.onChg = !strings.ToLower(v) == "false"
                LogDebug("Counters onChange=%v", onChange)
            }
        }
    }

    /* Init subscriber with cache use and defined time out */
    if c.chData, c.chClose, err = tele.GetSubChannel(chType, CHANNEL_PRODUCER_EMPTY, ""); err != nil {
        cmn.LogError("Failed to create LoMDataClient due to (%v)", err)
        return nil, err
    }
    cmn.LogDebug("NewLoMDataClient constructed. logLevel=%d", logLevel)

    return &c, nil
}


// String returns the target the client is querying.
func (c *LoMDataClient) String() string {
    return fmt.Sprintf("EventClient Prefix %v", c.prefix.GetTarget())
}

func (c *LoMDataClient) send(sndData JsonString_t) {
    if (qlen >= evtc.pq_max) {
        droppedData[c.chTypeStr]++
        cmn.LogError("Dropped %s Total dropped: %v", c.chTypeStr, droppedData)
        return
    }
    var fvp map[string]interface{}
    json.Unmarshal([]byte(sndData), &fvp)

    jv, err := json.Marshal(fvp)

    if err == nil {
        evtTv := &gnmipb.TypedValue {
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
    } else {
        cmn.LogCritical("Invalid event string: %v", sndData)
    }
}


func (c *LoMDataClient) StreamRun(q *queue.PriorityQueue, stop chan struct{}, wg *sync.WaitGroup, subscribe *gnmipb.SubscriptionList) {
    c.wg = wg
    /* caller has already added to wg */
    defer c.wg.Done()

    c.q = q
    c.chStop = stop

    for {
        select {
        case ev := <-c.chData:
            c.send(ev)

        case <- c.chStop:
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

