// Package tcp implements the baseline TCP carrier for Hawal Core v2.
package tcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
)

// Carrier is a control carrier used for comparisons and ordinary TCP paths.
// It does not provide session authentication or payload encryption by itself.
type Carrier struct{}

func (Carrier) Kind() carrier.Kind { return carrier.KindTCP }

func (Carrier) Capabilities() carrier.Capability {
	return carrier.CapabilityReliable | carrier.CapabilityOrdered
}

func (Carrier) Dial(ctx context.Context, endpoint carrier.Endpoint, options carrier.Options) (carrier.Link, error) {
	network := endpoint.Network
	if network == "" {
		network = "tcp"
	}
	dialer := net.Dialer{Timeout: options.ConnectTimeout, KeepAlive: options.KeepAlive}
	conn, err := dialer.DialContext(ctx, network, endpoint.Address)
	if err != nil {
		return nil, fmt.Errorf("tcp carrier dial: %w", err)
	}
	configure(conn, options)
	return newLink(conn)
}

func (Carrier) Listen(ctx context.Context, bind carrier.Bind, options carrier.Options) (carrier.Acceptor, error) {
	network := bind.Network
	if network == "" {
		network = "tcp"
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, network, bind.Address)
	if err != nil {
		return nil, fmt.Errorf("tcp carrier listen: %w", err)
	}
	return &acceptor{Listener: listener, options: options}, nil
}

type acceptor struct {
	net.Listener
	options carrier.Options
}

func (a *acceptor) Accept() (carrier.Link, error) {
	conn, err := a.Listener.Accept()
	if err != nil {
		return nil, err
	}
	configure(conn, a.options)
	return newLink(conn)
}

type link struct {
	net.Conn
	id            string
	establishedAt time.Time
}

func newLink(conn net.Conn) (*link, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("tcp carrier link id: %w", err)
	}
	return &link{Conn: conn, id: hex.EncodeToString(raw[:]), establishedAt: time.Now().UTC()}, nil
}

func (l *link) ID() string               { return l.id }
func (l *link) Kind() carrier.Kind       { return carrier.KindTCP }
func (l *link) EstablishedAt() time.Time { return l.establishedAt }

func configure(conn net.Conn, options carrier.Options) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	_ = tcpConn.SetNoDelay(options.NoDelay)
	if options.KeepAlive > 0 {
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetKeepAlivePeriod(options.KeepAlive)
	}
}
