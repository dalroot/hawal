package engine

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
)

// blackholeLink drops all outbound writes after blackhole is enabled.
type blackholeLink struct {
	net.Conn
	blackholed atomic.Bool
}

func (b *blackholeLink) Write(p []byte) (int, error) {
	if b.blackholed.Load() {
		// Simulate silent blackhole: writes succeed locally without error, but never reach peer
		return len(p), nil
	}
	return b.Conn.Write(p)
}

func (b *blackholeLink) ID() string               { return "blackhole-link" }
func (b *blackholeLink) Kind() carrier.Kind       { return carrier.KindTCP }
func (b *blackholeLink) EstablishedAt() time.Time { return time.Now() }

func TestSilentBlackholeDeadLinkDetection(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	codec1 := dummyCodec(t)
	codec2 := dummyCodec(t)

	bhLink := &blackholeLink{Conn: c1}
	normalLink := &dummyLink{Conn: c2, id: "server-link", established: time.Now()}

	opts := SessionOptions{
		PingInterval:    40 * time.Millisecond,
		DeadLinkTimeout: 120 * time.Millisecond,
	}

	clientSess, err := NewSessionWithOptions(bhLink, codec1, false, opts)
	if err != nil {
		t.Fatalf("failed to create client session: %v", err)
	}
	defer clientSess.Close()

	serverSess, err := NewSessionWithOptions(normalLink, codec2, true, opts)
	if err != nil {
		t.Fatalf("failed to create server session: %v", err)
	}
	defer serverSess.Close()

	// Wait 80ms to confirm initial ping/pong works and updates activity
	time.Sleep(80 * time.Millisecond)
	select {
	case <-clientSess.closed:
		t.Fatal("client session closed prematurely")
	default:
	}

	// Trigger silent blackhole: packets are absorbed silently, server stops receiving and responding
	bhLink.blackholed.Store(true)

	// Within DeadLinkTimeout (120ms), the client session must detect the silent blackhole and close
	select {
	case <-clientSess.closed:
		// Success! Dead link detected
		t.Logf("client session cleanly detected silent blackhole: %v", clientSess.closeErr)
	case <-time.After(350 * time.Millisecond):
		t.Fatal("timed out waiting for silent blackhole dead link detection")
	}
}

func TestWaitForSessionDuringRecovery(t *testing.T) {
	eng := &Engine{
		portMap: make(map[string]string),
	}

	ctx := context.Background()

	c1, c2 := net.Pipe()
	defer c2.Close()

	// Initially session is nil
	start := time.Now()
	go func() {
		time.Sleep(80 * time.Millisecond)

		codec := dummyCodec(t)
		link := &dummyLink{Conn: c1, id: "recovered-link", established: time.Now()}
		sess, err := NewSession(link, codec, false)
		if err != nil {
			t.Errorf("failed to create new session: %v", err)
			return
		}

		eng.mu.Lock()
		eng.session = sess
		eng.mu.Unlock()
	}()

	sess := eng.WaitForSession(ctx, 500*time.Millisecond)
	elapsed := time.Since(start)

	if sess == nil {
		t.Fatalf("WaitForSession returned nil, expected recovered session")
	}
	defer sess.Close()

	if elapsed < 70*time.Millisecond || elapsed > 350*time.Millisecond {
		t.Errorf("unexpected recovery wait duration: %v", elapsed)
	}
}

func TestListenerPortPermanenceDuringCarrierDrop(t *testing.T) {
	eng := &Engine{
		portMap: make(map[string]string),
	}

	rule := Rule{
		ListenPort: "127.0.0.1:0", // ephemeral local port
		TargetAddr: "127.0.0.1:19999",
	}

	fl, err := StartForwardListener(rule, true, eng.getReadySession)
	if err != nil {
		t.Fatalf("failed to start forward listener: %v", err)
	}
	defer fl.Close()

	addr := fl.listener.Addr().String()

	// Dial listener while engine has no session; listener should accept and wait
	conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to connect to listener port: %v", err)
	}
	_ = conn.Close()

	// Verify listener is still active and alive
	conn2, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("listener closed unexpectedly: %v", err)
	}
	_ = conn2.Close()
}
