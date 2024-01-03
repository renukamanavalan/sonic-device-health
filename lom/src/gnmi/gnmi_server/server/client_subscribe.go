package gnmi

import (
    "fmt"
    "io"
    "net"
    "sync"

    "github.com/Workiva/go-datastructures/queue"
    gnmipb "github.com/openconfig/gnmi/proto/gnmi"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    ldc "lom/src/gnmi/lom_data_clients"
    cmn "lom/src/lib/lomcommon"
)

// Client contains information about a subscribe client that has connected to the server.
type Client struct {
    addr      net.Addr
    sendMsg   int64
    recvMsg   int64
    errors    int64
    polled    chan struct{}
    stop      chan struct{}
    once      chan struct{}
    mu        sync.RWMutex
    q         *queue.PriorityQueue
    subscribe *gnmipb.SubscriptionList
    // Wait for all sub go routine to finish
    w sync.WaitGroup
}

// NewClient returns a new initialized client.
func NewClient(addr net.Addr) *Client {
    pq := queue.NewPriorityQueue(1, false)
    return &Client{
        addr: addr,
        q:    pq,
    }
}

// String returns the target the client is querying.
func (c *Client) String() string {
    return c.addr.String()
}

// Populate data path from prefix and subscription path.
func (c *Client) populatePathSubscription(sublist *gnmipb.SubscriptionList) (*gnmipb.Path, error) {

    cmn.LogInfo("prefix : %#v SubscribRequest : %#v", sublist.GetPrefix(), sublist)

    subscriptions := sublist.GetSubscription()
    if len(subscriptions) != 1 {
        return nil, fmt.Errorf("Expect only one subscription per request")
    }

    path := subscriptions[0].GetPath()

    cmn.LogInfo("gnmi Path : %v", path)
    return path, nil
}

// Run starts the subscribe client. The first message received must be a
// SubscriptionList. Once the client is started, it will run until the stream
// is closed or the schedule completes. For Poll queries the Run will block
// internally after sync until a Poll request is made to the server.
// Refer: doc/gNMI_Info.txt
func (c *Client) Run(stream gnmipb.GNMI_SubscribeServer) (err error) {
    defer cmn.LogInfo("Client %s shutdown", c)

    if stream == nil {
        return grpc.Errorf(codes.FailedPrecondition, "cannot start client: stream is nil")
    }

    var connectionKey string
    var valid bool

    defer func() {
        if err != nil {
            c.errors++
        }
    }()

    /* Recv returns SubscribeRequest - Refer: doc/gNMI_Info.txt */
    query, err := stream.Recv()
    c.recvMsg++
    if err != nil {
        if err == io.EOF {
            return grpc.Errorf(codes.Aborted, "stream EOF received before init")
        }
        return grpc.Errorf(grpc.Code(err), "received error from client")
    }

    cmn.LogInfo("Client %s recieved initial query %v", c, query)

    /* Return SubscriptionList - Refer: doc/gNMI_Info.txt */
    c.subscribe = query.GetSubscribe()

    if c.subscribe == nil {
        return grpc.Errorf(codes.InvalidArgument, "first message must be SubscriptionList: %q", query)
    }

    /*
     * Prefix used for all paths in the request. Type gNMI.Path - Refer: doc/gNMI_Info.txt
     * If two paths to be given are /a/b/c & /a/b/d, then one may set prefix as "/a/b" and
     * individual paths that follow may only say "c" or "d"
     *
     * Refer: doc/gNMI_Info.txt
     */
    prefix := c.subscribe.GetPrefix()

    /* target specified the source of data and expected in prefix only. Refer: doc/gNMI_Info.txt */
    target := prefix.GetTarget()

    path, err := c.populatePathSubscription(c.subscribe)
    if err != nil {
        return grpc.Errorf(codes.NotFound, "Invalid subscription path: %v %q", err, query)
    }

    var dc ldc.Client

    mode := c.subscribe.GetMode()

    cmn.LogError("mode=%v, origin=%q, target=%q", mode, origin, target)

    if ((target == "COUNTERS") || (target == "EVENTS")) &&
        (mode == gnmipb.SubscriptionList_STREAM) {
        dc, err = ldc.NewLoMDataClient(path, prefix, target)
    } else {
        return grpc.Errorf(codes.NotFound, "target=%v mode=%v", target, mode)
    }

    switch mode {
    case gnmipb.SubscriptionList_STREAM:
        c.stop = make(chan struct{}, 1)
        c.w.Add(1)
        go dc.StreamRun(c.q, c.stop, &c.w, c.subscribe)
        /*
           Not supported yet for subscription
               case gnmipb.SubscriptionList_POLL:
                   c.polled = make(chan struct{}, 1)
                   c.polled <- struct{}{}
                   c.w.Add(1)
                   go dc.PollRun(c.q, c.polled, &c.w, c.subscribe)
               case gnmipb.SubscriptionList_ONCE:
                   c.once = make(chan struct{}, 1)
                   c.once <- struct{}{}
                   c.w.Add(1)
                   go dc.OnceRun(c.q, c.once, &c.w, c.subscribe)
        */
    default:
        return grpc.Errorf(codes.InvalidArgument, "Unkown subscription mode: %q", query)
    }

    cmn.LogInfo("Client %s running", c)
    go c.recv(stream)
    err = c.send(stream, dc)
    c.Close()
    // Wait until all child go routines exited
    c.w.Wait()
    return grpc.Errorf(codes.InvalidArgument, "%s", err)
}

// Closing of client queue is triggered upon end of stream receive or stream error
// or fatal error of any client go routine .
// it will cause cancle of client context and exit of the send goroutines.
func (c *Client) Close() {
    c.mu.Lock()
    defer c.mu.Unlock()
    cmn.LogInfo("Client %s Close, sendMsg %v recvMsg %v errors %v", c, c.sendMsg, c.recvMsg, c.errors)
    if c.q != nil {
        if c.q.Disposed() {
            return
        }
        c.q.Dispose()
    }
    if c.stop != nil {
        close(c.stop)
    }
    if c.polled != nil {
        close(c.polled)
    }
    if c.once != nil {
        close(c.once)
    }
}

func (c *Client) recv(stream gnmipb.GNMI_SubscribeServer) {
    defer c.Close()

    for {
        cmn.LogInfo("Client %s blocking on stream.Recv()", c)
        event, err := stream.Recv()
        c.recvMsg++

        switch err {
        default:
            cmn.LogError("Client %s received error: %v", c, err)
            return
        case io.EOF:
            cmn.LogError("Client %s received io.EOF", c)
            if c.subscribe.Mode == gnmipb.SubscriptionList_STREAM {
                // The client->server could be closed after the sending the subscription list.
                // EOF is not a indication of client is not listening.
                // Instead stream.Context() which is signaled once the underlying connection is terminated.
                cmn.LogInfo("Waiting for client '%s'", c)
                // This context is done when the client connection is terminated.
                <-stream.Context().Done()
                cmn.LogInfo("Client is done '%s'", c)
            }
            return
        case nil:
        }

        if c.subscribe.Mode == gnmipb.SubscriptionList_POLL {
            cmn.LogError("Client %s received Poll event: %v", c, event)
            if _, ok := event.Request.(*gnmipb.SubscribeRequest_Poll); !ok {
                return
            }
            c.polled <- struct{}{}
            continue
        }
        cmn.LogInfo("Client %s received invalid event: %s", c, event)
    }
}

// send runs until process Queue returns an error.
func (c *Client) send(stream gnmipb.GNMI_SubscribeServer, dc ldc.Client) error {
    for {
        var val *ldc.Value
        items, err := c.q.Get(1)

        if items == nil {
            cmn.LogInfo("%v", err)
            return err
        }
        if err != nil {
            c.errors++
            cmn.LogError("%v", err)
            return fmt.Errorf("unexpected queue Gext(1): %v", err)
        }

        var resp *gnmipb.SubscribeResponse

        switch v := items[0].(type) {
        case ldc.Value:
            if resp, err = ldc.ValToResp(v); err != nil {
                c.errors++
                return err
            }
            val = &v
        default:
            cmn.LogError("Unknown data type %v for %s in queue", items[0], c)
            c.errors++
        }

        c.sendMsg++
        err = stream.Send(resp)
        if err != nil {
            cmn.LogError("Client %s sending error:%v", c, err)
            c.errors++
            dc.FailedSend()
            return err
        }

        dc.SentOne(val)
        cmn.LogInfo("Client %s done sending, msg count %d, msg %v", c, c.sendMsg, resp)
    }
}
