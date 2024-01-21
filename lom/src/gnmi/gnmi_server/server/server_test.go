package gnmi

// server_test covers gNMI get, subscribe (stream and poll) test
// Prerequisite: redis-server should be running.
import (
    "crypto/tls"
    "flag"
    "fmt"
    "strings"

    "os/user"
    "testing"
    "time"

    "github.com/openconfig/gnmi/client"
    "github.com/openconfig/ygot/ygot"

    pb "github.com/openconfig/gnmi/proto/gnmi"

    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"

    "github.com/agiledragon/gomonkey/v2"

    // Register supported client types.
    testcert "lom/src/gnmi/testdata/tls"
    tele "lom/src/lib/lomtelemetry"
)

func createServer(t *testing.T, port int64) *Server {
    t.Helper()
    certificate, err := testcert.NewCert()
    if err != nil {
        t.Fatalf("could not load server key pair: %s", err)
    }
    tlsCfg := &tls.Config{
        ClientAuth:   tls.RequestClientCert,
        Certificates: []tls.Certificate{certificate},
    }

    opts := []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsCfg))}
    cfg := &Config{Port: port, EnableNativeWrite: true, Threshold: 100}
    s, err := NewServer(cfg, opts)
    if err != nil {
        t.Errorf("Failed to create gNMI server: %v", err)
    }
    return s
}

func createAuthServer(t *testing.T, port int64) *Server {
    t.Helper()
    certificate, err := testcert.NewCert()
    if err != nil {
        t.Fatalf("could not load server key pair: %s", err)
    }
    tlsCfg := &tls.Config{
        ClientAuth:   tls.RequestClientCert,
        Certificates: []tls.Certificate{certificate},
    }

    opts := []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsCfg))}
    cfg := &Config{Port: port, UserAuth: AuthTypes{"password": true, "cert": true, "jwt": true}}
    s, err := NewServer(cfg, opts)
    if err != nil {
        t.Fatalf("Failed to create gNMI server: %v", err)
    }
    return s
}

func createInvalidServer(t *testing.T, port int64) *Server {
    certificate, err := testcert.NewCert()
    if err != nil {
        t.Errorf("could not load server key pair: %s", err)
    }
    tlsCfg := &tls.Config{
        ClientAuth:   tls.RequestClientCert,
        Certificates: []tls.Certificate{certificate},
    }

    opts := []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsCfg))}
    s, err := NewServer(nil, opts)
    if err != nil {
        return nil
    }
    return s
}

func runServer(t *testing.T, s *Server) {
    //t.Log("Starting RPC server on address:", s.Address())
    err := s.Serve() // blocks until close
    if err != nil {
        t.Fatalf("gRPC server err: %v", err)
    }
    //t.Log("Exiting RPC server on address", s.Address())
}

// subscriptionQuery represent the input to create an gnmi.Subscription instance.
type subscriptionQuery struct {
    Query          []string
    SubMode        pb.SubscriptionMode
    SampleInterval uint64
}

func pathToString(q client.Path) string {
    qq := make(client.Path, len(q))
    copy(qq, q)
    // Escape all slashes within a path element. ygot.StringToPath will handle
    // these escapes.
    for i, e := range qq {
        qq[i] = strings.Replace(e, "/", "\\/", -1)
    }
    return strings.Join(qq, "/")
}

// createQuery creates a client.Query with the given args. It assigns query.SubReq.
func createQuery(subListMode pb.SubscriptionList_Mode, target string, queries []subscriptionQuery, updatesOnly bool) (*client.Query, error) {
    s := &pb.SubscribeRequest_Subscribe{
        Subscribe: &pb.SubscriptionList{
            Mode:   subListMode,
            Prefix: &pb.Path{Target: target},
        },
    }
    if updatesOnly {
        s.Subscribe.UpdatesOnly = true
    }

    for _, qq := range queries {
        pp, err := ygot.StringToPath(pathToString(qq.Query), ygot.StructuredPath, ygot.StringSlicePath)
        if err != nil {
            return nil, fmt.Errorf("invalid query path %q: %v", qq, err)
        }
        s.Subscribe.Subscription = append(
            s.Subscribe.Subscription,
            &pb.Subscription{
                Path:           pp,
                Mode:           qq.SubMode,
                SampleInterval: qq.SampleInterval,
            })
    }

    subReq := &pb.SubscribeRequest{Request: s}
    query, err := client.NewQuery(subReq)
    query.TLS = &tls.Config{InsecureSkipVerify: true}
    return &query, err
}

// createQueryOrFail creates a query, in case of a failure it fails the test.
func createQueryOrFail(t *testing.T, subListMode pb.SubscriptionList_Mode, target string, queries []subscriptionQuery, updatesOnly bool) client.Query {
    q, err := createQuery(subListMode, target, queries, updatesOnly)
    if err != nil {
        t.Fatalf("failed to create query: %v", err)
    }

    return *q
}

// create query for subscribing to events.
func createEventsQuery(t *testing.T, paths ...string) client.Query {
    return createQueryOrFail(t,
        pb.SubscriptionList_STREAM,
        "EVENTS",
        []subscriptionQuery{
            {
                Query:   paths,
                SubMode: pb.SubscriptionMode_ON_CHANGE,
            },
        },
        false)
}

type tablePathValue struct {
    dbName    string
    tableName string
    tableKey  string
    delimitor string
    field     string
    value     string
    op        string
}

func TestCapabilities(t *testing.T) {
    //t.Log("Start server")
    s := createServer(t, 8085)
    go runServer(t, s)

    // prepareDb(t)

    //t.Log("Start gNMI client")
    tlsConfig := &tls.Config{InsecureSkipVerify: true}
    opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}

    //targetAddr := "30.57.185.38:8080"
    targetAddr := "127.0.0.1:8085"
    conn, err := grpc.Dial(targetAddr, opts...)
    if err != nil {
        t.Fatalf("Dialing to %q failed: %v", targetAddr, err)
    }
    defer conn.Close()

    gClient := pb.NewGNMIClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var req pb.CapabilityRequest
    resp, err := gClient.Capabilities(ctx, &req)
    if err == nil {
        t.Fatalf("Failed to not get Capabilities")
    }
    t.Logf("TODO: Verify capability (%v)", resp)

}

type loginCreds struct {
    Username, Password string
}

func (c *loginCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
    return map[string]string{
        "username": c.Username,
        "password": c.Password,
    }, nil
}

func (c *loginCreds) RequireTransportSecurity() bool {
    return true
}

func TestAuthCapabilities(t *testing.T) {
    mock1 := gomonkey.ApplyFunc(UserPwAuth, func(username string, passwd string) (bool, error) {
        return true, nil
    })
    defer mock1.Reset()

    s := createAuthServer(t, 8089)
    go runServer(t, s)
    defer s.s.Stop()

    currentUser, _ := user.Current()
    tlsConfig := &tls.Config{InsecureSkipVerify: true}
    cred := &loginCreds{Username: currentUser.Username, Password: "dummy"}
    opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)), grpc.WithPerRPCCredentials(cred)}

    targetAddr := "127.0.0.1:8089"
    conn, err := grpc.Dial(targetAddr, opts...)
    if err != nil {
        t.Fatalf("Dialing to %q failed: %v", targetAddr, err)
    }
    defer conn.Close()

    gClient := pb.NewGNMIClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var req pb.CapabilityRequest
    resp, err := gClient.Capabilities(ctx, &req)
    if err != nil {
        t.Fatalf("Failed to get Capabilities: %v", err)
    }
    if len(resp.SupportedModels) == 0 {
        t.Fatalf("No Supported Models found!")
    }
}

func TestEventsClient(t *testing.T) {
    HEARTBEAT_SET := 5

    tests := []struct {
        desc    string
        target  string
        pubData []string
        expErr  string
        rcvData []string
    }{
        {
            desc:   "New Data client fail - invalid target",
            target: "xyz",
            expErr: "Unexpected target=(xyz)",
        },
    }

    events := []string{
        `{ "index": 0, "foo0": "bar" }`,
        `{ "index": 1, "foo1": "bar" }`,
        `{ "index": 2, "foo2": "bar" }`,
        `{ "index": 3, "foo3": "bar" }`,
    }

    /* We need to publish events, so LoMDataClient will be able to send it
     * to gNMI client. Need Proxy to connect publisher and subscriber.
     * Publisher is explicitly created below.
     * LoMDataClient creates subscriber as internal data source.
     */
    var chPrxy chan<- int
    if ch, err := tele.RunPubSubProxy(tele.CHANNEL_TYPE_EVENTS); err != nil {
        t.Fatalf("Failed to RunPubSubProxy for events. err (%v)", err)
    } else {
        chPrxy = ch
    }
    defer close(chPrxy)

    s := createServer(t, 8081)
    go runServer(t, s)

    /* Build query */
    qstr := fmt.Sprintf("all[heartbeat=%d]", HEARTBEAT_SET)
    q := createEventsQuery(t, qstr)
    q.Addrs = []string{"127.0.0.1:8081"}

    for testNum, tt := range tests {
        t.Run(tt.desc, func(t *testing.T) {
            /* Create new gnmi client */
            c := client.New()
            defer c.Close()

            rcvdEventsCh := make(chan string, len(events))
            defer close(rcvdEventsCh)

            /* Receive notifications (which is events) from server */
            q.NotificationHandler = func(n client.Notification) error {
                if nn, ok := n.(client.Update); ok {
                    rcvdEventsCh <- fmt.Sprintf("%v", nn.Val)
                }
                return nil
            }

            /* Client sends subscribe req to server which will create
             * NewLoMDataClient. Dataclient will create internal subscribe
             * request tele.GetSubChannel for events
             */
            go func() {
                c.Subscribe(context.Background(), q)
            }()

            /* Tele pub/sub channel are ZMQ based, which is async. Hence pause
             * half second to let subscribe gets created
             */
            time.Sleep(500 * time.Millisecond)

            var pubCh chan<- tele.JsonString_t
            if ch, err := tele.GetPubChannel(tele.CHANNEL_TYPE_EVENTS, tele.CHANNEL_PRODUCER_OTHER,
                "test"); err != nil {
                t.Fatalf("test(%d): Failed to get Pubchannel err: (%v)", testNum, err)
            } else {
                pubCh = ch
            }

            defer close(pubCh)

            /* Publish data via LoM Telemetry Pub channel, which will be received by
             * LoMDataClient via LoM telemetry subchannel and send the same to gnmi
             * client via notification handler.
             */
            for _, ev := range events {
                pubCh <- tele.JsonString_t(ev)
            }

            /* Verify received notifications by gnmi client */
            for i := 0; i < len(events); i++ {
                select {
                case val := <-rcvdEventsCh:
                    if val != events[i] {
                        t.Fatalf("test(%d): index(%d): Rcvd (%s) != sent (%s)", testNum, i, val, events[i])
                    }
                case <-time.After(time.Second):
                    t.Fatalf("test(%d): Timeout: rcvd (%d) expect(%d)", testNum, i, len(events))
                }
            }
        })
    }
    s.s.Stop()
}

func TestServerPort(t *testing.T) {
    s := createServer(t, -8080)
    port := s.Port()
    if port != 0 {
        t.Errorf("Invalid port: %d", port)
    }
    s.s.Stop()
}

func TestInvalidServer(t *testing.T) {
    s := createInvalidServer(t, 9000)
    if s != nil {
        t.Errorf("Should not create invalid server")
    }
}

func init() {
    // Enable logs at UT setup
    flag.Lookup("v").Value.Set("10")
    flag.Lookup("log_dir").Value.Set("/tmp/telemetrytest")
}