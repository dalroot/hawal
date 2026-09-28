package secure

import (
	"bytes"
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
)

type pipeLink struct {
	net.Conn
	id   string
	kind carrier.Kind
	est  time.Time
}

func (p *pipeLink) ID() string              { return p.id }
func (p *pipeLink) Kind() carrier.Kind      { return p.kind }
func (p *pipeLink) EstablishedAt() time.Time { return p.est }

func newPipeLinks() (*pipeLink, *pipeLink) {
	c1, c2 := net.Pipe()
	now := time.Now()
	return &pipeLink{Conn: c1, id: "client-link", kind: carrier.KindTCP, est: now},
		&pipeLink{Conn: c2, id: "server-link", kind: carrier.KindTCP, est: now}
}

func TestNoiseHandshakerMutualSuccess(t *testing.T) {
	cLink, sLink := newPipeLinks()
	defer cLink.Close()
	defer sLink.Close()

	token := "very-secret-shared-token-2026"
	cHandshaker := NewNoiseHandshaker(token)
	sHandshaker := NewNoiseHandshaker(token)

	var (
		cRes Result
		sRes Result
		cErr error
		sErr error
		wg   sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		cRes, cErr = cHandshaker.Handshake(context.Background(), RoleInitiator, cLink, "server-peer", Limits{})
	}()

	go func() {
		defer wg.Done()
		sRes, sErr = sHandshaker.Handshake(context.Background(), RoleResponder, sLink, "client-peer", Limits{})
	}()

	wg.Wait()

	if cErr != nil {
		t.Fatalf("client handshake failed: %v", cErr)
	}
	if sErr != nil {
		t.Fatalf("server handshake failed: %v", sErr)
	}

	// Verify transcripts match
	if cRes.Secrets.Transcript != sRes.Secrets.Transcript {
		t.Fatal("transcripts do not match")
	}

	// Verify encryption between client SendAEAD and server ReceiveAEAD
	clientPlain := []byte("hello from client")
	nonce := cRes.Secrets.SendNonce
	ciphertext := cRes.Secrets.SendAEAD.Seal(nil, nonce, clientPlain, cRes.Secrets.Transcript[:])

	serverDecrypted, err := sRes.Secrets.ReceiveAEAD.Open(nil, nonce, ciphertext, sRes.Secrets.Transcript[:])
	if err != nil {
		t.Fatalf("server failed to decrypt client data: %v", err)
	}
	if !bytes.Equal(serverDecrypted, clientPlain) {
		t.Fatalf("got %q, want %q", serverDecrypted, clientPlain)
	}

	// Verify encryption between server SendAEAD and client ReceiveAEAD
	serverPlain := []byte("hello from server")
	sNonce := sRes.Secrets.SendNonce
	sCiphertext := sRes.Secrets.SendAEAD.Seal(nil, sNonce, serverPlain, sRes.Secrets.Transcript[:])

	clientDecrypted, err := cRes.Secrets.ReceiveAEAD.Open(nil, sNonce, sCiphertext, cRes.Secrets.Transcript[:])
	if err != nil {
		t.Fatalf("client failed to decrypt server data: %v", err)
	}
	if !bytes.Equal(clientDecrypted, serverPlain) {
		t.Fatalf("got %q, want %q", clientDecrypted, serverPlain)
	}
}

func TestNoiseHandshakerTokenMismatch(t *testing.T) {
	cLink, sLink := newPipeLinks()
	defer cLink.Close()
	defer sLink.Close()

	cHandshaker := NewNoiseHandshaker("token-A")
	sHandshaker := NewNoiseHandshaker("token-B")

	var (
		cErr error
		sErr error
		wg   sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		_, cErr = cHandshaker.Handshake(context.Background(), RoleInitiator, cLink, "server", Limits{Timeout: 500 * time.Millisecond})
	}()

	go func() {
		defer wg.Done()
		_, sErr = sHandshaker.Handshake(context.Background(), RoleResponder, sLink, "client", Limits{Timeout: 500 * time.Millisecond})
	}()

	wg.Wait()

	if sErr == nil {
		t.Fatal("expected server to reject client with mismatched token")
	}
	if cErr == nil {
		t.Fatal("expected client to fail handshake")
	}
}

func TestFirstFlightHasVariableSize(t *testing.T) {
	token := "test-token"
	handshaker := NewNoiseHandshaker(token)

	sizes := make(map[int]bool)
	for i := 0; i < 10; i++ {
		cLink, sLink := newPipeLinks()
		go func() {
			_, _ = handshaker.Handshake(context.Background(), RoleInitiator, cLink, "server", Limits{Timeout: 500 * time.Millisecond})
		}()

		var buf [1024]byte
		n, _ := sLink.Read(buf[:])
		_ = cLink.Close()
		_ = sLink.Close()
		if n > 0 {
			sizes[n] = true
		}
	}

	if len(sizes) <= 1 {
		t.Fatalf("expected variable first flight sizes, got: %v", sizes)
	}
}
