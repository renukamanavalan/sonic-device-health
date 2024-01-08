package gnmi

import (
    "bytes"
    "errors"
    "fmt"
    "github.com/golang/protobuf/proto"
    gnmipb "github.com/openconfig/gnmi/proto/gnmi"
    gnmi_extpb "github.com/openconfig/gnmi/proto/gnmi_ext"
    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/peer"
    "google.golang.org/grpc/reflection"
    "google.golang.org/grpc/status"
    ldc "lom/src/gnmi/lom_data_clients"
    lpb "lom/src/gnmi/proto"
    "net"
    "strings"
    "sync"
)

var (
    supportedEncodings = []gnmipb.Encoding{gnmipb.Encoding_JSON, gnmipb.Encoding_JSON_IETF, gnmipb.Encoding_PROTO}
)

// Server manages a single gNMI Server implementation. Each client that connects
// via Subscribe or Get will receive a stream of updates based on the requested
// path. Set request is processed by server too.
type Server struct {
    s       *grpc.Server
    lis     net.Listener
    config  *Config
    cMu     sync.Mutex
    clients map[string]*Client
}

// Config is a collection of values for Server
type Config struct {
    // Port for the Server to listen on. If 0 or unset the Server will pick a port
    // for this Server.
    Port             int64
    Threshold        int
    IdleConnDuration int
}

var maMu sync.Mutex

// New returns an initialized Server.
func NewServer(config *Config, opts []grpc.ServerOption) (*Server, error) {
    if config == nil {
        return nil, errors.New("config not provided")
    }

    s := grpc.NewServer(opts...)
    reflection.Register(s)

    srv := &Server{
        s:       s,
        config:  config,
        clients: map[string]*Client{},
    }
    var err error
    if srv.config.Port < 0 {
        srv.config.Port = 0
    }
    srv.lis, err = net.Listen("tcp", fmt.Sprintf(":%d", srv.config.Port))
    if err != nil {
        return nil, fmt.Errorf("failed to open listener port %d: %v", srv.config.Port, err)
    }
    gnmipb.RegisterGNMIServer(srv.s, srv)
    log.V(1).Infof("Created Server on %s, read-only: %t", srv.Address(), !srv.config.EnableNativeWrite)
    return srv, nil
}

// Serve will start the Server serving and block until closed.
func (srv *Server) Serve() error {
    s := srv.s
    if s == nil {
        return fmt.Errorf("Serve() failed: not initialized")
    }
    return srv.s.Serve(srv.lis)
}

// Address returns the port the Server is listening to.
func (srv *Server) Address() string {
    addr := srv.lis.Addr().String()
    return strings.Replace(addr, "[::]", "localhost", 1)
}

// Port returns the port the Server is listening to.
func (srv *Server) Port() int64 {
    return srv.config.Port
}

// Subscribe implements the gNMI Subscribe RPC.
func (s *Server) Subscribe(stream gnmipb.GNMI_SubscribeServer) error {
    ctx := stream.Context()

    pr, ok := peer.FromContext(ctx)
    if !ok {
        return grpc.Errorf(codes.InvalidArgument, "failed to get peer from ctx")
        //return fmt.Errorf("failed to get peer from ctx")
    }
    if pr.Addr == net.Addr(nil) {
        return grpc.Errorf(codes.InvalidArgument, "failed to get peer address")
    }

    c := NewClient(pr.Addr)

    c.setConnectionManager(s.config.Threshold)

    s.cMu.Lock()
    if oc, ok := s.clients[c.String()]; ok {
        log.V(2).Infof("Delete duplicate client %s", oc)
        oc.Close()
        delete(s.clients, c.String())
    }
    s.clients[c.String()] = c
    s.cMu.Unlock()

    err = c.Run(stream)
    s.cMu.Lock()
    delete(s.clients, c.String())
    s.cMu.Unlock()

    log.Flush()
    return err
}

// checkEncodingAndModel checks whether encoding and models are supported by the server. Return error if anything is unsupported.
func (s *Server) checkEncodingAndModel(encoding gnmipb.Encoding, models []*gnmipb.ModelData) error {
    hasSupportedEncoding := false
    for _, supportedEncoding := range supportedEncodings {
        if encoding == supportedEncoding {
            hasSupportedEncoding = true
            break
        }
    }
    if !hasSupportedEncoding {
        return fmt.Errorf("unsupported encoding: %s", gnmipb.Encoding_name[int32(encoding)])
    }

    return nil
}

// Get implements the Get RPC in gNMI spec.
func (s *Server) Get(ctx context.Context, req *gnmipb.GetRequest) (*gnmipb.GetResponse, error) {
    if req.GetType() != gnmipb.GetRequest_ALL {
        return nil, status.Errorf(codes.Unimplemented, "unsupported request type: %s", gnmipb.GetRequest_DataType_name[int32(req.GetType())])
    }

    if err = s.checkEncodingAndModel(req.GetEncoding(), req.GetUseModels()); err != nil {
        return nil, status.Error(codes.Unimplemented, err.Error())
    }

    target := ""
    prefix := req.GetPrefix()
    if prefix != nil {
        target = prefix.GetTarget()
    }

    paths := req.GetPath()
    log.V(2).Infof("GetRequest paths: %v", paths)

    return nil, status.Errorf(codes.Unimplemented, "target(%s) != COUNTERS - the only supported", target)

    /*
        TODO: When we do GET
        var dc ldc.Client
        defer dc.Close()
        notifications := make([]*gnmipb.Notification, len(paths))
        spbValues, err := dc.Get(nil)
        if err != nil {
            return nil, status.Error(codes.NotFound, err.Error())
        }

        for index, spbValue := range spbValues {
            update := &gnmipb.Update{
                Path: spbValue.GetPath(),
                Val:  spbValue.GetVal(),
            }

            notifications[index] = &gnmipb.Notification{
                Timestamp: spbValue.GetTimestamp(),
                Prefix:    prefix,
                Update:    []*gnmipb.Update{update},
            }
        }
        return &gnmipb.GetResponse{Notification: notifications}, nil
    */
}

func (s *Server) Set(ctx context.Context, req *gnmipb.SetRequest) (*gnmipb.SetResponse, error) {
    return nil, grpc.Errorf(codes.Unimplemented, "Capabilities() is not implemented")
    // TODO: Redbutton Set to be implemented.
}

func (s *Server) Capabilities(ctx context.Context, req *gnmipb.CapabilityRequest) (*gnmipb.CapabilityResponse, error) {
    return nil, grpc.Errorf(codes.Unimplemented, "Capabilities() is not implemented")
}
