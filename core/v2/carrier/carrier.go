// Package carrier defines the transport boundary used by Hawal Core v2.
package carrier

import (
	"context"
	"net"
	"time"
)

// Kind identifies a carrier family without exposing implementation details to
// the secure-session and multiplexing layers.
type Kind string

const (
	KindTCP      Kind = "tcp"
	KindTLSHTTP  Kind = "tls-http"
	KindQUIC     Kind = "quic"
	KindRawOwned Kind = "raw-owned"
	KindRawPaq   Kind = "rawpaq"
)

// Capability describes optional behavior a carrier can provide.
type Capability uint32

const (
	CapabilityReliable Capability = 1 << iota
	CapabilityOrdered
	CapabilityDatagram
	CapabilityMigration
)

// Endpoint is the remote side of a carrier connection.
type Endpoint struct {
	Network    string
	Address    string
	ServerName string
}

// Bind is the local address on which a carrier accepts links.
type Bind struct {
	Network string
	Address string
}

// Options contains carrier-neutral socket behavior. A carrier may ignore an
// option only when its transport cannot represent it.
type Options struct {
	ConnectTimeout time.Duration
	KeepAlive      time.Duration
	NoDelay        bool
}

// Link is one established carrier path. ID is random and process-local; it is
// not a peer identity or authentication token.
type Link interface {
	net.Conn
	ID() string
	Kind() Kind
	EstablishedAt() time.Time
}

// Acceptor accepts links until Close is called.
type Acceptor interface {
	Accept() (Link, error)
	Addr() net.Addr
	Close() error
}

// Carrier creates and accepts links. Authentication and payload encryption are
// deliberately owned by the secure-session layer, not by this interface.
type Carrier interface {
	Kind() Kind
	Capabilities() Capability
	Dial(ctx context.Context, endpoint Endpoint, options Options) (Link, error)
	Listen(ctx context.Context, bind Bind, options Options) (Acceptor, error)
}
