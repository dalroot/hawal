package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/dalroot/hawal/core/v2/carrier"
	"github.com/dalroot/hawal/core/v2/engine"
)

var (
	Version = "2.5.3"
)

type ConfigFile struct {
	Mode          string   `json:"mode"`
	Carrier       string   `json:"carrier"`
	BindAddr      string   `json:"bind_addr"`
	ConnectAddr   string   `json:"connect_addr"`
	Ports         []string `json:"ports"`
	Token         string   `json:"token"`
	NoDelay       bool     `json:"nodelay"`
	InsecureTLS   bool     `json:"insecure_tls"`
	ServerName    string   `json:"server_name"`
	InterfaceName string   `json:"interface"`
	RouterMAC     string   `json:"router_mac"`
}

func main() {
	mode := flag.String("mode", "", "Execution mode: 'server' or 'client'")
	carrierFlag := flag.String("carrier", "tcp", "Transport carrier: 'tcp', 'tls', or 'rawpaq'")
	bindAddr := flag.String("listen", "", "Carrier listen address for server (e.g. 0.0.0.0:3090)")
	connectAddr := flag.String("connect", "", "Server address for client to connect to (e.g. 1.2.3.4:3090)")
	portsFlag := flag.String("ports", "", "Comma-separated port rules (e.g. 443=127.0.0.1:443,2083=127.0.0.1:2083)")
	token := flag.String("token", "", "Shared authentication and encryption secret (Noise PFS)")
	noDelay := flag.Bool("nodelay", true, "Enable TCP_NODELAY for lowest latency")
	insecureTLS := flag.Bool("insecure", true, "Allow unverified certificates in TLS carrier")
	serverName := flag.String("sni", "", "ServerName (SNI) for TLS carrier camouflage")
	ifaceFlag := flag.String("iface", "", "Network interface for raw packet carrier (auto-detected if empty)")
	routerMACFlag := flag.String("router-mac", "", "Router/Gateway MAC address (auto-detected if empty)")
	configPath := flag.String("config", "", "Path to JSON configuration file")
	showVersion := flag.Bool("version", false, "Show Hawal Core version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Hawal Stealth Core (هه‌واڵ) v%s (Native Raw-TCP & Kernel BPF Engine)\n", strings.TrimPrefix(Version, "v"))
		return
	}

	cfg := ConfigFile{
		Mode:          *mode,
		Carrier:       *carrierFlag,
		BindAddr:      *bindAddr,
		ConnectAddr:   *connectAddr,
		Token:         *token,
		NoDelay:       *noDelay,
		InsecureTLS:   *insecureTLS,
		ServerName:    *serverName,
		InterfaceName: *ifaceFlag,
		RouterMAC:     *routerMACFlag,
	}

	if *portsFlag != "" {
		cfg.Ports = strings.Split(*portsFlag, ",")
	}

	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			log.Fatalf("Failed to read config file %q: %v", *configPath, err)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Fatalf("Failed to parse config file %q: %v", *configPath, err)
		}
	}

	if cfg.Token == "" {
		log.Fatalf("Authentication token is required (-token or config file)")
	}

	if cfg.Mode == "" {
		log.Fatalf("Execution mode is required (-mode=server or -mode=client)")
	}

	fmt.Println("⚡ ==========================================================")
	fmt.Printf("🚀 Hawal Stealth Core (هه‌واڵ) v%s\n", Version)
	fmt.Println("🔒 Architecture: Hybrid Multiplexed Carrier Engine")
	fmt.Printf("🎯 Mode: %s | Carrier: %s\n", cfg.Mode, cfg.Carrier)
	if cfg.Mode == "server" {
		fmt.Printf("📡 Listen Address: %s\n", cfg.BindAddr)
	} else {
		fmt.Printf("🔗 Connect Address: %s\n", cfg.ConnectAddr)
	}
	if len(cfg.Ports) > 0 {
		fmt.Printf("🔀 Port Rules: %s\n", strings.Join(cfg.Ports, ", "))
	}
	fmt.Println("==========================================================")

	engineCfg := engine.Config{
		Mode:        cfg.Mode,
		CarrierKind: carrier.Kind(cfg.Carrier),
		BindAddr:    cfg.BindAddr,
		ConnectAddr: cfg.ConnectAddr,
		Ports:       cfg.Ports,
		Token:       cfg.Token,
		NoDelay:     cfg.NoDelay,
		InsecureTLS:   cfg.InsecureTLS,
		ServerName:    cfg.ServerName,
		InterfaceName: cfg.InterfaceName,
		RouterMAC:     cfg.RouterMAC,
	}

	eng, err := engine.NewEngine(engineCfg)
	if err != nil {
		log.Fatalf("Engine initialization failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		if err := eng.Start(ctx); err != nil && ctx.Err() == nil {
			log.Fatalf("Engine terminated with error: %v", err)
		}
	}()

	<-sigChan
	fmt.Println("\n🛑 Shutting down Hawal Core cleanly...")
	cancel()
	_ = eng.Close()
}
