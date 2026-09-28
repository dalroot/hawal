// Package tls implements the standard TLS 1.3 carrier for Hawal Core v2.
// It camouflages session records inside genuine TLS 1.3 traffic.
package tls

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
)

type Config struct {
	Certificate tls.Certificate
	ServerName  string
	Insecure    bool
	ALPN        []string
}

// Carrier implements carrier.Carrier for standard TLS 1.3 streams.
type Carrier struct {
	config Config
}

func New(config Config) *Carrier {
	if len(config.ALPN) == 0 {
		config.ALPN = []string{"h2", "http/1.1"}
	}
	return &Carrier{config: config}
}

func (Carrier) Kind() carrier.Kind { return carrier.KindTLSHTTP }

func (Carrier) Capabilities() carrier.Capability {
	return carrier.CapabilityReliable | carrier.CapabilityOrdered
}

func (c *Carrier) Dial(ctx context.Context, endpoint carrier.Endpoint, options carrier.Options) (carrier.Link, error) {
	network := endpoint.Network
	if network == "" {
		network = "tcp"
	}
	dialer := net.Dialer{Timeout: options.ConnectTimeout, KeepAlive: options.KeepAlive}
	rawConn, err := dialer.DialContext(ctx, network, endpoint.Address)
	if err != nil {
		return nil, fmt.Errorf("tls carrier dial: %w", err)
	}

	serverName := endpoint.ServerName
	if serverName == "" {
		serverName = c.config.ServerName
	}
	if serverName == "" {
		host, _, splitErr := net.SplitHostPort(endpoint.Address)
		if splitErr == nil {
			serverName = host
		}
	}

	tlsConfig := &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: c.config.Insecure,
		MinVersion:         tls.VersionTLS13,
		NextProtos:         c.config.ALPN,
	}

	tlsConn := tls.Client(rawConn, tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("tls carrier handshake: %w", err)
	}

	return newLink(tlsConn)
}

func (c *Carrier) Listen(ctx context.Context, bind carrier.Bind, options carrier.Options) (carrier.Acceptor, error) {
	network := bind.Network
	if network == "" {
		network = "tcp"
	}

	cert := c.config.Certificate
	if len(cert.Certificate) == 0 {
		// Generate ephemeral self-signed cert if none provided
		var err error
		cert, err = generateEphemeralCert()
		if err != nil {
			return nil, fmt.Errorf("tls carrier generate cert: %w", err)
		}
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   c.config.ALPN,
	}

	rawListener, err := (&net.ListenConfig{}).Listen(ctx, network, bind.Address)
	if err != nil {
		return nil, fmt.Errorf("tls carrier listen: %w", err)
	}

	tlsListener := tls.NewListener(rawListener, tlsConfig)
	return &acceptor{Listener: tlsListener, options: options}, nil
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
		return nil, fmt.Errorf("tls carrier link id: %w", err)
	}
	return &link{Conn: conn, id: hex.EncodeToString(raw[:]), establishedAt: time.Now().UTC()}, nil
}

func (l *link) ID() string               { return l.id }
func (l *link) Kind() carrier.Kind       { return carrier.KindTLSHTTP }
func (l *link) EstablishedAt() time.Time { return l.establishedAt }

func generateEphemeralCert() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Hawal Ephemeral"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}, nil
}
