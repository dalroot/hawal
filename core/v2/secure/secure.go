// Package secure defines the authenticated-session boundary.
package secure

import (
	"context"
	"crypto/cipher"
	"io"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
)

// Role determines direction-specific key assignment.
type Role uint8

const (
	RoleInitiator Role = iota + 1
	RoleResponder
)

// IdentityRef names provisioned key material without exposing it.
type IdentityRef string

// Limits bound handshake work before authentication succeeds.
type Limits struct {
	Timeout       time.Duration
	MaxMessage    int
	MaxConcurrent int
}

// Secrets contains direction-separated traffic protectors produced by a
// vetted handshake implementation. Ownership transfers to the caller.
type Secrets struct {
	SendAEAD     cipher.AEAD
	ReceiveAEAD  cipher.AEAD
	SendNonce    []byte
	ReceiveNonce []byte
	Transcript   [32]byte
}

// Result is returned only after peer authentication and transcript binding.
type Result struct {
	Peer       IdentityRef
	Secrets    Secrets
	ExportedAt time.Time
}

// Handshaker may be implemented using TLS 1.3 or an audited Noise library.
// Record and mux packages never depend on the concrete handshake.
type Handshaker interface {
	Handshake(ctx context.Context, role Role, link carrier.Link, identity IdentityRef, limits Limits) (Result, error)
}

// Destroy overwrites nonce material and drops AEAD references. Go cannot
// guarantee compiler-proof memory erasure, but callers still minimize lifetime.
func (s *Secrets) Destroy() {
	for i := range s.SendNonce {
		s.SendNonce[i] = 0
	}
	for i := range s.ReceiveNonce {
		s.ReceiveNonce[i] = 0
	}
	s.SendAEAD = nil
	s.ReceiveAEAD = nil
}

// CloseWriter describes transports that support a half-close.
type CloseWriter interface {
	io.Writer
	CloseWrite() error
}
