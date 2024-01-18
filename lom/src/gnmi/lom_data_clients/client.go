// Package client provides a generic access layer for data available in system
package client

import (
    "sync"

    "github.com/Workiva/go-datastructures/queue"
    gnmipb "github.com/openconfig/gnmi/proto/gnmi"

    lpb "lom/src/gnmi/proto"
)

// Client defines a set of methods which every client must implement.
// This package provides one implmentation for now: the DbClient
type Client interface {
    // StreamRun will start watching service on data source
    // and enqueue data change to the priority queue.
    // It stops all activities upon receiving signal on stop channel
    // It should run as a go routine
    StreamRun(q *queue.PriorityQueue, stop chan struct{}, w *sync.WaitGroup, subscribe *gnmipb.SubscriptionList)
    // Poll will  start service to respond poll signal received on poll channel.
    // data read from data source will be enqueued on to the priority queue
    // The service will stop upon detection of poll channel closing.
    // It should run as a go routine
    PollRun(q *queue.PriorityQueue, poll chan struct{}, w *sync.WaitGroup, subscribe *gnmipb.SubscriptionList)
    OnceRun(q *queue.PriorityQueue, once chan struct{}, w *sync.WaitGroup, subscribe *gnmipb.SubscriptionList)
    // Get return data from the data source in format of *lpb.Value
    Get(w *sync.WaitGroup) ([]*lpb.Value, error)
    // Set data based on path and value
    Set(delete []*gnmipb.Path, replace []*gnmipb.Update, update []*gnmipb.Update) error
    // Capabilities of the switch
    Capabilities() []gnmipb.ModelData

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

// Implement Compare method for priority queue
func (val Value) Compare(other queue.Item) int {
    oval := other.(Value)
    if val.GetTimestamp() > oval.GetTimestamp() {
        return 1
    } else if val.GetTimestamp() == oval.GetTimestamp() {
        return 0
    }
    return -1
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
