package rawpaq

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
	"github.com/xtaci/kcp-go/v5"
)

// Carrier exposes one reliable ordered byte stream over raw packet IO and KCP.
// Hawal's secure session and mux run above it; Paqet's smux is not used.
type Carrier struct {
	config  Config
	backend PacketBackend
	block   kcp.BlockCrypt
}

func New(config Config, backend PacketBackend, outerBlock kcp.BlockCrypt) (*Carrier, error) {
	if config.LocalAddress == "" {
		config.LocalAddress = "0.0.0.0:0"
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if backend == nil {
		return nil, errors.New("rawpaq: packet backend is required")
	}
	return &Carrier{config: config, backend: backend, block: outerBlock}, nil
}

func (*Carrier) Kind() carrier.Kind { return carrier.KindRawPaq }

func (*Carrier) Capabilities() carrier.Capability {
	return carrier.CapabilityReliable | carrier.CapabilityOrdered
}

func (c *Carrier) Dial(ctx context.Context, endpoint carrier.Endpoint, _ carrier.Options) (carrier.Link, error) {
	if endpoint.Address == "" {
		return nil, errors.New("rawpaq: remote address is required")
	}
	request := c.request(RoleDialer, c.config.LocalAddress, endpoint.Address)
	if err := c.ensureReady(ctx, request); err != nil {
		return nil, err
	}
	packetConn, err := c.backend.Open(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("rawpaq: open %s packet backend: %w", c.backend.Name(), err)
	}
	session, err := kcp.NewConn(endpoint.Address, c.block, c.config.DataShards, c.config.ParityShards, packetConn)
	if err != nil {
		_ = packetConn.Close()
		return nil, fmt.Errorf("rawpaq: create KCP client: %w", err)
	}
	if err = applyKCP(session, c.config); err != nil {
		_ = session.Close()
		_ = packetConn.Close()
		return nil, err
	}
	return newLink(session, packetConn, c.config.WriteTimeout)
}

func (c *Carrier) Listen(ctx context.Context, bind carrier.Bind, _ carrier.Options) (carrier.Acceptor, error) {
	localAddress := bind.Address
	if localAddress == "" {
		localAddress = c.config.LocalAddress
	}
	if localAddress == "" {
		return nil, errors.New("rawpaq: local listen address is required")
	}
	request := c.request(RoleListener, localAddress, "")
	if err := c.ensureReady(ctx, request); err != nil {
		return nil, err
	}
	packetConn, err := c.backend.Open(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("rawpaq: open %s packet backend: %w", c.backend.Name(), err)
	}
	listener, err := kcp.ServeConn(c.block, c.config.DataShards, c.config.ParityShards, packetConn)
	if err != nil {
		_ = packetConn.Close()
		return nil, fmt.Errorf("rawpaq: create KCP listener: %w", err)
	}
	return &acceptor{listener: listener, packetConn: packetConn, config: c.config}, nil
}

func (c *Carrier) request(role Role, local, remote string) PacketRequest {
	return PacketRequest{Role: role, InterfaceName: c.config.InterfaceName, LocalAddress: local, RouterMAC: c.config.RouterMAC, RemoteAddress: remote, SourcePort: c.config.SourcePort}
}

func (c *Carrier) ensureReady(ctx context.Context, request PacketRequest) error {
	report, err := c.backend.Preflight(ctx, request)
	if err != nil {
		return fmt.Errorf("rawpaq: %s preflight: %w", c.backend.Name(), err)
	}
	if !report.Ready {
		return &PreflightError{Backend: c.backend.Name(), Checks: append([]Check(nil), report.Checks...)}
	}
	return nil
}

type PreflightError struct {
	Backend string
	Checks  []Check
}

func (e *PreflightError) Error() string {
	return "rawpaq: packet backend preflight failed: " + e.Backend
}

type acceptor struct {
	listener   *kcp.Listener
	packetConn net.PacketConn
	config     Config
	closeOnce  sync.Once
	closeErr   error
}

func (a *acceptor) Accept() (carrier.Link, error) {
	session, err := a.listener.AcceptKCP()
	if err != nil {
		return nil, err
	}
	if err = applyKCP(session, a.config); err != nil {
		_ = session.Close()
		return nil, err
	}
	return newLink(session, nil, a.config.WriteTimeout)
}

func (a *acceptor) Addr() net.Addr { return a.listener.Addr() }

func (a *acceptor) Close() error {
	a.closeOnce.Do(func() {
		if err := a.listener.Close(); err != nil {
			a.closeErr = err
		}
		if err := a.packetConn.Close(); err != nil && a.closeErr == nil {
			a.closeErr = err
		}
	})
	return a.closeErr
}

type link struct {
	*kcp.UDPSession
	packetConn   net.PacketConn
	id           string
	established  time.Time
	writeTimeout time.Duration
	closeOnce    sync.Once
	closeErr     error
}

func newLink(session *kcp.UDPSession, ownedPacketConn net.PacketConn, writeTimeout time.Duration) (*link, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		_ = session.Close()
		if ownedPacketConn != nil {
			_ = ownedPacketConn.Close()
		}
		return nil, fmt.Errorf("rawpaq: generate link ID: %w", err)
	}
	return &link{
		UDPSession:   session,
		packetConn:   ownedPacketConn,
		id:           hex.EncodeToString(raw[:]),
		established:  time.Now().UTC(),
		writeTimeout: writeTimeout,
	}, nil
}

func (l *link) ID() string               { return l.id }
func (l *link) Kind() carrier.Kind       { return carrier.KindRawPaq }
func (l *link) EstablishedAt() time.Time { return l.established }

func (l *link) Write(b []byte) (int, error) {
	if l.writeTimeout > 0 {
		_ = l.UDPSession.SetWriteDeadline(time.Now().Add(l.writeTimeout))
	}
	return l.UDPSession.Write(b)
}

func (l *link) Close() error {
	l.closeOnce.Do(func() {
		var pcErr error
		if l.packetConn != nil {
			pcErr = l.packetConn.Close()
		}
		udpErr := l.UDPSession.Close()
		if pcErr != nil {
			l.closeErr = pcErr
		} else {
			l.closeErr = udpErr
		}
	})
	return l.closeErr
}
