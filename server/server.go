// Package server DigitalRebar Provision Server
//
// An RestFUL API-driven Provisioner and DHCP server
//
// Terms Of Service:
//
// There are no TOS at this moment, use at your own risk we take no responsibility
//
//     Schemes: https
//     BasePath: /api/v3
//     Version: 0.1.0
//     License: APL https://raw.githubusercontent.com/digitalrebar/digitalrebar/master/LICENSE.md
//     Contact: Greg Althaus<greg@rackn.com> http://rackn.com
//
//     Security:
//       - basicAuth: []
//       - Bearer: []
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
// swagger:meta
package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/provision"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/frontend"
	"github.com/digitalrebar/provision/midlayer"
)

var EmbeddedAssetsExtractFunc func(string, string) error

type ProgOpts struct {
	VersionFlag         bool   `long:"version" description:"Print Version and exit"`
	DisableTftpServer   bool   `long:"disable-tftp" description:"Disable TFTP server"`
	DisableProvisioner  bool   `long:"disable-provisioner" description:"Disable provisioner"`
	DisableDHCP         bool   `long:"disable-dhcp" description:"Disable DHCP server"`
	DisableBINL         bool   `long:"disable-pxe" description:"Disable PXE/BINL server"`
	StaticPort          int    `long:"static-port" description:"Port the static HTTP file server should listen on" default:"8091"`
	TftpPort            int    `long:"tftp-port" description:"Port for the TFTP server to listen on" default:"69"`
	ApiPort             int    `long:"api-port" description:"Port for the API server to listen on" default:"8092"`
	DhcpPort            int    `long:"dhcp-port" description:"Port for the DHCP server to listen on" default:"67"`
	BinlPort            int    `long:"binl-port" description:"Port for the PXE/BINL server to listen on" default:"4011"`
	UnknownTokenTimeout int    `long:"unknown-token-timeout" description:"The default timeout in seconds for the machine create authorization token" default:"600"`
	KnownTokenTimeout   int    `long:"known-token-timeout" description:"The default timeout in seconds for the machine update authorization token" default:"3600"`
	OurAddress          string `long:"static-ip" description:"IP address to advertise for the static HTTP file server" default:""`
	ForceStatic         bool   `long:"force-static" description:"Force the system to always use the static IP."`

	BackEndType    string `long:"backend" description:"Storage to use for persistent data. Can be either 'consul', 'directory', or a store URI" default:"directory"`
	LocalContent   string `long:"local-content" description:"Storage to use for local overrides." default:"directory:///etc/dr-provision?codec=yaml"`
	DefaultContent string `long:"default-content" description:"Store URL for local content" default:"file:///usr/share/dr-provision/default.yaml?codec=yaml"`

	BaseRoot        string `long:"base-root" description:"Base directory for other root dirs." default:"/var/lib/dr-provision"`
	DataRoot        string `long:"data-root" description:"Location we should store runtime information in" default:"digitalrebar"`
	PluginRoot      string `long:"plugin-root" description:"Directory for plugins" default:"plugins"`
	PluginCommRoot  string `long:"plugin-comm-root" description:"Directory for the communications for plugins" default:"/var/run"`
	LogRoot         string `long:"log-root" description:"Directory for job logs" default:"job-logs"`
	SaasContentRoot string `long:"saas-content-root" description:"Directory for additional content" default:"saas-content"`
	FileRoot        string `long:"file-root" description:"Root of filesystem we should manage" default:"tftpboot"`
	ReplaceRoot     string `long:"replace-root" description:"Root of filesystem we should use to replace embedded assets" default:"replace"`

	LocalUI        string `long:"local-ui" description:"Root of Local UI Pages" default:"ux"`
	UIUrl          string `long:"ui-url" description:"URL to redirect to UI" default:"https://portal.rackn.io"`
	DhcpInterfaces string `long:"dhcp-ifs" description:"Comma-seperated list of interfaces to listen for DHCP packets" default:""`
	DefaultStage   string `long:"default-stage" description:"The default stage for the nodes" default:"none"`
	DefaultBootEnv string `long:"default-boot-env" description:"The default bootenv for the nodes" default:"local"`
	UnknownBootEnv string `long:"unknown-boot-env" description:"The unknown bootenv for the system.  Should be \"ignore\" or \"discovery\"" default:"ignore"`

	DebugBootEnv  string `long:"debug-bootenv" description:"Debug level for the BootEnv System" default:"warn"`
	DebugDhcp     string `long:"debug-dhcp" description:"Debug level for the DHCP Server" default:"warn"`
	DebugRenderer string `long:"debug-renderer" description:"Debug level for the Template Renderer" default:"warn"`
	DebugFrontend string `long:"debug-frontend" description:"Debug level for the Frontend" default:"warn"`
	DebugPlugins  string `long:"debug-plugins" description:"Debug level for the Plug-in layer" default:"warn"`
	TlsKeyFile    string `long:"tls-key" description:"The TLS Key File" default:"server.key"`
	TlsCertFile   string `long:"tls-cert" description:"The TLS Cert File" default:"server.crt"`
	UseOldCiphers bool   `long:"use-old-ciphers" description:"Use Original Less Secure Cipher List"`
	DrpId         string `long:"drp-id" description:"The id of this Digital Rebar Provision instance" default:""`
	CurveOrBits   string `long:"cert-type" description:"Type of cert to generate. values are: P224, P256, P384, P521, RSA, or <number of RSA bits>" default:"P384"`

	BaseTokenSecret     string `long:"base-token-secret" description:"Auth Token secret to allow revocation of all tokens" default:""`
	SystemGrantorSecret string `long:"system-grantor-secret" description:"Auth Token secret to allow revocation of all Machine tokens" default:""`
	FakePinger          bool   `hidden:"true" long:"fake-pinger"`
	DefaultLogLevel     string `long:"log-level" description:"Level to log messages at" default:"warn"`
}

func mkdir(d string) error {
	return os.MkdirAll(d, 0755)
}

func Server(c_opts *ProgOpts) {
	localLogger := log.New(os.Stderr, "dr-provision", log.LstdFlags|log.Lmicroseconds|log.LUTC)
	localLogger.Fatalf(server(localLogger, c_opts))
}

func server(localLogger *log.Logger, c_opts *ProgOpts) string {
	var err error

	if c_opts.VersionFlag {
		return fmt.Sprintf("Version: %s", provision.RS_VERSION)
	}
	localLogger.Printf("Version: %s\n", provision.RS_VERSION)

	// Make base root dir
	if err = mkdir(c_opts.BaseRoot); err != nil {
		return fmt.Sprintf("Error creating required directory %s: %v", c_opts.BaseRoot, err)
	}

	// Make other dirs as needed - adjust the dirs as well.
	if strings.IndexRune(c_opts.FileRoot, filepath.Separator) != 0 {
		c_opts.FileRoot = filepath.Join(c_opts.BaseRoot, c_opts.FileRoot)
	}
	if strings.IndexRune(c_opts.PluginRoot, filepath.Separator) != 0 {
		c_opts.PluginRoot = filepath.Join(c_opts.BaseRoot, c_opts.PluginRoot)
	}
	if strings.IndexRune(c_opts.PluginCommRoot, filepath.Separator) != 0 {
		c_opts.PluginCommRoot = filepath.Join(c_opts.BaseRoot, c_opts.PluginCommRoot)
	}
	if len(c_opts.PluginCommRoot) > 70 {
		return fmt.Sprintf("PluginCommRoot Must be less than 70 characters")
	}
	if strings.IndexRune(c_opts.DataRoot, filepath.Separator) != 0 {
		c_opts.DataRoot = filepath.Join(c_opts.BaseRoot, c_opts.DataRoot)
	}
	if strings.IndexRune(c_opts.LogRoot, filepath.Separator) != 0 {
		c_opts.LogRoot = filepath.Join(c_opts.BaseRoot, c_opts.LogRoot)
	}
	if strings.IndexRune(c_opts.SaasContentRoot, filepath.Separator) != 0 {
		c_opts.SaasContentRoot = filepath.Join(c_opts.BaseRoot, c_opts.SaasContentRoot)
	}
	if strings.IndexRune(c_opts.ReplaceRoot, filepath.Separator) != 0 {
		c_opts.ReplaceRoot = filepath.Join(c_opts.BaseRoot, c_opts.ReplaceRoot)
	}
	if strings.IndexRune(c_opts.LocalUI, filepath.Separator) != 0 {
		c_opts.LocalUI = filepath.Join(c_opts.BaseRoot, c_opts.LocalUI)
	}
	if err = mkdir(path.Join(c_opts.FileRoot, "isos")); err != nil {
		return fmt.Sprintf("Error creating required directory %s: %v", c_opts.FileRoot, err)
	}
	if err = mkdir(path.Join(c_opts.FileRoot, "files")); err != nil {
		return fmt.Sprintf("Error creating required directory %s: %v", c_opts.FileRoot, err)
	}
	if err = mkdir(c_opts.ReplaceRoot); err != nil {
		return fmt.Sprintf("Error creating required directory %s: %v", c_opts.ReplaceRoot, err)
	}
	if err = mkdir(c_opts.PluginRoot); err != nil {
		return fmt.Sprintf("Error creating required directory %s: %v", c_opts.PluginRoot, err)
	}
	if err = mkdir(c_opts.PluginCommRoot); err != nil {
		return fmt.Sprintf("Error creating required directory %s: %v", c_opts.PluginCommRoot, err)
	}
	if err = mkdir(c_opts.DataRoot); err != nil {
		return fmt.Sprintf("Error creating required directory %s: %v", c_opts.DataRoot, err)
	}
	if err = mkdir(c_opts.LogRoot); err != nil {
		return fmt.Sprintf("Error creating required directory %s: %v", c_opts.LogRoot, err)
	}
	if err = mkdir(c_opts.LocalUI); err != nil {
		return fmt.Sprintf("Error creating required directory %s: %v", c_opts.LocalUI, err)
	}
	if err = mkdir(c_opts.SaasContentRoot); err != nil {
		return fmt.Sprintf("Error creating required directory %s: %v", c_opts.SaasContentRoot, err)
	}
	localLogger.Printf("Extracting Default Assets\n")
	if EmbeddedAssetsExtractFunc != nil {
		localLogger.Printf("Extracting Default Assets\n")
		if err := EmbeddedAssetsExtractFunc(c_opts.ReplaceRoot, c_opts.FileRoot); err != nil {
			return fmt.Sprintf("Unable to extract assets: %v", err)
		}
	}

	// Make data store
	dtStore, err := midlayer.DefaultDataStack(c_opts.DataRoot, c_opts.BackEndType,
		c_opts.LocalContent, c_opts.DefaultContent, c_opts.SaasContentRoot, c_opts.FileRoot)
	if err != nil {
		return fmt.Sprintf("Unable to create DataStack: %v", err)
	}
	logLevel, err := logger.ParseLevel(c_opts.DefaultLogLevel)
	if err != nil {
		localLogger.Printf("Invalid log level %s", c_opts.DefaultLogLevel)
		return fmt.Sprintf("Try one of `trace`,`debug`,`info`,`warn`,`error`,`fatal`,`panic`")
	}

	// We have a backend, now get default assets
	buf := logger.New(localLogger).SetDefaultLevel(logLevel)
	services := make([]midlayer.Service, 0, 0)
	publishers := backend.NewPublishers(localLogger)

	dt := backend.NewDataTracker(dtStore,
		c_opts.FileRoot,
		c_opts.LogRoot,
		c_opts.OurAddress,
		c_opts.ForceStatic,
		c_opts.StaticPort,
		c_opts.ApiPort,
		buf.Log("backend"),
		map[string]string{
			"debugBootEnv":        c_opts.DebugBootEnv,
			"debugDhcp":           c_opts.DebugDhcp,
			"debugRenderer":       c_opts.DebugRenderer,
			"debugFrontend":       c_opts.DebugFrontend,
			"debugPlugins":        c_opts.DebugPlugins,
			"defaultStage":        c_opts.DefaultStage,
			"logLevel":            c_opts.DefaultLogLevel,
			"defaultBootEnv":      c_opts.DefaultBootEnv,
			"unknownBootEnv":      c_opts.UnknownBootEnv,
			"knownTokenTimeout":   fmt.Sprintf("%d", c_opts.KnownTokenTimeout),
			"unknownTokenTimeout": fmt.Sprintf("%d", c_opts.UnknownTokenTimeout),
			"baseTokenSecret":     c_opts.BaseTokenSecret,
			"systemGrantorSecret": c_opts.SystemGrantorSecret,
		},
		publishers)

	// No DrpId - get a mac address
	if c_opts.DrpId == "" {
		intfs, err := net.Interfaces()
		if err != nil {
			return fmt.Sprintf("Error getting interfaces for DrpId: %v", err)
		}

		for _, intf := range intfs {
			if (intf.Flags & net.FlagLoopback) == net.FlagLoopback {
				continue
			}
			if (intf.Flags & net.FlagUp) != net.FlagUp {
				continue
			}
			if strings.HasPrefix(intf.Name, "veth") {
				continue
			}
			c_opts.DrpId = intf.HardwareAddr.String()
			break
		}
	}

	pc, err := midlayer.InitPluginController(c_opts.PluginRoot, c_opts.PluginCommRoot, dt, publishers)
	if err != nil {
		return fmt.Sprintf("Error starting plugin service: %v", err)
	} else {
		services = append(services, pc)
	}

	fe := frontend.NewFrontend(dt, buf.Log("frontend"),
		c_opts.OurAddress,
		c_opts.ApiPort, c_opts.StaticPort, c_opts.DhcpPort, c_opts.BinlPort,
		c_opts.FileRoot,
		c_opts.LocalUI, c_opts.UIUrl, nil, publishers, c_opts.DrpId, pc,
		c_opts.DisableDHCP, c_opts.DisableTftpServer, c_opts.DisableProvisioner, c_opts.DisableBINL,
		c_opts.SaasContentRoot)
	fe.TftpPort = c_opts.TftpPort
	fe.BinlPort = c_opts.BinlPort
	fe.NoBinl = c_opts.DisableBINL
	backend.SetLogPublisher(buf, publishers)

	// Start the controller now that we have a frontend to front.
	pc.StartRouter(fe.ApiGroup)

	if _, err := os.Stat(c_opts.TlsCertFile); os.IsNotExist(err) {
		if err = buildKeys(c_opts.CurveOrBits, c_opts.TlsCertFile, c_opts.TlsKeyFile); err != nil {
			return fmt.Sprintf("Error building certs: %v", err)
		}
	}

	if !c_opts.DisableTftpServer {
		localLogger.Printf("Starting TFTP server")
		if svc, err := midlayer.ServeTftp(fmt.Sprintf(":%d", c_opts.TftpPort), dt.FS.TftpResponder(), buf.Log("static"), publishers); err != nil {
			return fmt.Sprintf("Error starting TFTP server: %v", err)
		} else {
			services = append(services, svc)
		}
	}

	if !c_opts.DisableProvisioner {
		localLogger.Printf("Starting static file server")
		if svc, err := midlayer.ServeStatic(fmt.Sprintf(":%d", c_opts.StaticPort), dt.FS, buf.Log("static"), publishers); err != nil {
			return fmt.Sprintf("Error starting static file server: %v", err)
		} else {
			services = append(services, svc)
		}
	}

	if !c_opts.DisableDHCP {
		localLogger.Printf("Starting DHCP server")
		if svc, err := midlayer.StartDhcpHandler(dt, buf.Log("dhcp"), c_opts.DhcpInterfaces, c_opts.DhcpPort, publishers, false, c_opts.FakePinger); err != nil {
			return fmt.Sprintf("Error starting DHCP server: %v", err)
		} else {
			services = append(services, svc)
		}

		if !c_opts.DisableBINL {
			localLogger.Printf("Starting PXE/BINL server")
			if svc, err := midlayer.StartDhcpHandler(dt, buf.Log("dhcp"), c_opts.DhcpInterfaces, c_opts.BinlPort, publishers, true, c_opts.FakePinger); err != nil {
				return fmt.Sprintf("Error starting PXE/BINL server: %v", err)
			} else {
				services = append(services, svc)
			}
		}
	}

	var cfg *tls.Config
	if !c_opts.UseOldCiphers {
		cfg = &tls.Config{
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			},
		}
	}
	srv := &http.Server{
		TLSConfig: cfg,
		Addr:      fmt.Sprintf(":%d", c_opts.ApiPort),
		Handler:   fe.MgmtApi,
		ConnState: func(n net.Conn, cs http.ConnState) {
			if cs == http.StateActive {
				l := fe.Logger.Fork()
				laddr, lok := n.LocalAddr().(*net.TCPAddr)
				raddr, rok := n.RemoteAddr().(*net.TCPAddr)
				if lok && rok && cs == http.StateActive {
					backend.AddToCache(l, laddr.IP, raddr.IP)
				}
			}
		},
	}
	services = append(services, srv)

	// Handle SIGHUP, SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)

	go func() {
		// Wait for Api to come up
		for count := 0; count < 5; count++ {
			if count > 0 {
				log.Printf("Waiting for API (%d) to come up...\n", count)
			}
			timeout := time.Duration(5 * time.Second)
			tr := &http.Transport{
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
				TLSHandshakeTimeout:   5 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			}
			client := &http.Client{Transport: tr, Timeout: timeout}
			if _, err := client.Get(fmt.Sprintf("https://127.0.0.1:%d/api/v3", c_opts.ApiPort)); err == nil {
				break
			}
		}

		// Start the controller now that we have a frontend to front.
		if err := pc.StartController(); err != nil {
			log.Printf("Error starting plugin service: %v", err)
			ch <- syscall.SIGTERM
		}

		for {
			s := <-ch
			log.Println(s)

			switch s {
			case syscall.SIGABRT:
				localLogger.Printf("Dumping all goroutine stacks")
				pprof.Lookup("goroutine").WriteTo(os.Stderr, 2)
				localLogger.Printf("Dumping stacks of contested mutexes")
				pprof.Lookup("mutex").WriteTo(os.Stderr, 2)
				localLogger.Printf("Exiting")
				os.Exit(1)
			case syscall.SIGHUP:
				localLogger.Println("Reloading data stores...")
				// Make data store - THIS IS BAD if datastore is memory.
				dtStore, err := midlayer.DefaultDataStack(c_opts.DataRoot, c_opts.BackEndType,
					c_opts.LocalContent, c_opts.DefaultContent, c_opts.SaasContentRoot, c_opts.FileRoot)
				if err != nil {
					localLogger.Printf("Unable to create new DataStack on SIGHUP: %v", err)
				} else {
					rt := dt.Request(dt.Logger)
					rt.AllLocked(func(d backend.Stores) {
						dt.ReplaceBackend(rt, dtStore)
					})
					localLogger.Println("Reload Complete")
				}
			case syscall.SIGTERM, syscall.SIGINT:
				// Stop the service gracefully.
				for _, svc := range services {
					localLogger.Println("Shutting down server...")
					if err := svc.Shutdown(context.Background()); err != nil {
						localLogger.Printf("could not shutdown: %v", err)
					}
				}
				break
			}
		}
	}()

	localLogger.Printf("Starting API server")
	if err = srv.ListenAndServeTLS(c_opts.TlsCertFile, c_opts.TlsKeyFile); err != http.ErrServerClosed {
		// Stop the service gracefully.
		for _, svc := range services {
			localLogger.Println("Shutting down server...")
			if err := svc.Shutdown(context.Background()); err != http.ErrServerClosed {
				localLogger.Printf("could not shutdown: %v", err)
			}
		}
		return fmt.Sprintf("Error running API service: %v\n", err)
	}
	return "Exiting"
}
