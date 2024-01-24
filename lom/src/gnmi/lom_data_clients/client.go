// Package client provides a generic access layer for data available in system
package client

import (
    "sync"

    "github.com/Workiva/go-datastructures/queue"
    gnmipb "github.com/openconfig/gnmi/proto/gnmi"

    lpb "lom/src/gnmi/proto"
    cmn "lom/src/lib/lomcommon"
)

// Client defines a set of methods which every client must implement.
// This package provides one implmentation for now: the DbClient
type Client interface {
    // StreamRun will start watching service on data source
    // and enqueue data change to the priority queue.
    // It stops all activities upon receiving signal on stop channel
    // It should run as a go routine
    StreamRun(q *queue.PriorityQueue, stop chan struct{}, w *sync.WaitGroup, subscribe *gnmipb.SubscriptionList)

    // Close provides implemenation for explicit cleanup of Client
    Close() error

    // callbacks on send failed
    FailedSend()

    // callback on sent
    SentOne(*Value)
}

type Stream interface {
    Send(m *gnmipb.SubscribeResponse) error
}

type Value struct {
    *lpb.Value
}

/* Implement Compare method for priority queue
 *
 * https://pkg.go.dev/github.com/golang-collections/go-datastructures/queue
 * type Item interface {
 *     Compare returns a bool that can be used to determine
 *     ordering in the priority queue.  Assuming the queue
 *     is in ascending order, this should return > logic.
 *     Return 1 to indicate this object is greater than the
 *     the other logic, 0 to indicate equality, and -1 to indicate
 *     less than other.
 *
 *     Compare(other Item) int
 * }
 */
func (val Value) Compare(other queue.Item) (ret int) {
    oval := other.(Value)
    if val.GetSendIndex() > oval.GetSendIndex() {
        ret = 1
    } else if val.GetSendIndex() == oval.GetSendIndex() {
        ret = 0
    } else {
        ret = -1
    }
    return
}

func (val Value) GetTimestamp() int64 {
    return val.Value.GetTimestamp()
}

// Convert from LoM Value (as defined in Proto) to its corresponding gNMI proto stream
// response type.
func ValToResp(val Value) (*gnmipb.SubscribeResponse, error) {
    return &gnmipb.SubscribeResponse{
        Response: &gnmipb.SubscribeResponse_Update{
            Update: &gnmipb.Notification{
                Timestamp: val.GetTimestamp(),
                Prefix:    val.GetPrefix(),
                Update: []*gnmipb.Update{
                    {
                        Path: val.GetPath(),
                        Val:  val.GetVal(),
                    },
                },
            },
        },
    }, nil
}
