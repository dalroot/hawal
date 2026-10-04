package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
	tcpcarrier "github.com/dalroot/hawal/core/v2/carrier/tcp"
)

// trueBlackholeConn drops outbound writes and absorbs inbound reads without returning EOF,
// perfectly replicating an in-path middlebox silent blackhole where the socket remains open.
type trueBlackholeConn struct {
	net.Conn
	blackholed atomic.Bool
	closed     atomic.Bool
	closeChan  chan struct{}
}

func newTrueBlackholeConn(c net.Conn) *trueBlackholeConn {
	return &trueBlackholeConn{
		Conn:      c,
		closeChan: make(chan struct{}),
	}
}

func (b *trueBlackholeConn) Write(p []byte) (int, error) {
	if b.closed.Load() {
		return 0, net.ErrClosed
	}
	if b.blackholed.Load() {
		// Silent drop: write succeeds locally into the void
		return len(p), nil
	}
	return b.Conn.Write(p)
}

func (b *trueBlackholeConn) Read(p []byte) (int, error) {
	if b.closed.Load() {
		return 0, net.ErrClosed
	}
	if b.blackholed.Load() {
		// In a true silent blackhole, no packets arrive from the network.
		// It does NOT receive EOF or RST from the peer. It blocks until locally closed.
		<-b.closeChan
		return 0, net.ErrClosed
	}
	return b.Conn.Read(p)
}

func (b *trueBlackholeConn) Close() error {
	if b.closed.CompareAndSwap(false, true) {
		close(b.closeChan)
		return b.Conn.Close()
	}
	return nil
}

func (b *trueBlackholeConn) ID() string               { return "true-blackhole-link" }
func (b *trueBlackholeConn) Kind() carrier.Kind       { return carrier.KindTCP }
func (b *trueBlackholeConn) EstablishedAt() time.Time { return time.Now() }

func TestSilentBlackholeDeadLinkDetection(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	codec1 := dummyCodec(t)
	codec2 := dummyCodec(t)

	bhLink := newTrueBlackholeConn(c1)
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

	// Trigger silent blackhole: packets are absorbed silently, server stops receiving and responding.
	// No EOF, no RST, no error from transport!
	bhLink.blackholed.Store(true)

	// Within DeadLinkTimeout (120ms), the client session must detect the silent blackhole purely via Liveness
	select {
	case <-clientSess.closed:
		errStr := clientSess.closeErr.Error()
		t.Logf("client session cleanly detected silent blackhole: %v", errStr)
		if !strings.Contains(errStr, "dead link detected") {
			t.Fatalf("expected 'dead link detected' error, got: %v (should not be EOF!)", errStr)
		}
	case <-time.After(400 * time.Millisecond):
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

type testBlackholeCarrier struct {
	underlying carrier.Carrier
	onDial     func(net.Conn) net.Conn
}

func (t *testBlackholeCarrier) Kind() carrier.Kind { return carrier.KindTCP }
func (t *testBlackholeCarrier) Capabilities() carrier.Capability {
	return t.underlying.Capabilities()
}
func (t *testBlackholeCarrier) Dial(ctx context.Context, ep carrier.Endpoint, opts carrier.Options) (carrier.Link, error) {
	link, err := t.underlying.Dial(ctx, ep, opts)
	if err != nil {
		return nil, err
	}
	wrapped := t.onDial(link)
	return &wrappedTestLink{conn: wrapped, Link: link}, nil
}
func (t *testBlackholeCarrier) Listen(ctx context.Context, bind carrier.Bind, opts carrier.Options) (carrier.Acceptor, error) {
	return t.underlying.Listen(ctx, bind, opts)
}

type wrappedTestLink struct {
	carrier.Link
	conn net.Conn
}

func (w *wrappedTestLink) Read(p []byte) (int, error)         { return w.conn.Read(p) }
func (w *wrappedTestLink) Write(p []byte) (int, error)        { return w.conn.Write(p) }
func (w *wrappedTestLink) Close() error                       { return w.conn.Close() }
func (w *wrappedTestLink) LocalAddr() net.Addr                { return w.conn.LocalAddr() }
func (w *wrappedTestLink) RemoteAddr() net.Addr               { return w.conn.RemoteAddr() }
func (w *wrappedTestLink) SetDeadline(t time.Time) error      { return w.conn.SetDeadline(t) }
func (w *wrappedTestLink) SetReadDeadline(t time.Time) error  { return w.conn.SetReadDeadline(t) }
func (w *wrappedTestLink) SetWriteDeadline(t time.Time) error { return w.conn.SetWriteDeadline(t) }

func TestEndToEndSilentBlackholeRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Start a local echo service (representing destination service e.g. webserver / xray)
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echoLn.Close()
	echoAddr := echoLn.Addr().String()

	go func() {
		for {
			conn, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()

	// 2. Pick free port for server carrier
	carLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	carAddr := carLn.Addr().String()
	_ = carLn.Close()

	// Ingress port (e.g. 24704 on client)
	ingressLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ingressAddr := ingressLn.Addr().String()
	_ = ingressLn.Close()

	token := "test-secret-recovery-token"

	fastSessionOpts := SessionOptions{
		PingInterval:    30 * time.Millisecond,
		DeadLinkTimeout: 90 * time.Millisecond,
	}

	// 3. Start Server Engine
	srvEngine, err := NewEngine(Config{
		Mode:        "server",
		CarrierKind: carrier.KindTCP,
		BindAddr:    carAddr,
		Token:       token,
		NoDelay:     true,
		InsecureTLS: true,
		SessionOpts: fastSessionOpts,
	})
	if err != nil {
		t.Fatalf("NewEngine(server) error: %v", err)
	}
	defer srvEngine.Close()

	go func() {
		_ = srvEngine.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// 4. Create custom client registry with controllable blackhole carrier
	var firstConn *trueBlackholeConn
	var dialCount atomic.Int32

	blackholeReg := carrier.NewRegistry()
	_ = blackholeReg.Register(carrier.KindTCP, func() (carrier.Carrier, error) {
		return &testBlackholeCarrier{
			underlying: tcpcarrier.Carrier{},
			onDial: func(c net.Conn) net.Conn {
				count := dialCount.Add(1)
				if count == 1 {
					// First carrier: wrap in trueBlackholeConn
					tb := newTrueBlackholeConn(c)
					firstConn = tb
					return tb
				}
				// Subsequent carrier: clean normal connection
				return c
			},
		}, nil
	})

	// 5. Start Client Engine with forward rule: ingressAddr -> echoAddr
	cliEngine, err := NewEngineWithRegistry(Config{
		Mode:        "client",
		CarrierKind: carrier.KindTCP,
		ConnectAddr: carAddr,
		Ports:       []string{fmt.Sprintf("%s=%s", ingressAddr, echoAddr)},
		Token:       token,
		NoDelay:     true,
		InsecureTLS: true,
		SessionOpts: fastSessionOpts,
	}, blackholeReg)
	if err != nil {
		t.Fatalf("NewEngineWithRegistry(client) error: %v", err)
	}
	defer cliEngine.Close()

	go func() {
		_ = cliEngine.Start(ctx)
	}()

	// Wait for tunnel #1 to become active
	var cliSess *Session
	for i := 0; i < 50; i++ {
		cliSess = cliEngine.ActiveSession()
		if cliSess != nil && firstConn != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if cliSess == nil || firstConn == nil {
		t.Fatal("initial carrier #1 failed to establish")
	}

	// 6. Test initial user connection through ingressAddr
	testPayload1 := []byte("DATA_BEFORE_BLACKHOLE_VERIFIED_OK")
	conn1, err := net.DialTimeout("tcp", ingressAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to dial ingress port: %v", err)
	}
	_, _ = conn1.Write(testPayload1)
	recv1 := make([]byte, len(testPayload1))
	_, _ = io.ReadFull(conn1, recv1)
	_ = conn1.Close()

	if !bytes.Equal(testPayload1, recv1) {
		t.Fatalf("echoed data before blackhole mismatch: %s vs %s", recv1, testPayload1)
	}
	t.Logf("✅ Initial data transfer verified through forward port %s", ingressAddr)

	// 7. ACTIVATE TRUE SILENT BLACKHOLE ON CARRIER #1!
	// Packets vanish silently on the wire. No EOF, no RST!
	t0 := time.Now()
	firstConn.blackholed.Store(true)
	t.Logf("🚨 Silent blackhole triggered on Carrier #1 at %v", t0)

	// 8. Watch Client detect dead carrier purely via liveness timeout
	select {
	case <-cliSess.closed:
		tDetect := time.Since(t0)
		t.Logf("✅ Carrier #1 detected dead in %v: %v", tDetect, cliSess.closeErr)
		if !strings.Contains(cliSess.closeErr.Error(), "dead link detected") {
			t.Fatalf("expected dead link detected error, got: %v", cliSess.closeErr)
		}
	case <-time.After(600 * time.Millisecond):
		t.Fatal("timed out waiting for client to detect dead carrier #1")
	}

	// 9. Watch Client autonomously dial Carrier #2 to the SAME server port
	t.Logf("⏳ Awaiting autonomous re-anchor with Carrier #2...")
	recoveredSess := cliEngine.WaitForSession(ctx, 3*time.Second)
	if recoveredSess == nil {
		t.Fatal("client failed to autonomously establish Carrier #2")
	}
	tRecover := time.Since(t0)
	t.Logf("⚡ Carrier #2 established successfully in %v! Dial count = %d", tRecover, dialCount.Load())

	if dialCount.Load() < 2 {
		t.Fatalf("expected at least 2 dials (carrier #1 + carrier #2), got: %d", dialCount.Load())
	}

	// 10. TEST USER CONNECTION ON THE SAME INGRESS PORT (e.g. 24704)
	testPayload2 := []byte("DATA_AFTER_AUTONOMOUS_RECOVERY_VERIFIED_100%_SUCCESS")
	conn2, err := net.DialTimeout("tcp", ingressAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect to ingress port after recovery: %v", err)
	}
	defer conn2.Close()

	_, err = conn2.Write(testPayload2)
	if err != nil {
		t.Fatalf("failed to write data after recovery: %v", err)
	}

	recv2 := make([]byte, len(testPayload2))
	_, err = io.ReadFull(conn2, recv2)
	if err != nil {
		t.Fatalf("failed to read echoed data after recovery: %v", err)
	}

	if !bytes.Equal(testPayload2, recv2) {
		t.Fatalf("echoed data after recovery mismatch: %s vs %s", recv2, testPayload2)
	}

	t.Logf("🎉 SUCCESS: User connection cleanly transferred real payload after autonomous silent blackhole recovery!")
}
