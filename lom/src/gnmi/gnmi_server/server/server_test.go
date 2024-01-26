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
    "google.golang.org/grpc/metadata"

    "github.com/agiledragon/gomonkey/v2"

    // Register supported client types.
    gclient "github.com/jipanyang/gnmi/client/gnmi"
    testcert "lom/src/gnmi/testdata/tls"
    lom_utils "lom/src/gnmi/utils"
    cmn "lom/src/lib/lomcommon"
    tele "lom/src/lib/lomtelemetry"
)

var clientTypes = []string{gclient.Type}

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
func createEventsQuery(t *testing.T, target string, paths ...string) client.Query {
    return createQueryOrFail(t,
        pb.SubscriptionList_STREAM,
        target,
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
    if err != nil {
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
    if len(resp.SupportedModels) != 0 {
        t.Fatalf("Expect: No Supported Models. But found (%d)", len(resp.SupportedModels))
    }
}

func compare_maps(s map[string]any, d map[string]any) (bool, string) {
    if len(s) != len(d) {
        return false, fmt.Sprintf("len mismatch (%d) != (%d)", len(s), len(d))
    }
    for k, v := range s {
        if v != d[k] {
            return false, fmt.Sprintf("key:%s (%T)(%v) != (%T)(%v)", k, v, v, d[k], d[k])
        }
    }
    return true, ""
}

func TestEventsClient(t *testing.T) {
    HEARTBEAT_SET := 5

    evts := [10]map[string]any{}

    for i := 0; i < len(evts); i++ {
        evts[i] = map[string]any{"index": float64(i), "foo": fmt.Sprintf("bar_%d", i)}
    }

    tests := []struct {
        desc   string
        target string
        pubCnt int
        rcvCnt int
        expErr string
    }{
        {
            desc:   "New Data client fail for invalid target",
            target: "xyz",
            expErr: "rpc error: code = NotFound desc = target=xyz mode=STREAM",
        },
        {
            desc:   "New Data client succeed",
            target: "EVENTS",
            pubCnt: 4,
            rcvCnt: 4,
        },
    }

    s := createServer(t, 8081)
    defer s.s.Stop()

    go runServer(t, s)

    /* To get client data, simulate events publish. To do so, we need to
     * init service & publish.
     */
    if err := tele.TelemetryServiceInit(); err != nil {
        t.Fatalf("Failed to call TelemetryServiceInit. err (%v)", err)
    }
    defer tele.TelemetryServiceShut()

    if err := tele.PublishInit(tele.CHANNEL_PRODUCER_OTHER, "TestEventsClient"); err != nil {
        t.Fatalf("Failed to call tele.PublishInit. err (%v)", err)
    }
    defer tele.PublishTerminate()

    for testNum, tt := range tests {
        t.Run(tt.desc, func(t *testing.T) {
            /* Create new gnmi client */
            var errFail error
            cmn.LogInfo("test(%d): START    ------------------", testNum)

            c := client.New()

            /* Create buffered channel for max expected to help not block.*/
            rcvdEventsCh := make(chan map[string]any, tt.rcvCnt)

            defer func() {
                /* Close client before closing channel used by notification handler */
                c.Close()
                close(rcvdEventsCh)
                cmn.LogInfo("client closed")
            }()

            /* Build query */
            qstr := fmt.Sprintf("all[heartbeat=%d]", HEARTBEAT_SET)
            q := createEventsQuery(t, tt.target, qstr)
            q.Addrs = []string{"127.0.0.1:8081"}

            /* Receive notifications (which is events) from server */
            q.NotificationHandler = func(n client.Notification) error {
                if nn, ok := n.(client.Update); ok {
                    if v, ok := nn.Val.(map[string]any); ok {
                        rcvdEventsCh <- v
                    } else {
                        cmn.LogError("Notification (%T) != map[string]any", nn.Val)
                    }
                }
                return nil
            }

            /* Client sends subscribe req to server which will create
             * NewLoMDataClient. Dataclient will create internal subscribe
             * request tele.GetSubChannel for events
             */
            go func() {
                /* https://github.com/openconfig/gnmi/blob/master/subscribe/subscribe.go */
                errFail = c.Subscribe(context.Background(), q)
                cmn.LogInfo("c.Subscribe: err=(%v)", errFail)
            }()

            /* Subscribe request creates a new LoMDataClient which in turn subscribes
             * internally for local events. Internal pub/sub is ZMQ based, which is async.
             * Hence pause half second to let subscribe gets created.
             * More over the c.Subscribe itself could fail. So this pause helps assess.
             */
            time.Sleep(500 * time.Millisecond)

            if tt.pubCnt != 0 {
                /* Publish data via LoM Telemetry Pub channel, which will be received by
                 * LoMDataClient via LoM telemetry subchannel and send the same to gnmi
                 * client via notification handler.
                 */
                for _, ev := range evts[:tt.pubCnt] {
                    if err := tele.PublishEvent(ev); err != nil {
                        t.Fatalf("Failed to call PublishEvent. err(%v) ev(%v)", err, ev)
                    }
                }

                /* Verify received notifications by gnmi client */
                for i := 0; i < tt.rcvCnt; i++ {
                    select {
                    case val := <-rcvdEventsCh:
                        if res, msg := compare_maps(val, evts[i]); !res {
                            e := cmn.LogError("test[%d]: index(%d): msg(%s)",
                                testNum, i, msg)
                            if errFail == nil {
                                errFail = e
                            }
                        }
                    case <-time.After(time.Second):
                        t.Fatalf("test(%d): Timeout: rcvd (%d) expect(%d)", testNum, i, tt.rcvCnt)
                    }
                }
            }
            if tt.expErr != "" {
                if errFail == nil {
                    t.Fatalf("test(%d): Expect failure (%s)", testNum, tt.expErr)
                } else if tt.expErr != fmt.Sprint(errFail) {
                    t.Fatalf("CHECK: *************test(%d): Expect failure (%s) != failure (%v)",
                        testNum, tt.expErr, errFail)
                }
            } else if errFail != nil {
                t.Fatalf("test(%d): Unexpected failure (%v)", testNum, errFail)
            }
            cmn.LogInfo("test(%d): COMPLETE ------------------", testNum)
        })
    }
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

func TestBasicAuthenAndAuthor(t *testing.T) {
    reqCtx := lom_utils.RequestContext{}
    failures := []string{
        "Invalid context",
    }

    /*
       "No Username Provided",
       "Unauthenticated",
       "Invalid Password",
    */

    for _, msg := range failures {
        mocks := []*gomonkey.Patches{}
        switch {
        case msg == "Invalid context":
            mocks = append(mocks, gomonkey.ApplyFunc(lom_utils.GetContext,
                func(ctx context.Context) (*lom_utils.RequestContext, context.Context) {
                    return &reqCtx, ctx
                }))
            mocks = append(mocks, gomonkey.ApplyFunc(metadata.FromIncomingContext,
                func(ctx context.Context) (metadata.MD, bool) {
                    return metadata.MD{}, false
                }))
        default:
            t.Fatalf("Unhandled failure (%s)", msg)
        }

        defer func() {
            for _, m := range mocks {
                m.Reset()
            }
        }()

        var ctx context.Context
        c, err := BasicAuthenAndAuthor(ctx)
        if c != ctx {
            t.Fatalf("Failed to get context back")
        }
        if msg != "" {
            if err == nil {
                t.Fatalf("Failed to fail. Expect err(%s)", msg)
            } else if !strings.Contains(fmt.Sprint(err), msg) {
                t.Fatalf("err mismatch exp(%s) NOT in err(%v)", msg, err)
            }
        } else if err != nil {
            t.Fatalf("Failed NOT to fail. Res (%v)", err)
        }
    }
}

func init() {
    // Enable logs at UT setup
    flag.Lookup("v").Value.Set("10")
    flag.Lookup("log_dir").Value.Set("/tmp/telemetrytest")
}
