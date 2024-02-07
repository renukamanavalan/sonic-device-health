package client

import (
    "errors"
    "fmt"
    "reflect"
    "strings"
    "testing"

    "github.com/agiledragon/gomonkey/v2"

    gnmipb "github.com/openconfig/gnmi/proto/gnmi"

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


func TestLoMDataClient(t *testing.T) {
    {
        /* Test NewLoMDataClient */
        cntrPrefix := &gnmipb.Path{Target: "COUNTERS"}
        pathWithKeys :=  &gnmipb.Path {
            Elem: []*gnmipb.PathElem {
                &gnmipb.PathElem {
                    Key: map[string]string {
                        PARAM_UPD_FREQ: "100",  /* higher than max */
                        PARAM_QSIZE:    "10241",/* Value in range */
                        PARAM_ON_CHANGE:"false",/* Non default val */
                    },
                },
            },
        }

        lstTest := map[string] struct {
            path    *gnmipb.Path
            prefix  *gnmipb.Path
            subErr  error
            cl      *LoMDataClient
        } {
            "Unexpected target=": { prefix: &gnmipb.Path{Target: "Foo"} },
            "Failed to create LoMDataClient for": {
                prefix: cntrPrefix,
                path: &gnmipb.Path{},
                subErr: errors.New("mock"),
             },
            "": {
                prefix: cntrPrefix,
                path: pathWithKeys,
                cl: &LoMDataClient {
                    chType:     tele.CHANNEL_TYPE_COUNTERS,
                    chTypeStr:  tele.CHANNEL_TYPE_STR[tele.CHANNEL_TYPE_COUNTERS],
                    prefix:     cntrPrefix,
                    path:       pathWithKeys,
                    pq_max:     10241,
                    updFreq:    PARAM_UPD_FREQ_MIN,
                    onChg:      false,
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
                if _, ok := cl.(*LoMDataClient); !ok {
                    t.Fatalf("Expected LoMDataClient. Got (%T)", cl)
                }
            }
        }
    }
}

