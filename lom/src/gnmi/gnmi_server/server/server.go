package gnmi

import (
    "bytes"
    "errors"
    "fmt"
    gnmipb "github.com/openconfig/gnmi/proto/gnmi"
    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/peer"
    "google.golang.org/grpc/reflection"
    "google.golang.org/grpc/status"
    "net"
    "strings"
    "sync"

    "lom/src/gnmi/utils"
    cmn "lom/src/lib/lomcommon"
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
type AuthTypes map[string]bool

// Config is a collection of values for Server
type Config struct {
    // Port for the Server to listen on. If 0 or unset the Server will pick a port
    // for this Server.
    Port              int64
    Threshold         int
    UserAuth          AuthTypes
    EnableNativeWrite bool
    IdleConnDuration  int
}

var AuthLock sync.Mutex

func (i AuthTypes) String() string {
    if i["none"] {
        return ""
    }
    b := new(bytes.Buffer)
    for key, value := range i {
        if value {
            fmt.Fprintf(b, "%s ", key)
        }
    }
    return b.String()
}

func (i AuthTypes) Any() bool {
    if i["none"] {
        return false
    }
    for _, value := range i {
        if value {
            return true
        }
    }
    return false
}

func (i AuthTypes) Enabled(mode string) bool {
    if i["none"] {
        return false
    }
    if value, exist := i[mode]; exist && value {
        return true
    }
    return false
}

func (i AuthTypes) Set(mode string) error {
    modes := strings.Split(mode, ",")
    for _, m := range modes {
        m = strings.Trim(m, " ")
        if m == "none" || m == "" {
            i["none"] = true
            return nil
        }

        if _, exist := i[m]; !exist {
            return fmt.Errorf("Expecting one or more of 'cert', 'password' or 'jwt'")
        }
        i[m] = true
    }
    return nil
}

func (i AuthTypes) Unset(mode string) error {
    modes := strings.Split(mode, ",")
    for _, m := range modes {
        m = strings.Trim(m, " ")
        if _, exist := i[m]; !exist {
            return fmt.Errorf("Expecting one or more of 'cert', 'password' or 'jwt'")
        }
        i[m] = false
    }
    return nil
}

// New returns an initialized Server.
func NewServer(config *Config, opts []grpc.ServerOption) (*Server, error) {
    if config == nil {
        return nil, errors.New("config not provided")
    }

    lom_utils.InitCounters()

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
    cmn.LogInfo("Created Server on %s, read-only: %t", srv.Address(), !srv.config.EnableNativeWrite)
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

func authenticate(UserAuth AuthTypes, ctx context.Context) (context.Context, error) {
    var err error
    success := false
    rc, ctx := lom_utils.GetContext(ctx)
    if !UserAuth.Any() {
        //No Auth enabled
        rc.Auth.AuthEnabled = false
        return ctx, nil
    }
    rc.Auth.AuthEnabled = true
    if UserAuth.Enabled("password") {
        ctx, err = BasicAuthenAndAuthor(ctx)
        if err == nil {
            success = true
        }
    }
    if !success && UserAuth.Enabled("jwt") {
        _, ctx, err = JwtAuthenAndAuthor(ctx)
        if err == nil {
            success = true
        }
    }
    if !success && UserAuth.Enabled("cert") {
        ctx, err = ClientCertAuthenAndAuthor(ctx)
        if err == nil {
            success = true
        }
    }

    //Allow for future authentication mechanisms here...

    if !success {
        return ctx, status.Error(codes.Unauthenticated, "Unauthenticated")
    }
    cmn.LogInfo("authenticate user %v, roles %v", rc.Auth.User, rc.Auth.Roles)

    return ctx, nil
}

// Subscribe implements the gNMI Subscribe RPC.
func (s *Server) Subscribe(stream gnmipb.GNMI_SubscribeServer) error {
    ctx := stream.Context()
    ctx, err := authenticate(s.config.UserAuth, ctx)
    if err != nil {
        return err
    }

    pr, ok := peer.FromContext(ctx)
    if !ok {
        return grpc.Errorf(codes.InvalidArgument, "failed to get peer from ctx")
        //return fmt.Errorf("failed to get peer from ctx")
    }
    if pr.Addr == net.Addr(nil) {
        return grpc.Errorf(codes.InvalidArgument, "failed to get peer address")
    }

    /* TODO: authorize the user
       msg, ok := credentials.AuthorizeUser(ctx)
       if !ok {
           cmn.LogInfo("denied a Set request: %v", msg)
           return nil, status.Error(codes.PermissionDenied, msg)
       }
    */

    c := NewClient(pr.Addr)

    s.cMu.Lock()
    if oc, ok := s.clients[c.String()]; ok {
        cmn.LogInfo("Delete duplicate client %s", oc)
        oc.Close()
        delete(s.clients, c.String())
    }
    s.clients[c.String()] = c
    s.cMu.Unlock()

    err = c.Run(stream)
    s.cMu.Lock()
    delete(s.clients, c.String())
    s.cMu.Unlock()

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
    return nil, grpc.Errorf(codes.Unimplemented, "Get() is not implemented")
}

func (s *Server) Set(ctx context.Context, req *gnmipb.SetRequest) (*gnmipb.SetResponse, error) {
    return nil, grpc.Errorf(codes.Unimplemented, "Set() is not implemented")
}

func (s *Server) Capabilities(ctx context.Context, req *gnmipb.CapabilityRequest) (*gnmipb.CapabilityResponse, error) {
    return nil, grpc.Errorf(codes.Unimplemented, "Capabilities() is not implemented")
}