package engine_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
	"github.com/dalroot/hawal/core/v2/engine"
)

func TestEngineLoopbackTCP(t *testing.T) {
	testEngineCarrier(t, carrier.KindTCP)
}

func TestEngineLoopbackTLS(t *testing.T) {
	testEngineCarrier(t, carrier.KindTLSHTTP)
}

func TestEngineLoopbackRawPaq(t *testing.T) {
	testEngineCarrier(t, carrier.KindRawPaq)
}

func testEngineCarrier(t *testing.T, kind carrier.Kind) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Start a local echo service
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

	// 2. Pick free ports for carrier and forward ingress
	carLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	carAddr := carLn.Addr().String()
	_ = carLn.Close()

	ingressLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ingressAddr := ingressLn.Addr().String()
	_ = ingressLn.Close()

	token := "test-secret-token-12345678"

	// 3. Start Hawal v2 Server (Listening for carrier, and forward ingress -> echo target)
	srvEngine, err := engine.NewEngine(engine.Config{
		Mode:        "server",
		CarrierKind: kind,
		BindAddr:    carAddr,
		Ports:       []string{fmt.Sprintf("%s=%s", ingressAddr, echoAddr)},
		Token:       token,
		NoDelay:     true,
		InsecureTLS: true,
	})
	if err != nil {
		t.Fatalf("NewEngine(server) error: %v", err)
	}
	defer srvEngine.Close()

	go func() {
		_ = srvEngine.Start(ctx)
	}()

	// Give server time to bind
	time.Sleep(100 * time.Millisecond)

	// 4. Start Hawal v2 Client (Connecting to server, serves egress to echo service)
	cliEngine, err := engine.NewEngine(engine.Config{
		Mode:        "client",
		CarrierKind: kind,
		ConnectAddr: carAddr,
		Token:       token,
		NoDelay:     true,
		InsecureTLS: true,
	})
	if err != nil {
		t.Fatalf("NewEngine(client) error: %v", err)
	}
	defer cliEngine.Close()

	go func() {
		_ = cliEngine.Start(ctx)
	}()

	// Wait for tunnel and session to become active
	var activeSession *engine.Session
	for i := 0; i < 50; i++ {
		activeSession = srvEngine.ActiveSession()
		if activeSession != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if activeSession == nil {
		t.Fatal("v2 session failed to establish within timeout")
	}

	// 5. Connect test client to ingress port and send test payload
	testPayload := make([]byte, 64*1024) // 64 KB
	if _, err := rand.Read(testPayload); err != nil {
		t.Fatal(err)
	}

	conn, err := net.DialTimeout("tcp", ingressAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("Dial ingress %s error: %v", ingressAddr, err)
	}
	defer conn.Close()

	recvBuf := make([]byte, len(testPayload))
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		_, _ = io.ReadFull(conn, recvBuf)
	}()

	if _, err := conn.Write(testPayload); err != nil {
		t.Fatalf("Write test payload error: %v", err)
	}

	wg.Wait()

	if !bytes.Equal(testPayload, recvBuf) {
		t.Fatalf("Echoed payload does not match sent payload (%d bytes)", len(testPayload))
	}
}
