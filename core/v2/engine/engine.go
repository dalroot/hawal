package engine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
	rawpaqcarrier "github.com/dalroot/hawal/core/v2/carrier/rawpaq"
	tcpcarrier "github.com/dalroot/hawal/core/v2/carrier/tcp"
	tlscarrier "github.com/dalroot/hawal/core/v2/carrier/tls"
	"github.com/dalroot/hawal/core/v2/record"
	"github.com/dalroot/hawal/core/v2/secure"
)

type Config struct {
	Mode        string       `json:"mode"`         // "server" or "client"
	CarrierKind carrier.Kind `json:"carrier"`      // "tcp", "tls-http", "rawpaq"
	BindAddr    string       `json:"bind_addr"`    // Carrier listen address (e.g. "0.0.0.0:3090")
	ConnectAddr string       `json:"connect_addr"` // Carrier dial address (e.g. "1.2.3.4:3090")
	Ports       []string     `json:"ports"`        // Forwarding rules: ["8080=127.0.0.1:8080"]
	Token       string       `json:"token"`        // Noise pre-shared authentication secret
	NoDelay     bool         `json:"nodelay"`
	InsecureTLS bool         `json:"insecure_tls"`
	ServerName  string       `json:"server_name"`
	InterfaceName string         `json:"interface"`
	RouterMAC     string         `json:"router_mac"`
	SessionOpts   SessionOptions `json:"-"`
}

type Engine struct {
	cfg        Config
	registry   *carrier.Registry
	handshaker secure.Handshaker
	rules      []Rule
	portMap    map[string]string

	mu        sync.RWMutex
	session   *Session
	listeners []*ForwardListener

	cancel context.CancelFunc
}

func NewEngine(cfg Config) (*Engine, error) {
	return NewEngineWithRegistry(cfg, nil)
}

func NewEngineWithRegistry(cfg Config, reg *carrier.Registry) (*Engine, error) {
	if cfg.Token == "" {
		return nil, errors.New("engine: token is required")
	}
	if cfg.Mode != "server" && cfg.Mode != "client" {
		return nil, fmt.Errorf("engine: invalid mode %q (must be 'server' or 'client')", cfg.Mode)
	}

	// Canonicalize carrier names: "tls" -> "tls-http"
	if cfg.CarrierKind == "tls" || cfg.CarrierKind == "" {
		if cfg.CarrierKind == "" {
			cfg.CarrierKind = carrier.KindTCP
		} else {
			cfg.CarrierKind = carrier.KindTLSHTTP
		}
	}

	if reg == nil {
		reg = carrier.NewRegistry()
		if err := reg.Register(carrier.KindTCP, func() (carrier.Carrier, error) {
			return tcpcarrier.Carrier{}, nil
		}); err != nil {
			return nil, err
		}

		if err := reg.Register(carrier.KindTLSHTTP, func() (carrier.Carrier, error) {
			return tlscarrier.New(tlscarrier.Config{
				Insecure:   cfg.InsecureTLS,
				ServerName: cfg.ServerName,
			}), nil
		}); err != nil {
			return nil, err
		}

		if err := reg.Register(carrier.KindRawPaq, func() (carrier.Carrier, error) {
			rawCfg := rawpaqcarrier.DefaultConfig()
			if cfg.InterfaceName != "" {
				rawCfg.InterfaceName = cfg.InterfaceName
			}
			if cfg.RouterMAC != "" {
				rawCfg.RouterMAC = cfg.RouterMAC
			}
			return rawpaqcarrier.New(rawCfg, rawpaqcarrier.DefaultBackend(), nil)
		}); err != nil {
			return nil, err
		}
	}

	rules, portMap := ParseRules(cfg.Ports)

	return &Engine{
		cfg:        cfg,
		registry:   reg,
		handshaker: secure.NewNoiseHandshaker(cfg.Token),
		rules:      rules,
		portMap:    portMap,
	}, nil
}

func (e *Engine) ActiveSession() *Session {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.session
}

func (e *Engine) getReadySession() *Session {
	return e.WaitForSession(context.Background(), 3*time.Second)
}

// WaitForSession waits for an active, non-closed session up to the given timeout.
func (e *Engine) WaitForSession(ctx context.Context, timeout time.Duration) *Session {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()

	for {
		e.mu.RLock()
		sess := e.session
		e.mu.RUnlock()

		if sess != nil {
			select {
			case <-sess.closed:
				// session is closed, keep waiting
			default:
				return sess
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			return nil
		case <-ticker.C:
		}
	}
}

func (e *Engine) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	defer cancel()

	car, err := e.registry.Build(e.cfg.CarrierKind)
	if err != nil {
		return fmt.Errorf("engine: build carrier %q: %w", e.cfg.CarrierKind, err)
	}

	switch e.cfg.Mode {
	case "server":
		return e.runServer(ctx, car)
	case "client":
		return e.runClient(ctx, car)
	default:
		return fmt.Errorf("engine: unknown mode %q", e.cfg.Mode)
	}
}

func (e *Engine) Close() error {
	e.mu.Lock()
	if e.cancel != nil {
		e.cancel()
	}
	for _, l := range e.listeners {
		_ = l.Close()
	}
	e.listeners = nil
	if e.session != nil {
		_ = e.session.Close()
		e.session = nil
	}
	e.mu.Unlock()
	return nil
}

func (e *Engine) runServer(ctx context.Context, car carrier.Carrier) error {
	if e.cfg.BindAddr == "" {
		return errors.New("engine: server requires bind_addr")
	}

	acceptor, err := car.Listen(ctx, carrier.Bind{Network: "tcp", Address: e.cfg.BindAddr}, carrier.Options{
		NoDelay:   e.cfg.NoDelay,
		KeepAlive: 30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("engine: carrier listen failed: %w", err)
	}
	defer acceptor.Close()

	log.Printf("[Hawal-v2] 🚀 Server carrier [%s] listening on %s", e.cfg.CarrierKind, e.cfg.BindAddr)

	// Start user-facing forward listeners on configured ports
	e.mu.Lock()
	for _, rule := range e.rules {
		fl, err := StartForwardListener(rule, e.cfg.NoDelay, e.getReadySession)
		if err != nil {
			log.Printf("[Hawal-v2] ⚠️ Failed to bind forward port %s: %v", rule.ListenPort, err)
			continue
		}
		e.listeners = append(e.listeners, fl)
		log.Printf("[Hawal-v2] 🟢 Forwarding ingress port %s -> %s", rule.ListenPort, rule.TargetAddr)
	}
	e.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		link, err := acceptor.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("[Hawal-v2] Accept error: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		go e.handleServerLink(ctx, link)
	}
}

func (e *Engine) handleServerLink(ctx context.Context, link carrier.Link) {
	log.Printf("[Hawal-v2] 🔌 Incoming link from %s (ID: %s)", link.RemoteAddr(), link.ID())

	res, err := e.handshaker.Handshake(ctx, secure.RoleResponder, link, "server", secure.Limits{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		log.Printf("[Hawal-v2] ⚠️ Handshake failed from %s: %v (Dumping connection)", link.RemoteAddr(), err)
		_ = link.Close()
		return
	}

	opts := record.DefaultOptions()
	opts.MaskLength = true

	codec, err := record.New(
		res.Secrets.SendAEAD,
		res.Secrets.ReceiveAEAD,
		res.Secrets.SendNonce,
		res.Secrets.ReceiveNonce,
		res.Secrets.Transcript,
		opts,
	)
	if err != nil {
		log.Printf("[Hawal-v2] Failed to initialize codec: %v", err)
		_ = link.Close()
		return
	}

	sess, err := NewSessionWithOptions(link, codec, true, e.cfg.SessionOpts)
	if err != nil {
		log.Printf("[Hawal-v2] Failed to create session: %v", err)
		_ = link.Close()
		return
	}

	e.mu.Lock()
	if e.session != nil {
		log.Printf("[Hawal-v2] Replacing prior active session")
		_ = e.session.Close()
	}
	e.session = sess
	e.mu.Unlock()

	log.Printf("[Hawal-v2] ⚡ Authenticated v2 session established with %s (PFS: X25519, AEAD: ChaCha20-Poly1305)", link.RemoteAddr())

	// Also serve egress for any streams opened by the peer
	if err := ServeEgress(ctx, sess, e.portMap, e.cfg.NoDelay); err != nil && ctx.Err() == nil {
		log.Printf("[Hawal-v2] Session egress ended: %v", err)
	}
}

func (e *Engine) runClient(ctx context.Context, car carrier.Carrier) error {
	if e.cfg.ConnectAddr == "" {
		return errors.New("engine: client requires connect_addr")
	}

	// If client is acting as ingress, start listeners
	e.mu.Lock()
	if len(e.listeners) == 0 && len(e.rules) > 0 {
		for _, rule := range e.rules {
			fl, err := StartForwardListener(rule, e.cfg.NoDelay, e.getReadySession)
			if err != nil {
				log.Printf("[Hawal-v2] ⚠️ Failed to bind forward port %s: %v", rule.ListenPort, err)
				continue
			}
			e.listeners = append(e.listeners, fl)
			log.Printf("[Hawal-v2] 🟢 Client forwarding ingress port %s -> %s", rule.ListenPort, rule.TargetAddr)
		}
	}
	e.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		log.Printf("[Hawal-v2] 🔌 Connecting to %s via [%s]...", e.cfg.ConnectAddr, e.cfg.CarrierKind)

		link, err := car.Dial(ctx, carrier.Endpoint{
			Network:    "tcp",
			Address:    e.cfg.ConnectAddr,
			ServerName: e.cfg.ServerName,
		}, carrier.Options{
			ConnectTimeout: 10 * time.Second,
			KeepAlive:      30 * time.Second,
			NoDelay:        e.cfg.NoDelay,
		})

		if err != nil {
			log.Printf("[Hawal-v2] ⚠️ Dial error: %v. Reconnecting in 3s...", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(3 * time.Second):
				continue
			}
		}

		res, err := e.handshaker.Handshake(ctx, secure.RoleInitiator, link, "client", secure.Limits{
			Timeout: 10 * time.Second,
		})
		if err != nil {
			log.Printf("[Hawal-v2] ⚠️ Handshake failed: %v. Reconnecting in 3s...", err)
			_ = link.Close()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(3 * time.Second):
				continue
			}
		}

		opts := record.DefaultOptions()
		opts.MaskLength = true

		codec, err := record.New(
			res.Secrets.SendAEAD,
			res.Secrets.ReceiveAEAD,
			res.Secrets.SendNonce,
			res.Secrets.ReceiveNonce,
			res.Secrets.Transcript,
			opts,
		)
		if err != nil {
			log.Printf("[Hawal-v2] Failed to initialize codec: %v", err)
			_ = link.Close()
			continue
		}

		sess, err := NewSessionWithOptions(link, codec, false, e.cfg.SessionOpts)
		if err != nil {
			log.Printf("[Hawal-v2] Failed to create session: %v", err)
			_ = link.Close()
			continue
		}

		e.mu.Lock()
		e.session = sess
		e.mu.Unlock()

		log.Printf("[Hawal-v2] 🟢 Stealth v2 tunnel active to %s (PFS: X25519, AEAD: ChaCha20-Poly1305)", e.cfg.ConnectAddr)

		// Serve egress streams received from server
		if err := ServeEgress(ctx, sess, e.portMap, e.cfg.NoDelay); err != nil && ctx.Err() == nil {
			log.Printf("[Hawal-v2] Tunnel dropped: %v. Reconnecting...", err)
		}

		_ = sess.Close()
		e.mu.Lock()
		if e.session == sess {
			e.session = nil
		}
		e.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}
