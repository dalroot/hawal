package rawpaq

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
)

func TestKCPRoundTripWithoutSMux(t *testing.T) {
	config := DefaultConfig()
	config.Mode = ModeFast3
	config.LocalAddress = "127.0.0.1:0"
	c, err := New(config, UDPBackend{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := c.Listen(context.Background(), carrier.Bind{Address: "127.0.0.1:0"}, carrier.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverErr := make(chan error, 1)
	go func() {
		link, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverErr <- acceptErr
			return
		}
		defer link.Close()
		_, acceptErr = io.CopyN(link, link, 4)
		serverErr <- acceptErr
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := c.Dial(ctx, carrier.Endpoint{Address: listener.Addr().String()}, carrier.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	_ = client.SetDeadline(time.Now().Add(10 * time.Second))
	if client.Kind() != carrier.KindRawPaq || client.ID() == "" {
		t.Fatalf("invalid rawpaq link metadata: %q %q", client.Kind(), client.ID())
	}
	if _, err = client.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	response := make([]byte, 4)
	if _, err = io.ReadFull(client, response); err != nil {
		t.Fatal(err)
	}
	if string(response) != "ping" {
		t.Fatalf("unexpected echo %q", response)
	}
	if err = <-serverErr; err != nil {
		t.Fatal(err)
	}
}

type failedBackend struct{}

func (failedBackend) Name() string { return "failed" }
func (failedBackend) Preflight(context.Context, PacketRequest) (PreflightReport, error) {
	return PreflightReport{Checks: []Check{{Code: CheckPrivileges, Detail: "missing capability"}}}, nil
}
func (failedBackend) Open(context.Context, PacketRequest) (net.PacketConn, error) {
	panic("Open must not run after failed preflight")
}

func TestFailedPreflightDoesNotOpenResources(t *testing.T) {
	config := DefaultConfig()
	config.LocalAddress = "127.0.0.1:0"
	c, err := New(config, failedBackend{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Dial(context.Background(), carrier.Endpoint{Address: "127.0.0.1:9999"}, carrier.Options{})
	if _, ok := err.(*PreflightError); !ok {
		t.Fatalf("Dial error = %T %v", err, err)
	}
}
