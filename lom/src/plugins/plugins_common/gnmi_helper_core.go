package plugins_common

import (
    "context"
    "errors"
    "fmt"
    "io"
    "lom/src/lib/lomcommon"
    "strings"
    "sync"
    "time"

    ext_gnmi "github.com/openconfig/gnmi/proto/gnmi"
    "google.golang.org/grpc"
    ext_grpc "google.golang.org/grpc"
    ext_codes "google.golang.org/grpc/codes"
    ext_insecure "google.golang.org/grpc/credentials/insecure"
    ext_status "google.golang.org/grpc/status"
)

/*
 * This file contains the wrappers to interact with the gNMI server.
 * Wrappers are divided into 2 parts:
    * 1. Low level wrappers to interact with the gNMI server(In this file).
    * 2. High level wrappers build on top of the low level wrappers for easier use(Refer other file).
 *
 * gnmiServerConnector - Low Level Wrappers
 * GNMISession - High Level Wrappers
 *
 * GNMI connection to server supported only as :
    * 1. Insecure
    * 2. Mode : STREAM
    * 3. STREAM Type : SubscriptionMode_TARGET_DEFINED
*/

/************************************************************************************************************
Wrappers to interact with the gNMI server. Low level implementations
*************************************************************************************************************/

// Interface for ext_grpc(google.golang.org/grpc) methods
type igRPCDialerExt interface {
    //Dial(target string, opts ...ext_grpc.DialOption) (iGRPCConnExt, error)
    DialContext(ctx context.Context, target string, opts ...ext_grpc.DialOption) (iGRPCConnExt, error)
}
type gRPCDialer struct{}

/*
    func (d *gRPCDialer) Dial(target string, opts ...ext_grpc.DialOption) (iGRPCConnExt, error) {
        //External Call : func Dial(target string, opts ...DialOption) (*ClientConn, error)
        conn, err := ext_grpc.Dial(target, opts...)
        if err != nil {
            return nil, err
        }
        return &gRPCConnWrapperExt{ClientConn: conn}, nil
    }
*/
func (d *gRPCDialer) DialContext(ctx context.Context, target string, opts ...ext_grpc.DialOption) (iGRPCConnExt, error) {
    //External Call : func DialContext(ctx context.Context, target string, opts ...DialOption) (*ClientConn, error)
    conn, err := ext_grpc.DialContext(ctx, target, opts...)
    if err != nil {
        return nil, err
    }
    return &gRPCConnWrapperExt{ClientConn: conn}, nil
}

// Interface for ext_gnmi.GNMIClient struct methods(github.com/openconfig/gnmi/proto/gnmi)
type iGNMIClientExt interface {
    Capabilities(ctx context.Context, in *ext_gnmi.CapabilityRequest, opts ...ext_grpc.CallOption) (*ext_gnmi.CapabilityResponse, error)
    Get(ctx context.Context, in *ext_gnmi.GetRequest, opts ...ext_grpc.CallOption) (*ext_gnmi.GetResponse, error)
    Subscribe(ctx context.Context, opts ...ext_grpc.CallOption) (ext_gnmi.GNMI_SubscribeClient, error)
    //Set(ctx context.Context, in *ext_gnmi.SetRequest, opts ...ext_grpc.CallOption) (*ext_gnmi.SetResponse, error)
}
type gNMIExtClientWrapper struct {
    ext_gnmi.GNMIClient
}

// Wrapper for ext_gnmi.NewGNMIClient method(proto/gnmi/gnmi.pb.go)
type igNMIClientMethodsExt interface {
    NewGNMIClient(conn iGRPCConnExt) (iGNMIClientExt, error)
}
type gNMIClientMethodsExt struct{}

func (f *gNMIClientMethodsExt) NewGNMIClient(conn iGRPCConnExt) (iGNMIClientExt, error) {
    clientconn, ok := conn.(*gRPCConnWrapperExt)
    if !ok {
        return nil, fmt.Errorf("error: NewGNMIClient: conn is not of type gRPCConnWrapperExt")
    }
    //External Call : func NewGNMIClient(cc grpc.ClientConnInterface) GNMIClient
    return &gNMIExtClientWrapper{ext_gnmi.NewGNMIClient(clientconn.ClientConn)}, nil
}

// Interface for *grpc.ClientConn methods
type iGRPCConnExt interface {
    Close() error
}

// ClientConnWrapper wraps a grpc.ClientConn and implements IClientConn
type gRPCConnWrapperExt struct {
    *ext_grpc.ClientConn
}

func (ccw *gRPCConnWrapperExt) Close() error {
    return ccw.ClientConn.Close()
}

/*
 * gnmiServerConnector is a struct that holds the connection details to gNMI server.
 */
type gnmiServerConnector struct {
    server   string         // server is the address of the GNMI server.
    mu       sync.Mutex     // mu is a mutex that ensures thread safety when accessing the gnmiServerConnector.
    e_conn   iGRPCConnExt   // conn is the gRPC connection object to the GNMI server.
    e_client iGNMIClientExt // client holds the GNMI client object for the GNMI server once the connection is established.
}

var (
    // gnmiServerConnectorInstances is a map that holds the GNMI server connector instances for each server.
    // The key is the server name and the value is the corresponding GNMI server connector instance.
    gnmiServerConnectorInstances = make(map[string]*gnmiServerConnector)

    // gnmiServerConnectorCounts is a map that holds the number of clients connected to each server.
    // The key is the server name and the value is the number of clients connected to that server.
    gnmiServerConnectorCounts = make(map[string]int)

    // gnmiServerConnectorMutex is a mutex that ensures thread safety when accessing the gnmiServerConnectorInstances and gnmiServerConnectorCounts maps.
    gnmiServerConnectorMutex sync.Mutex

    // GNMI_CONN_TIMEOUT is the timeout duration for GNMI connections.
    // If a connection is not established within this time, it is considered failed.
    GNMI_CONN_TIMEOUT = 3 * time.Second // TO-DO : Goutham : Make it configurable later
)

/* grpc Internal struct to implement the credentials.PerRPCCredentials interface */
type usernamePassword struct {
    username string
    password string
}

/* grpc Internal function to implement the credentials.PerRPCCredentials interface */
func (up *usernamePassword) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
    return map[string]string{
        "username": up.username,
        "password": up.password,
    }, nil
}

/* grpc Internal function to implement the credentials.PerRPCCredentials interface */
func (up *usernamePassword) RequireTransportSecurity() bool {
    return false
}

/*
 * getGNMIInstance returns a GNMI client instance for the specified server.
 *
 * Parameters:
 * - server: A string. This is the address of the gNMI server.
 * - username: A string. This is the username for authentication with the gNMI server.
 * - password: A string. This is the password for authentication with the gNMI server.
 *
 * Returns:
 * - A pointer to a gnmiServerConnector. This is the GNMI client instance for the specified server.
 *   The gnmiServerConnector struct is a wrapper around the gnmi.GNMIClient interface that provides a higher level API for interacting with the gNMI server.
 *
 * - An error. This is nil if the function completed successfully and non-nil if an error occurred (e.g., if there's an error while creating the GNMI client instance).
 *
 * Thread safe
 */
func getGNMIInstance(ext_dialer igRPCDialerExt, ext_clientMethod igNMIClientMethodsExt, server, username, password string) (*gnmiServerConnector, error) {
    // Validate inputs
    if ext_dialer == nil {
        return nil, errors.New("ext_dialer cannot be nil")
    }
    if ext_clientMethod == nil {
        return nil, errors.New("ext_clientMethod cannot be nil")
    }
    if server == "" {
        return nil, errors.New("server cannot be empty")
    }
    if username == "" {
        return nil, errors.New("username cannot be empty")
    }
    if password == "" {
        return nil, errors.New("password cannot be empty")
    }

    gnmiServerConnectorMutex.Lock()
    defer gnmiServerConnectorMutex.Unlock()

    // Check if an instance for the server already exists
    if instance, ok := gnmiServerConnectorInstances[server]; ok {
        // If an instance exists, increment the client count and return the existing instance
        gnmiServerConnectorCounts[server]++
        lomcommon.LogInfo("Incremented client count for server %s to %d", server, gnmiServerConnectorCounts[server])
        return instance, nil
    }

    // If no instance exists, create a new GNMI client instance
    lomcommon.LogInfo("No existing instance for server %s. Creating a new one.", server)
    instance, err := newGNMIClient(ext_dialer, ext_clientMethod, server, username, password)
    if err != nil {
        lomcommon.LogInfo("Error creating new GNMI client instance for server %s: %v", server, err)
        return nil, err
    }

    gnmiServerConnectorInstances[server] = instance
    gnmiServerConnectorCounts[server] = 1
    lomcommon.LogInfo("Created new GNMI client instance for server %s", server)

    return instance, nil
}

/*
 * newGNMIClient creates a new GNMI client for the specified server with the specified username and password.
 *
 * Parameters:
 * - server: A string. This is the address of the gNMI server.
 * - username: A string. This is the username for authentication with the gNMI server.
 * - password: A string. This is the password for authentication with the gNMI server.
 *
 * Returns:
 * - A pointer to a gnmiServerConnector. This is the GNMI client instance for the specified server.
 *   The gnmiServerConnector struct is a wrapper around the gnmi.GNMIClient interface that provides a higher level API for interacting with the gNMI server.
 *
 * - An error. This is nil if the function completed successfully and non-nil if an error occurred (e.g., if there's an error while creating the GNMI client
 *   instance).
 */
func newGNMIClient(ext_dialer igRPCDialerExt, ext_clientMethod igNMIClientMethodsExt, server, username, password string) (*gnmiServerConnector, error) {
    // Call newGNMIClientConnection to create a new gRPC connection to the server
    clientconn, err := newGNMIClientConnection(ext_dialer, server, username, password, ext_grpc.WithTransportCredentials(ext_insecure.NewCredentials()))
    if err != nil {
        return nil, fmt.Errorf("failed to create new client connection: %v", err)
    }

    // Create a new GNMI client using the connection
    gnmiClient, err := ext_clientMethod.NewGNMIClient(clientconn)
    if err != nil {
        // If there's an error while creating the client, return the error
        return nil, fmt.Errorf("failed to create new GNMI client: %v", err)
    }

    // Return a gnmiServerConnector struct that wraps the connection and the GNMI client
    return &gnmiServerConnector{
        e_conn:   clientconn,
        e_client: gnmiClient,
        server:   server,
    }, nil
}

/*
* newGNMIClientConnection creates a new gRPC connection to the specified server with the specified username and password.
* The function uses the DialContext method to establish the connection, which allows for context cancellation and timeouts.
* The grpc.WithBlock() dial option is used to make the DialContext function block until the connection is ready.
*
* Parameters:
* - server: A string. This is the address of the gNMI server.
* - username: A string. This is the username for authentication with the gNMI server.
* - password: A string. This is the password for authentication with the gNMI server.
* - opts: A variadic slice of grpc.DialOption. These are additional dial options for the gRPC connection.
*
  - Returns:
  - - A pointer to a grpc.ClientConn. This is the gRPC connection to the server. If the connection cannot be established within the context's timeout,
    the function will return a non-nil error and the connection object may be nil or in a TRANSIENT_FAILURE state.

*
  - - An error. This is nil if the function completed successfully and non-nil if an error occurred (e.g., if there's an error while dialing the server
    or if the context's deadline is exceeded).
*/
func newGNMIClientConnection(ext_dialer igRPCDialerExt, server, username, password string, opts ...ext_grpc.DialOption) (iGRPCConnExt, error) {
    creds := &usernamePassword{
        username: username,
        password: password,
    }

    // Append two dial options to the opts slice: one for insecure transport credentials, one for per-RPC credentials using the usernamePassword struct
    // TO-DO  : Goutham : Add non block option
    opts = append(opts, ext_grpc.WithTransportCredentials(ext_insecure.NewCredentials()), ext_grpc.WithPerRPCCredentials(creds), grpc.WithBlock())

    // Create a context with a timeout of GNMI_CONN_TIMEOUT
    ctx, cancel := context.WithTimeout(context.Background(), GNMI_CONN_TIMEOUT)
    defer cancel()

    // Call grpc.DialContext with the context, server address, and the opts slice to create the gRPC connection
    conn, err := ext_dialer.DialContext(ctx, server, opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to dial: %v", err)
    }

    return conn, nil
}

/*
 * Capabilities fetches the capabilities of the GNMI server.
 *
 * Parameters:
 * - ctx: A context.Context. This is the context for the capabilities request.
 *
 * Returns:
 * - A pointer to a gnmi.CapabilityResponse. This is the response from the server, which includes the server's capabilities.
 *
 * - An error. This is nil if the function completed successfully and non-nil if an error occurred (e.g., if the client is not initialized or if there's an error while getting the capabilities).
 *
 * Thread safe
 */
func (gs *gnmiServerConnector) capabilities(ctx context.Context) (*ext_gnmi.CapabilityResponse, error) {
    // Check if the client is initialized
    if gs.e_client == nil {
        return nil, errors.New("client is not initialized")
    }

    // Call the Capabilities method of the client with a new CapabilityRequest and the specified context
    response, err := gs.e_client.Capabilities(ctx, &ext_gnmi.CapabilityRequest{})
    if err != nil {
        if ext_status.Code(err) == ext_codes.Canceled {
            return nil, fmt.Errorf("context was cancelled: %v", err)
        } else if ext_status.Code(err) == ext_codes.DeadlineExceeded {
            return nil, fmt.Errorf("context deadline exceeded: %v", err)
        } else {
            return nil, fmt.Errorf("failed to get capabilities: %v", err)
        }
    }

    return response, nil
}

func splitPath(path string) []*ext_gnmi.PathElem {
    elems := make([]*ext_gnmi.PathElem, 0)
    for _, elem := range strings.Split(path, "/") {
        if elem != "" {
            elems = append(elems, &ext_gnmi.PathElem{Name: elem})
        }
    }
    return elems
}

/*
 * Get fetches the specified paths from the GNMI server.
 *
 * Parameters:
 * - ctx: A context.Context. This is the context for the get request.
 * - prefix: A string. This is the prefix used to identify the origin of the data.
 * - paths: A slice of strings. These are the paths to fetch from the server.
 *
 * Returns:
 * - A pointer to a gnmi.GetResponse. This is the response from the server, which includes the data for the requested paths.
 *
 * - An error. This is nil if the function completed successfully and non-nil if an error occurred (e.g., if the client is not initialized, no paths are provided, or there's an error while getting the data).
 *
 * Thread safe
 */
func (gs *gnmiServerConnector) get(ctx context.Context, prefix string, paths []string) (*ext_gnmi.GetResponse, error) {
    // Check if the client is initialized
    if gs.e_client == nil {
        return nil, errors.New("client is not initialized")
    }

    // Check if any paths are provided
    if len(paths) == 0 {
        return nil, errors.New("no paths provided")
    }

    // Create a slice of gnmi.Path objects from the paths
    pathList := make([]*ext_gnmi.Path, len(paths))
    for i, path := range paths {
        pathList[i] = &ext_gnmi.Path{Elem: splitPath(path)}
    }

    // Create a GetRequest with the paths and the prefix
    getRequest := &ext_gnmi.GetRequest{
        Path:   pathList,
        Prefix: &ext_gnmi.Path{Elem: splitPath(prefix)},
    }

    // Call the Get method of the client with the GetRequest and the specified context
    response, err := gs.e_client.Get(ctx, getRequest)
    if err != nil {
        if ext_status.Code(err) == ext_codes.Canceled {
            return nil, fmt.Errorf("context was cancelled: %v", err)
        } else if ext_status.Code(err) == ext_codes.DeadlineExceeded {
            return nil, fmt.Errorf("context deadline exceeded: %v", err)
        } else {
            return nil, fmt.Errorf("failed to get: %v", err)
        }
    }

    return response, nil
}

/*
 * subscribe sends a subscribe request to the GNMI server for the specified paths.
 *
 * Parameters:
 * - ctx: A context.Context. This is the context for the subscribe request.
 * - mode: A gnmi.SubscriptionList_Mode. This is the subscription mode (STREAM, ONCE, POLL).
 * - prefix: A string. This is the prefix used to identify the origin of the data.
 * - paths: A slice of strings. These are the paths to subscribe to on the server.
 *
 * Returns:
 * - A gnmi.GNMI_SubscribeClient. This is the subscribe client for the subscription. The subscribe client is used to receive updates from the server.
 *
 * - An error. This is nil if the function completed successfully and non-nil if an error occurred (e.g., if the client is not initialized, no paths are provided, or there's an error while subscribing or sending the subscribe request).
 *
 * Thread safe
 */
func (gs *gnmiServerConnector) subscribe(ctx context.Context, mode ext_gnmi.SubscriptionList_Mode, prefix string, paths []string) (ext_gnmi.GNMI_SubscribeClient, error) {
    lomcommon.LogInfo("subscribe: mode: %v, prefix: %v, paths: %v", mode, prefix, paths)
    // Check if the client is initialized
    if gs.e_client == nil {
        return nil, errors.New("client is not initialized")
    }

    // Check if any paths are provided
    if len(paths) == 0 {
        return nil, errors.New("no paths provided")
    }

    // Create a slice of gnmi.Path objects from the paths
    pathList := make([]*ext_gnmi.Path, len(paths))
    for i, path := range paths {
        pathList[i] = &ext_gnmi.Path{Elem: splitPath(path)}
    }

    // Create a slice of gnmi.Subscription objects from the pathList
    subscriptions := make([]*ext_gnmi.Subscription, len(pathList))
    for i, path := range pathList {
        subscriptions[i] = &ext_gnmi.Subscription{Path: path} // mode is left to default (SubscriptionMode_TARGET_DEFINED)
    }

    // Create a SubscriptionList with the subscriptions, the prefix, and the mode
    subscriptionList := &ext_gnmi.SubscriptionList{
        Prefix:       &ext_gnmi.Path{Target: prefix}, // single target
        Subscription: subscriptions,
        Mode:         mode,
    }

    // Call the Subscribe method of the client with the specified context to create a subscribe client
    subscribeClient, err := gs.e_client.Subscribe(ctx)
    if err != nil {
        if ext_status.Code(err) == ext_codes.Canceled {
            return nil, fmt.Errorf("context was cancelled: %v", err)
        } else if ext_status.Code(err) == ext_codes.DeadlineExceeded {
            return nil, fmt.Errorf("context deadline exceeded: %v", err)
        } else {
            return nil, fmt.Errorf("failed to subscribe: %w", err)
        }
    }

    // Send a SubscribeRequest with the SubscriptionList to the subscribe client
    err = subscribeClient.Send(&ext_gnmi.SubscribeRequest{
        Request: &ext_gnmi.SubscribeRequest_Subscribe{
            Subscribe: subscriptionList,
        },
    })

    // If an error occurs while sending the SubscribeRequest
    if err != nil {
        if subscribeClient != nil {
            subscribeClient.CloseSend() // close the send direction of the stream
        }
        return nil, fmt.Errorf("failed to send subscribe request: %w", err)
    }
    return subscribeClient, nil
}

/*
 * SubscribeStream sends a STREAM subscribe request to the GNMI server for the specified paths.
 *
 * Parameters:
 * - ctx: A context.Context. This is the context for the subscribe request.
 * - prefix: A string. This is the prefix used to identify the origin of the data.
 * - paths: A slice of strings. These are the paths to subscribe to on the server.
 *
 * Returns:
 * - A gnmi.GNMI_SubscribeClient. This is the subscribe client for the subscription. The subscribe client is used to receive updates from the server.
 *
 * - An error. This is nil if the function completed successfully and non-nil if an error occurred (e.g., if there's an error while subscribing).
 *
 * Thread safe
 */
func (gs *gnmiServerConnector) subscribeStream(ctx context.Context, prefix string, paths []string) (ext_gnmi.GNMI_SubscribeClient, error) {
    // Call the subscribe method with the STREAM mode, the specified context, prefix, and paths
    subscribeClient, err := gs.subscribe(ctx, ext_gnmi.SubscriptionList_STREAM, prefix, paths)
    if err != nil {
        return nil, err
    }

    return subscribeClient, nil
}

/*
 * Close closes the connection to the GNMI server.
 *
 * Parameters: None.
 *
 * Returns:
 * - An error. This is nil if the function completed successfully and non-nil if an error occurred (e.g., if the connection is not initialized, there's no client count for the server, or there's an error while closing the connection).
 *
 * Thread safe
 */
func (gs *gnmiServerConnector) close() error {
    gs.mu.Lock()
    defer gs.mu.Unlock()

    // Check if the connection is initialized
    if gs.e_conn == nil {
        return errors.New("connection is not initialized")
    }

    gnmiServerConnectorMutex.Lock()
    defer gnmiServerConnectorMutex.Unlock()

    // Get the client count for the server
    count, ok := gnmiServerConnectorCounts[gs.server]
    if !ok {
        // If there's no client count for the server, return an error
        return fmt.Errorf("no client count for server: %s", gs.server)
    }

    // Decrement the client count
    count--

    // If the client count becomes 0, close the connection and set it to nil
    if count == 0 {
        lomcommon.LogInfo("Client count for server %s is 0, closing connection", gs.server)
        err := gs.e_conn.Close()
        gs.e_conn = nil
        if err != nil {
            return fmt.Errorf("failed to close connection: %w", err)
        }
        // cleanup the client instance
        delete(gnmiServerConnectorInstances, gs.server)
        delete(gnmiServerConnectorCounts, gs.server)
        lomcommon.LogInfo("Closed connection for server %s", gs.server)
    } else {
        // If the client count is not 0, update the client count for the server
        gnmiServerConnectorCounts[gs.server] = count
        lomcommon.LogInfo("Updated client count for server %s to %d", gs.server, count)
    }

    return nil
}

/*
 * Server returns the server address of the gnmiServerConnector.
 *
 * Returns:
 * - A string. This is the address of the gNMI server.
 */
func (gs *gnmiServerConnector) Server() string {
    return gs.server
}

var SubscriptionCancelled = &ext_gnmi.Notification{} // Define a special value for cancelled subscriptions

/*
 * Helper fuction ReceiveSubscriptions starts receiving notifications from the GNMI server for the current subscription.
 *
 * Parameters:
 * - client: A gnmi.GNMI_SubscribeClient. This is the subscribe client for the subscription. The subscribe client is used to receive updates from
 * the server.
 * - gnmiNotificationsCh: A send-only channel of *gnmi.Notification. This is the channel on which to send the received notifications.
 *
 * Returns:
 * - An error. This is nil if the function completed successfully and non-nil if an error occurred (e.g., if there's an error while receiving the
 * subscription).
 *
 * The function returns a send-only channel of *gnmi.Notification. Each *gnmi.Notification represents a single gNMI notification, which is a
 * collection of updates (and optionally deletes) to a set of paths in the data tree. The structure of *gnmi.Notification is as follows:
 *
 * Refer : https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md#3514-the-subscriberesponse-message
 *         https://github.com/openconfig/gnmi/blob/master/proto/gnmi/gnmi.proto
 *
 * message Notification {
 *   int64 timestamp = 1; // The timestamp associated with the data in the Update message.
 *   string prefix = 2; // The prefix used to identify the origin of the data.
 *   repeated Update update = 3; // The data updates.
 *   repeated string delete = 4; // The paths to delete.
 *   bool sync_response = 5; // Indicates that this message is the last in a SubscribeResponse message.
 *   string alias = 6; // The alias used to identify the origin of the data.
 *   Error error = 7; // The error associated with the data in the Update message.
 * }
 *
 * message Update {
 *   string path = 1; // The path to the data.
 *   Value value = 2; // The updated value.
 *   int64 timestamp = 3; // The timestamp associated with the data.
 * }
 *
 * message Value {
 *   oneof value {
 *     string string_val = 1;
 *     int64 int_val = 2;
 *     uint64 uint_val = 3;
 *     bool bool_val = 4;
 *     bytes bytes_val = 5;
 *     float float_val = 6;
 *     double double_val = 7;
 *   }
 * }
 *
 * This function is thread safe and must be called after Subscribe and before Unsubscribe.
 * It must be called only once at a time per session.
 * Blocking call
 */
func receiveSubscriptions(client ext_gnmi.GNMI_SubscribeClient, gnmiNotificationsCh chan<- *ext_gnmi.Notification) error {
    for {
        // Call the Recv method of the client to receive a notification
        response, err := client.Recv()
        if err != nil {
            if err == io.EOF {
                // If the error is EOF return nil
                //close(gnmiNotificationsCh)
                return nil
            }
            // If the subscription was cancelled by the user, return SubscriptionCancelled
            if ext_status.Code(err) == ext_codes.Canceled {
                lomcommon.LogInfo("Subscription was cancelled by the user.")
                gnmiNotificationsCh <- SubscriptionCancelled
                //close(gnmiNotificationsCh)
                return nil
            } else {
                lomcommon.LogInfo("Error received from client.Recv(): %v\n", err)
            }

            // Check if the error is temporary
            if err, ok := err.(interface{ Temporary() bool }); ok && err.Temporary() {
                // If the error is temporary, sleep for a second and continue to the next iteration
                lomcommon.LogInfo("Temporary error: %v. Retrying in 1 second...", err)
                time.Sleep(time.Second)
                continue
            }

            // If the error is not temporary, return a formatted error message
            //close(gnmiNotificationsCh)
            return fmt.Errorf("error receiving subscription: %w", err)
        }

        // Check if the response is nil
        if response == nil {
            continue
        }

        // Check the type of the response
        switch res := response.Response.(type) {
        case *ext_gnmi.SubscribeResponse_SyncResponse:
            //lomcommon.LogInfo("SyncResponse received")
            //close(gnmiNotificationsCh)
            //return nil
        case *ext_gnmi.SubscribeResponse_Update:
            //lomcommon.LogInfo("Received notification")
            gnmiNotificationsCh <- res.Update
        default:
            // If the response type is unexpected, return an error
            //close(gnmiNotificationsCh)
            return errors.New("unexpected response type")
        }
    }
}
