package tls

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
)

func TestTLSCarrierRoundTrip(t *testing.T) {
	c := New(Config{Insecure: true})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	acceptor, err := c.Listen(ctx, carrier.Bind{Address: "127.0.0.1:0"}, carrier.Options{})
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer acceptor.Close()

	addr := acceptor.Addr().String()

	var (
		serverLink carrier.Link
		serverErr  error
		wg         sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		serverLink, serverErr = acceptor.Accept()
		if serverErr == nil {
			defer serverLink.Close()
			_, serverErr = io.Copy(serverLink, serverLink)
		}
	}()

	clientLink, err := c.Dial(ctx, carrier.Endpoint{Address: addr}, carrier.Options{})
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer clientLink.Close()

	if clientLink.Kind() != carrier.KindTLSHTTP {
		t.Fatalf("expected KindTLSHTTP, got %v", clientLink.Kind())
	}
	if clientLink.ID() == "" {
		t.Fatal("expected non-empty link ID")
	}

	payload := []byte("hello tls carrier")
	if _, err := clientLink.Write(payload); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(clientLink, buf); err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if !bytes.Equal(buf, payload) {
		t.Fatalf("got %q, want %q", buf, payload)
	}
}
