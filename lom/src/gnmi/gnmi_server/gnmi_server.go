package main

import (
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log/syslog"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	gnmi "lom/src/gnmi/gnmi_server/server"
	testcert "lom/src/gnmi/testdata/tls"

	cmn "lom/src/lib/lomcommon"
)

const APP_NAME_DEAULT = "LOM_GNMI_SERVER"

var (
    userAuth = gnmi.AuthTypes{"password": false, "cert": false, "jwt": false}
    port     = flag.Int("port", -1, "port to listen on")
    // Certificate files.
    caCert            = flag.String("ca_crt", "", "CA certificate for client certificate validation. Optional.")
    serverCert        = flag.String("server_crt", "", "TLS server certificate")
    serverKey         = flag.String("server_key", "", "TLS server private key")
    insecure          = flag.Bool("insecure", false, "Skip providing TLS cert and key, for testing only!")
    noTLS             = flag.Bool("noTLS", false, "disable TLS, for testing only!")
    allowNoClientCert = flag.Bool("allow_no_client_auth", false,
        "When set, telemetry server will request but not require a client certificate.")
    jwtRefInt          = flag.Uint64("jwt_refresh_int", 900, "Seconds before JWT expiry the token can be refreshed.")
    jwtValInt          = flag.Uint64("jwt_valid_int", 3600, "Seconds that JWT token is valid for.")
    gnmi_native_write  = flag.Bool("gnmi_native_write", gnmi.ENABLE_NATIVE_WRITE, "Enable gNMI native write")
    threshold          = flag.Int("threshold", 100, "max number of client connections")
    idle_conn_duration = flag.Int("idle_conn_duration", 5, "Seconds before server closes idle connections")
pathFlag           = flag.String("path", "", "Config files path")
    modeFlag           = flag.String("mode", "", "Mode of operation. Choice: PROD, test")
    syslogLevelFlag    = flag.Int("syslog_level", 6, "Syslog level")
)

func main() {
// setup application prefix for logging
    cmn.SetPrefix("core")

    // setup agentname to logging
    cmn.SetAgentName(APP_NAME_DEAULT)

    flag.Var(userAuth, "client_auth", "Client auth mode(s) - none,cert,password")
    flag.Parse()

    if *modeFlag == "PROD" {
        cmn.SetLoMRunMode(cmn.LoMRunMode_Prod)
        cmn.InitSyslogWriter(*pathFlag)
        cmn.LogInfo("Starting LoMgnmiServer in PROD mode")
    }

    cmn.SetLogLevel(syslog.Priority(cmn.ValidatedVal(strconv.Itoa(*syslogLevelFlag), int(syslog.LOG_DEBUG),
        int(syslog.LOG_ERR), int(syslog.LOG_INFO), "LogLevel")))

    var defUserAuth gnmi.AuthTypes
    if *gnmi_native_write {
        //In read/write mode we want to enable auth by default.
        defUserAuth = gnmi.AuthTypes{"password": true, "cert": false, "jwt": true}
    } else {
        defUserAuth = gnmi.AuthTypes{"jwt": false, "password": false, "cert": false}
    }

    if isFlagPassed("client_auth") {
        cmn.LogInfo("client_auth provided")
    } else {
        cmn.LogInfo("client_auth not provided, using defaults.")
        userAuth = defUserAuth
    }

    switch {
    case *port <= 0:
        cmn.LogError("port must be > 0.")
        return
    }

    switch {
    case *threshold < 0:
        cmn.LogError("threshold must be >= 0.")
        return
    }

    switch {
    case *idle_conn_duration < 0:
        cmn.LogError("idle_conn_duration must be >= 0, 0 meaning inf")
        return
    }

    gnmi.JwtRefreshInt = time.Duration(*jwtRefInt * uint64(time.Second))
    gnmi.JwtValidInt = time.Duration(*jwtValInt * uint64(time.Second))

    cfg := &gnmi.Config{}
    cfg.Port = int64(*port)
    cfg.EnableNativeWrite = bool(*gnmi_native_write)
    cfg.Threshold = int(*threshold)
    cfg.IdleConnDuration = int(*idle_conn_duration)
    var opts []grpc.ServerOption

    if !*noTLS {
        var certificate tls.Certificate
        var err error
        if *insecure {
            certificate, err = testcert.NewCert()
            if err != nil {
                cmn.LogPanic("could not load server key pair: %s", err)
            }
        } else {
            switch {
            case *serverCert == "":
                cmn.LogError("serverCert must be set.")
                return
            case *serverKey == "":
                cmn.LogError("serverKey must be set.")
                return
            }
            certificate, err = tls.LoadX509KeyPair(*serverCert, *serverKey)
            if err != nil {
                currentTime := time.Now().UTC()
                cmn.LogInfo("Server Cert md5 checksum: %x at time %s", md5.Sum([]byte(*serverCert)), currentTime.String())
                cmn.LogInfo("Server Key md5 checksum: %x at time %s", md5.Sum([]byte(*serverKey)), currentTime.String())
                cmn.LogPanic("could not load server key pair: %s", err)
            }
        }

        tlsCfg := &tls.Config{
            ClientAuth:               tls.RequireAndVerifyClientCert,
            Certificates:             []tls.Certificate{certificate},
            MinVersion:               tls.VersionTLS12,
            CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
            PreferServerCipherSuites: true,
            CipherSuites: []uint16{
                tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
                tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
                tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
                tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
                tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
                tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
            },
        }

        if *allowNoClientCert {
            // RequestClientCert will ask client for a certificate but won't
            // require it to proceed. If certificate is provided, it will be
            // verified.
            tlsCfg.ClientAuth = tls.RequestClientCert
        }

        if *caCert != "" {
            ca, err := ioutil.ReadFile(*caCert)
            if err != nil {
                cmn.LogPanic("could not read CA certificate: %s", err)
            }
            certPool := x509.NewCertPool()
            if ok := certPool.AppendCertsFromPEM(ca); !ok {
                cmn.LogPanic("failed to append CA certificate")
            }
            tlsCfg.ClientCAs = certPool
        } else {
            if userAuth.Enabled("cert") {
                userAuth.Unset("cert")
                cmn.LogWarning("client_auth mode cert requires ca_crt option. Disabling cert mode authentication.")
            }
        }

        keep_alive_params := keepalive.ServerParameters{
            MaxConnectionIdle: time.Duration(cfg.IdleConnDuration) * time.Second, // duration in which idle connection will be closed, default is inf
        }

        opts = []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsCfg))}

        if cfg.IdleConnDuration > 0 { // non inf case
            opts = append(opts, grpc.KeepaliveParams(keep_alive_params))
        }

        cfg.UserAuth = userAuth

        gnmi.GenerateJwtSecretKey()
    }

    s, err := gnmi.NewServer(cfg, opts)
    if err != nil {
        cmn.LogError("Failed to create gNMI server: %v", err)
        return
    }

    cmn.LogInfo("Auth Modes: (%v)", userAuth)
    cmn.LogInfo("Starting RPC server on address: %s", s.Address())
    s.Serve() // blocks until close
}

func isFlagPassed(name string) bool {
    found := false
    flag.Visit(func(f *flag.Flag) {
        if f.Name == name {
            found = true
        }
    })
    return found
}

func getflag(name string) string {
    val := ""
    flag.VisitAll(func(f *flag.Flag) {
        if f.Name == name {
            val = f.Value.String()
        }
    })
    return val
}
