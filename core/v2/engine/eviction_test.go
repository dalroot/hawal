package engine_test

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
	"github.com/dalroot/hawal/core/v2/engine"
	"github.com/dalroot/hawal/core/v2/record"
	"golang.org/x/crypto/chacha20poly1305"
)

type recordingLink struct {
	net.Conn
	writes atomic.Int64
	closed atomic.Bool
}

func newRecordingLink(c net.Conn) *recordingLink {
	return &recordingLink{Conn: c}
}

func (r *recordingLink) ID() string               { return "rec-link" }
func (r *recordingLink) Kind() carrier.Kind       { return carrier.KindTCP }
func (r *recordingLink) EstablishedAt() time.Time { return time.Now() }

func (r *recordingLink) Write(b []byte) (int, error) {
	if r.closed.Load() {
		return 0, net.ErrClosed
	}
	r.writes.Add(int64(len(b)))
	return r.Conn.Write(b)
}

func (r *recordingLink) Close() error {
	r.closed.Store(true)
	return r.Conn.Close()
}

func TestSessionAtomicEvictionStopsWrites(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	recLink := newRecordingLink(c1)

	key := make([]byte, chacha20poly1305.KeySize)
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		t.Fatal(err)
	}

	nonce := make([]byte, aead.NonceSize())
	codec, err := record.New(aead, aead, nonce, nonce, [32]byte{}, record.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	opts := engine.SessionOptions{
		PingInterval:    10 * time.Millisecond,
		DeadLinkTimeout: 100 * time.Millisecond,
	}

	sess, err := engine.NewSessionWithOptions(recLink, codec, false, opts)
	if err != nil {
		t.Fatal(err)
	}

	if sess.IsEvicted() {
		t.Fatalf("session should not be evicted initially")
	}

	// Wait a moment for background pump
	time.Sleep(20 * time.Millisecond)

	// Atomically evict session
	if err := sess.Evict(); err != nil {
		t.Fatalf("Evict error: %v", err)
	}

	if !sess.IsEvicted() {
		t.Fatalf("session must report IsEvicted == true")
	}

	initialWrites := recLink.writes.Load()

	// Attempt to open streams or enqueue data after eviction
	_, err = sess.OpenStream("127.0.0.1:8080")
	if err == nil {
		t.Fatalf("expected error opening stream on evicted session, got nil")
	}

	// Allow goroutines to cycle
	time.Sleep(50 * time.Millisecond)

	// Assert that NO additional writes reached the link after eviction
	finalWrites := recLink.writes.Load()
	if finalWrites > initialWrites {
		t.Fatalf("evicted session leaked writes to wire: before=%d after=%d", initialWrites, finalWrites)
	}
}
