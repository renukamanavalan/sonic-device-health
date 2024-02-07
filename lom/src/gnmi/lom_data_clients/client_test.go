package client

import (
    "encoding/json"
    "errors"
    "fmt"
    "reflect"
    "strings"
    "testing"

    "github.com/Workiva/go-datastructures/queue"
    "github.com/agiledragon/gomonkey/v2"

    gnmipb "github.com/openconfig/gnmi/proto/gnmi"

    lpb "lom/src/gnmi/proto"
    cmn "lom/src/lib/lomcommon"
    tele "lom/src/lib/lomtelemetry"
)

func TestDropped(t *testing.T) {
    dExpect := map[string]int{
        "foo": 5,
        "bar": 1,
    }

    for k, v := range dExpect {
        for i := 0; i < v; i++ {
            droppedData.inc(k)
        }
    }

    if !reflect.DeepEqual(dExpect, droppedData.data) {
        t.Fatalf("Incorrect data exp(%v) != res(%v)", dExpect, droppedData)
    }
}

func testLogs(logMsgs []string, msg string) bool {
    for _, lmsg := range logMsgs {
        if strings.Contains(lmsg, msg) {
            cmn.LogDebug("Found Msg: (%s) exp(%s)", lmsg, msg)
            return true
        }
        cmn.LogDebug("Skip Msg: (%s)", lmsg)
    }
    return false
}

func TestLoMDataClient(t *testing.T) {
    {
        /* Test NewLoMDataClient */
        cntrPrefix := &gnmipb.Path{Target: "COUNTERS"}
        pathWithKeys := &gnmipb.Path{
            Elem: []*gnmipb.PathElem{
                &gnmipb.PathElem{
                    Key: map[string]string{
                        PARAM_UPD_FREQ:  "100",   /* higher than max */
                        PARAM_QSIZE:     "10241", /* Value in range */
                        PARAM_ON_CHANGE: "false", /* Non default val */
                    },
                },
            },
        }

        lstTest := map[string]struct {
            path   *gnmipb.Path
            prefix *gnmipb.Path
            subErr error
            cl     *LoMDataClient
        }{
            "Unexpected target=": {prefix: &gnmipb.Path{Target: "Foo"}},
            "Failed to create LoMDataClient for": {
                prefix: cntrPrefix,
                path:   &gnmipb.Path{},
                subErr: errors.New("mock"),
            },
            "": {
                prefix: cntrPrefix,
                path:   pathWithKeys,
                cl: &LoMDataClient{
                    chType:    tele.CHANNEL_TYPE_COUNTERS,
                    chTypeStr: tele.CHANNEL_TYPE_STR[tele.CHANNEL_TYPE_COUNTERS],
                    prefix:    cntrPrefix,
                    path:      pathWithKeys,
                    pq_max:    10241,
                    updFreq:   PARAM_UPD_FREQ_MIN,
                    onChg:     false,
                },
            },
        }

        for msg, td := range lstTest {
            mockSub := gomonkey.ApplyFunc(tele.GetSubChannel,
                func(tele.ChannelType_t, tele.ChannelProducer_t, string, string) (<-chan tele.JsonString_t, chan<- int, error) {
                    return nil, nil, td.subErr
                })
            defer mockSub.Reset()

            cl, err := NewLoMDataClient(td.path, td.prefix)
            if msg != "" {
                if (err == nil) || !strings.Contains(fmt.Sprint(err), msg) {
                    t.Fatalf("Failing to file as expected (%s) != err(%v)", msg, err)
                }
            } else {
                if err != nil {
                    t.Fatalf("Expected to succeed err=(%v)", err)
                }
                if lcd, ok := cl.(*LoMDataClient); !ok {
                    t.Fatalf("Expected LoMDataClient. Got (%T)", cl)
                } else if lcd.String() != td.cl.String() {
                    t.Fatalf("clients don't match lcd(%s) != exp(%s)", lcd.String(), td.cl.String())
                }
            }
        }
    }
    {
        /* Test sendData */
        cmn.LogDebug("Test sendData START =================")
        testMsgs := []string{
            "Dropped %s Total droppedx:",
            "Total dropped",
            "Invalid event message",
            "Invalid event string",
            "Queue error:",
        }
        for _, msg := range testMsgs {
            cl := &LoMDataClient{}
            var sndData tele.JsonString_t
            var marshalErr error = nil
            var qErr error = nil
            cl.q = queue.NewPriorityQueue(1, false)
            cl.pq_max = 10
            cl.chType = tele.CHANNEL_TYPE_COUNTERS

            switch {
            case msg == "Dropped %s Total droppedx:":
                cl.pq_max = -1
            case msg == "Total dropped":
                cl.pq_max = -1
            case msg == "Invalid event message":
                sndData = "foo"
            case msg == "Invalid event string":
                sndData = "{}"
                marshalErr = errors.New("mockX")
            case msg == "Queue error:":
                sndData = "{}"
                qErr = errors.New("mockY")
            }

            var mockFn *gomonkey.Patches = nil
            if marshalErr != nil {
                cmn.LogDebug("Mocked Marshal")
                mockFn = gomonkey.ApplyFunc(json.Marshal, func(v any) ([]byte, error) {
                    return nil, marshalErr
                })
            } else if qErr != nil {
                cmn.LogDebug("Mocked for Q")
                mockFn = gomonkey.ApplyMethod(reflect.TypeOf(&queue.PriorityQueue{}), "Put",
                    func(pq *queue.PriorityQueue, item ...queue.Item) error {
                        return qErr
                    })
            } else {
                cmn.LogDebug("Mocked for None")
            }

            /*
               mockLen := gomonkey.ApplyMethod(reflect.TypeOf(&queue.PriorityQueue{}), "Len",
                       func(pq *queue.PriorityQueue) int {
                           return 0
                   })
               mockLen.Reset()
            */

            logMsgs := []string{}

            mockErr := gomonkey.ApplyFunc(cmn.LogError, func(s string, a ...interface{}) error {
                logMsgs = append(logMsgs, s)
                return nil
            })
            defer mockErr.Reset()

            mockCrit := gomonkey.ApplyFunc(cmn.LogCritical, func(s string, a ...interface{}) error {
                logMsgs = append(logMsgs, s)
                return nil
            })
            defer mockCrit.Reset()

            sendData(cl, sndData)

            if mockFn != nil {
                mockFn.Reset()
                cmn.LogDebug("Mock reset")
            }

            if !testLogs(logMsgs, msg) {
                t.Fatalf("Expected msg(%s) not found", msg)
            }
        }
        cmn.LogDebug("Test sendData END   =================")
    }
    {
        /* Test Sent One */
        cl := &LoMDataClient{}

        val := &Value{&lpb.Value{SendIndex: 1}}
        cl.lastReportedSentIndex = 10

        logMsgs := []string{}

        mockErr := gomonkey.ApplyFunc(cmn.LogError, func(s string, a ...interface{}) error {
            logMsgs = append(logMsgs, s)
            return nil
        })
        defer mockErr.Reset()

        cl.SentOne(val)

        msg := "Internal indices issue sentIndex"
        if !testLogs(logMsgs, msg) {
            t.Fatalf("Expected msg(%s) not found", msg)
        }
        if nil != cl.Close() {
            t.Fatalf("Expected nil for LoMDataClient.Close")
        }
    }
}
