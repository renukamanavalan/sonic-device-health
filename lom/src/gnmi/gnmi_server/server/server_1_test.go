package gnmi

// server_test covers gNMI get, subscribe (stream and poll) test
// Prerequisite: redis-server should be running.
import (
    "errors"
    "fmt"
    "io"
    "reflect"
    "strings"
    "testing"

    "github.com/agiledragon/gomonkey/v2"

    gnmipb "github.com/openconfig/gnmi/proto/gnmi"
    "golang.org/x/net/context"
    "google.golang.org/grpc/metadata"
    //cmn "lom/src/lib/lomcommon"
)

type gnmiSubsServer struct {}

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

func TestPopulatePathSubscription(t *testing.T) {
    slist := gnmipb.SubscriptionList{}

    c := Client{}
    
    if ret, err := c.populatePathSubscription(&slist); (ret != nil) || (err == nil) {
        t.Fatalf("Failed to fail Client.populatePathSubscription ret(%v) err(%v)", ret, err)
    }

    if err := c.Run(nil); err == nil {
        t.Fatalf("Failed to fail Client.Run err(%v)", err)
    }

    {
        var i = gnmiSubsServer{}
        var j gnmipb.GNMI_SubscribeServer = &i
        path := gnmipb.Path {}
        sr := gnmipb.SubscribeRequest{}
        sl := gnmipb.SubscriptionList{Prefix: &path}
        slS := gnmipb.SubscriptionList{
            Prefix: &path,
            Subscription: []*gnmipb.Subscription { &gnmipb.Subscription{}},
            Mode: gnmipb.SubscriptionList_POLL,
        }
        slM := gnmipb.SubscriptionList{
            Prefix: &path,
            Subscription: []*gnmipb.Subscription { &gnmipb.Subscription{}},
            Mode: gnmipb.SubscriptionList_STREAM,
        }
        //slNil := gnmipb.SubscriptionList{}

        lst := map[string] struct {
                    err error
                    sl  *gnmipb.SubscriptionList } {
            "stream EOF received before init": { io.EOF, nil },
            "received error from client": { errors.New("mock"), nil },
            "first message must be SubscriptionList": { nil, nil },
            "Invalid subscription path": { nil, &sl },
            "Unkown subscription mode": { nil, &slS },
            "Unknown target": { nil, &slM },
        }

        for s, e := range lst {
            mockTmp := gomonkey.ApplyMethod(reflect.TypeOf(&gnmiSubsServer{}), "Recv",
                    func(stream gnmipb.GNMI_SubscribeServer) (*gnmipb.SubscribeRequest, error) {
                        return &sr, e.err
                })
            defer mockTmp.Reset()

            mockSr := gomonkey.ApplyMethod(reflect.TypeOf(&gnmipb.SubscribeRequest{}), "GetSubscribe",
                    func() *gnmipb.SubscriptionList {
                        return e.sl
                })
                defer mockSr.Reset()

            if ret := c.Run(j); ((ret == nil) ||
                    !strings.Contains(fmt.Sprint(ret), s)) {
                t.Fatalf("Failed to fail Client.Run ret(%v) expect e(%v) s(%s)", ret, e, s)
            }
        }
    }


}
