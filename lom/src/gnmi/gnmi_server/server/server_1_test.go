package gnmi

// server_test covers gNMI get, subscribe (stream and poll) test
// Prerequisite: redis-server should be running.
import (
    "errors"
    "fmt"
    "io"
    "net"
    "reflect"
    "strings"
    "testing"
    "time"

    "github.com/Workiva/go-datastructures/queue"
    "github.com/agiledragon/gomonkey/v2"

    gnmipb "github.com/openconfig/gnmi/proto/gnmi"
    "golang.org/x/net/context"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/peer"

    ldc "lom/src/gnmi/lom_data_clients"
    lpb "lom/src/gnmi/proto"
    cmn "lom/src/lib/lomcommon"
)

type gnmiSubsServer struct{}

func (*gnmiSubsServer) Send(*gnmipb.SubscribeResponse) error {
    return nil
}

func (*gnmiSubsServer) Recv() (*gnmipb.SubscribeRequest, error) {
    return nil, nil
}

func (*gnmiSubsServer) SetHeader(metadata.MD) error {
    return nil
}

func (*gnmiSubsServer) SendHeader(metadata.MD) error {
    return nil
}

func (*gnmiSubsServer) SetTrailer(metadata.MD) {
}

func (*gnmiSubsServer) Context() context.Context {
    return nil
}

func (*gnmiSubsServer) SendMsg(m any) error {
    return nil
}

func (*gnmiSubsServer) RecvMsg(m any) error {
    return nil
}

type ctxContext struct{}

func (*ctxContext) Deadline() (deadline time.Time, ok bool) {
    return time.Now(), true
}

func (*ctxContext) Done() <-chan struct{} {
    return nil
}

func (*ctxContext) Err() error {
    return nil
}

func (*ctxContext) Value(key any) any {
    return nil
}

type netAddr struct {
    network string
    addr    string
}

func (n *netAddr) Network() string {
    return n.network
}

func (n *netAddr) String() string {
    return n.addr
}

func TestPopulatePathSubscription(t *testing.T) {
    slist := gnmipb.SubscriptionList{}

    {
        c := Client{}

        if ret, err := c.populatePathSubscription(&slist); (ret != nil) || (err == nil) {
            t.Fatalf("Failed to fail Client.populatePathSubscription ret(%v) err(%v)", ret, err)
        }

        if err := c.Run(nil); err == nil {
            t.Fatalf("Failed to fail Client.Run err(%v)", err)
        }
    }

    {
        /* Test error paths of func (c *Client) Run */

        c := Client{}
        var i = gnmiSubsServer{}
        var j gnmipb.GNMI_SubscribeServer = &i
        path := gnmipb.Path{}
        sr := gnmipb.SubscribeRequest{}
        sl := gnmipb.SubscriptionList{Prefix: &path}
        slS := gnmipb.SubscriptionList{
            Prefix:       &path,
            Subscription: []*gnmipb.Subscription{&gnmipb.Subscription{}},
            Mode:         gnmipb.SubscriptionList_POLL,
        }
        slM := gnmipb.SubscriptionList{
            Prefix:       &path,
            Subscription: []*gnmipb.Subscription{&gnmipb.Subscription{}},
            Mode:         gnmipb.SubscriptionList_STREAM,
        }
        //slNil := gnmipb.SubscriptionList{}

        lst := map[string]struct {
            err error
            sl  *gnmipb.SubscriptionList
        }{
            "stream EOF received before init":        {io.EOF, nil},
            "received error from client":             {errors.New("mock"), nil},
            "first message must be SubscriptionList": {nil, nil},
            "Invalid subscription path":              {nil, &sl},
            "Unkown subscription mode":               {nil, &slS},
            "Unknown target":                         {nil, &slM},
        }

        for s, e := range lst {
            mockTmp := gomonkey.ApplyMethod(reflect.TypeOf(&gnmiSubsServer{}), "Recv",
                func() (*gnmipb.SubscribeRequest, error) {
                    return &sr, e.err
                })
            defer mockTmp.Reset()

            mockSr := gomonkey.ApplyMethod(reflect.TypeOf(&gnmipb.SubscribeRequest{}), "GetSubscribe",
                func() *gnmipb.SubscriptionList {
                    return e.sl
                })
            defer mockSr.Reset()

            if ret := c.Run(j); (ret == nil) ||
                !strings.Contains(fmt.Sprint(ret), s) {
                t.Fatalf("Failed to fail Client.Run ret(%v) expect e(%v) s(%s)", ret, e, s)
            }
        }
    }

    {
        /* Test error path of func (c *Client) recv */

        testLst := map[string][]error{
            "Client is done":         []error{io.EOF},
            "received invalid event": []error{nil, errors.New("foo")},
        }

        for msg, recvErr := range testLst {
            errIndex := 0
            c := Client{}
            srv := gnmiSubsServer{}
            var srvG gnmipb.GNMI_SubscribeServer = &srv
            ctx := ctxContext{}
            var cctx context.Context = &ctx

            mockTmp := gomonkey.ApplyMethod(reflect.TypeOf(&gnmiSubsServer{}), "Recv",
                func() (*gnmipb.SubscribeRequest, error) {
                    e := recvErr[errIndex]
                    errIndex++
                    if errIndex >= len(recvErr) {
                        errIndex = 0
                    }
                    cmn.LogInfo("mock Recv err (%v)", e)
                    return nil, e
                })
            defer mockTmp.Reset()

            c.subscribe = &gnmipb.SubscriptionList{Mode: gnmipb.SubscriptionList_STREAM}

            mockCtx := gomonkey.ApplyMethod(reflect.TypeOf(&gnmiSubsServer{}), "Context",
                func() context.Context {
                    return cctx
                })
            defer mockCtx.Reset()

            ch := make(chan struct{}, 1)
            mockDone := gomonkey.ApplyMethod(reflect.TypeOf(&ctxContext{}), "Done",
                func() <-chan struct{} {
                    ch <- struct{}{}
                    return ch
                })
            defer mockDone.Reset()

            logMsgs := []string{}
            mockLog := gomonkey.ApplyFunc(cmn.LogInfo, func(s string, a ...interface{}) {
                logMsgs = append(logMsgs, s)
                t.Logf("Mocked log: (%s)", s)
            })
            defer mockLog.Reset()

            c.recv(srvG)

            found := false
            for _, logMsg := range logMsgs {
                if strings.Contains(logMsg, msg) {
                    found = true
                }
            }
            if !found {
                t.Fatalf("Failed to see log (%s)", msg)
            }
        }
    }
}

type mockVal struct{}

func (val mockVal) Compare(other queue.Item) int {
    return 0
}

func TestClientSend(t *testing.T) {
    {
        /* Test error path of func (c *Client) Send */

        lstTest := map[string]struct {
            items  []queue.Item
            getErr error
            sndErr error
            valErr error
            valRes *gnmipb.SubscribeResponse
        }{
            "Q.get failed with":      {getErr: errors.New("mock")},
            "Get received nil items": {},
            "Failed to convert to gnmipb.SubscribeResponse": {
                items:  []queue.Item{ldc.Value{&lpb.Value{}}},
                valErr: errors.New("mock"),
            },
            "Unknown data type": {items: []queue.Item{&mockVal{}}},
            "Client failing to send error": {
                items:  []queue.Item{ldc.Value{&lpb.Value{}}},
                sndErr: errors.New("mock"),
                valRes: &gnmipb.SubscribeResponse{},
            },
        }

        for msg, tdata := range lstTest {
            c := Client{}
            srv := gnmiSubsServer{}
            var srvG gnmipb.GNMI_SubscribeServer = &srv
            lomDc := ldc.LoMDataClient{}
            var dc ldc.Client = &lomDc

            mockSnd := gomonkey.ApplyMethod(reflect.TypeOf(&gnmiSubsServer{}), "Send",
                func(*gnmiSubsServer, *gnmipb.SubscribeResponse) error {
                    return tdata.sndErr
                })
            defer mockSnd.Reset()

            mockQ := gomonkey.ApplyMethod(reflect.TypeOf(&queue.PriorityQueue{}), "Get",
                func(pq *queue.PriorityQueue, i int) ([]queue.Item, error) {
                    return tdata.items, tdata.getErr
                })
            defer mockQ.Reset()

            mockResp := gomonkey.ApplyFunc(ldc.ValToResp,
                func(ldc.Value) (*gnmipb.SubscribeResponse, error) {
                    return tdata.valRes, tdata.valErr
                })
            defer mockResp.Reset()

            err := c.send(srvG, dc)

            if (err == nil) || !strings.Contains(fmt.Sprint(err), msg) {
                t.Fatalf("Failing to fail as expected (%s) != err(%v)", msg, err)
            }

        }
    }
}

func TestAuthTypes(t *testing.T) {
    //af := AuthTypes {"foo": false, "bar": false}

    tests := map[string]struct {
        auth       AuthTypes
        setStr     string
        expErr     bool
        unsetStr   string
        unexpErr   bool
        retStr     string
        retAny     bool
        retEnabled bool
    }{
        "init":  {setStr: "nil"},
        "empty": {auth: AuthTypes{}},
        "none":  {auth: AuthTypes{}, setStr: "none"},
        "foo": {
            auth:       AuthTypes{"foo": false, "bar": false},
            setStr:     "foo",
            retStr:     "foo",
            retAny:     true,
            retEnabled: true,
        },
        "barfoo": {
            auth:       AuthTypes{"foo": false, "bar": false},
            setStr:     "bar, foo",
            retStr:     "foo bar",
            retAny:     true,
            retEnabled: true,
        },
        "fooxxxbar": {
            auth:       AuthTypes{"foo": false, "bar": false},
            setStr:     "foo, xxx, bar",
            expErr:     true,
            retStr:     "foo",
            retAny:     true,
            retEnabled: true,
        },
        "unsetfoo": {
            auth:     AuthTypes{"foo": false, "bar": false},
            setStr:   "foo",
            unsetStr: "foo",
        },
        "unsetmissfoo": {
            auth:       AuthTypes{"foo": false, "bar": false},
            setStr:     "foo",
            unsetStr:   "xxx",
            unexpErr:   true,
            retStr:     "foo",
            retAny:     true,
            retEnabled: true,
        },
    }

    for tk, td := range tests {
        ap := &td.auth

        if td.setStr != "nil" {
            if err := ap.Set(td.setStr); td.expErr != (err != nil) {
                t.Fatalf("(%s): Failed set expErr(%v) err(%v)", tk, td.expErr, err)
            }
        }

        if td.unsetStr != "" {
            if err := ap.Unset(td.unsetStr); td.unexpErr != (err != nil) {
                t.Fatalf("(%s): Failed unset expErr(%v) err(%v)", tk, td.expErr, err)
            }
        }

        if s := strings.TrimSpace(ap.String()); s != td.retStr {
            t.Fatalf("(%s): Expect (%s) != (%s)", tk, td.retStr, s)
        }

        if ap.Any() != td.retAny {
            t.Fatalf("(%s): Expect (%v)", tk, td.retAny)
        }

        if ap.Enabled("foo") != td.retEnabled {
            t.Fatalf("(%s): Expect (%v)", tk, td.retEnabled)
        }
    }
    {
        /* Test corner case server.serve */
        s := Server{}
        if err := s.Serve(); err == nil {
            t.Fatalf("Expect server.Serve to fail")
        }
    }
}

func TestAuthenticate(t *testing.T) {

    ctxObj := ctxContext{}
    var cctx context.Context = &ctxObj
    logMsg := ""

    mockJwt := gomonkey.ApplyFunc(JwtAuthenAndAuthor,
        func(ctx context.Context) (*lpb.JwtToken, context.Context, error) {
            logMsg = "jwtAuth"
            return nil, ctx, nil
        })
    defer mockJwt.Reset()

    mockCert := gomonkey.ApplyFunc(ClientCertAuthenAndAuthor,
        func(ctx context.Context) (context.Context, error) {
            logMsg = "certAuth"
            return ctx, nil
        })
    defer mockCert.Reset()

    testAuth := map[string]AuthTypes{
        "jwtAuth":         AuthTypes{"jwt": true},
        "certAuth":        AuthTypes{"cert": true},
        "Unauthenticated": AuthTypes{"foo": true},
    }

    for msg, at := range testAuth {
        logMsg = ""
        _, err := authenticate(at, cctx)
        if err != nil {
            if !strings.Contains(fmt.Sprint(err), msg) {
                t.Fatalf("Expected err (%s) not in (%v)", msg, err)
            }
        } else if logMsg != msg {
            t.Fatalf("expect msg(%s) != logMsg(%s)", msg, logMsg)
        }
    }
}

func TestSubscribe(t *testing.T) {
    naddrObj := netAddr{"tcp", "10.10.10.10"}
    var nAddr net.Addr = &naddrObj

    /* To simulate duplicate client */
    cl := NewClient(nAddr)
    s := Server{
        config:  &Config{UserAuth: AuthTypes{"foo": true}},
        clients: map[string]*Client{cl.String(): cl},
    }

    ctxObj := ctxContext{}
    var cctx context.Context = &ctxObj

    pr := peer.Peer{Addr: net.Addr(nil)}
    pr_ok := false

    mockPeer := gomonkey.ApplyFunc(peer.FromContext,
        func(ctx context.Context) (*peer.Peer, bool) {
            return &pr, pr_ok
        })
    defer mockPeer.Reset()

    mockStr := gomonkey.ApplyMethod(reflect.TypeOf(&gnmiSubsServer{}), "Recv",
        func() (*gnmipb.SubscribeRequest, error) {
            return nil, errors.New("mock")
        })
    defer mockStr.Reset()

    mockCtx := gomonkey.ApplyMethod(reflect.TypeOf(&gnmiSubsServer{}), "Context",
        func() context.Context {
            return cctx
        })
    defer mockCtx.Reset()

    mockJwt := gomonkey.ApplyFunc(JwtAuthenAndAuthor,
        func(ctx context.Context) (*lpb.JwtToken, context.Context, error) {
            return nil, ctx, nil
        })
    defer mockJwt.Reset()

    var subs = gnmiSubsServer{}
    var gsubs gnmipb.GNMI_SubscribeServer = &subs

    testMsgs := []string{
        "Unauthenticated",
        "failed to get peer from ctx",
        "failed to get peer address",
        "received error from client",
    }

    for _, msg := range testMsgs {
        switch {
        case msg == "Unauthenticated":
        case msg == "failed to get peer from ctx":
            /* Pass authenticate with jwt mock above */
            s.config.UserAuth = AuthTypes{"jwt": true}
        case msg == "failed to get peer address":
            /* Let peer.FromContext succeed via mock above */
            pr_ok = true
        case msg == "received error from client":
            pr.Addr = nAddr
        }

        if err := s.Subscribe(gsubs); err == nil {
            t.Fatalf("%s: Failed to fail", msg)
        } else if !strings.Contains(fmt.Sprint(err), msg) {
            t.Fatalf("Expect msg(%s) != err(%v)", msg, err)
        }
    }

}

func TestServerMisc(t *testing.T) {
    s := Server{}

    ctxObj := ctxContext{}
    var cctx context.Context = &ctxObj

    if ret, err := s.Get(cctx, nil); (ret != nil) || (err == nil) {
        t.Fatalf("Failed to fail server.Get ret(%v) err(%v)", ret, err)
    }

    if ret, err := s.Set(cctx, nil); (ret != nil) || (err == nil) {
        t.Fatalf("Failed to fail server.Set ret(%v) err(%v)", ret, err)
    }

    if err := s.checkEncodingAndModel(gnmipb.Encoding_JSON_IETF, []*gnmipb.ModelData{}); err != nil {
        t.Fatalf("Expected to succeed. err(%v)", err)
    }

    if err := s.checkEncodingAndModel(gnmipb.Encoding_BYTES, []*gnmipb.ModelData{}); err == nil {
        t.Fatalf("Failed to fail for unknown encoding Encoding_BYTES")
    }

    s.config = &Config{UserAuth: AuthTypes{"foo": true}}
    if ret, err := s.Capabilities(cctx, nil); (ret != nil) || (err == nil) {
        t.Fatalf("Failed to fail server.Capabilities ret(%v) err(%v)", ret, err)
    }
}
