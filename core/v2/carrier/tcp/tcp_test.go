package tcp

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
)

func TestCarrierRoundTrip(t *testing.T) {
	c := Carrier{}
	listener, err := c.Listen(context.Background(), carrier.Bind{Address: "127.0.0.1:0"}, carrier.Options{NoDelay: true})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverErr := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverErr <- acceptErr
			return
		}
		defer conn.Close()
		_, acceptErr = io.CopyN(conn, conn, 4)
		serverErr <- acceptErr
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	client, err := c.Dial(ctx, carrier.Endpoint{Address: listener.Addr().String()}, carrier.Options{NoDelay: true})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if client.Kind() != carrier.KindTCP || client.ID() == "" || client.EstablishedAt().IsZero() {
		t.Fatalf("invalid link metadata: kind=%q id=%q at=%v", client.Kind(), client.ID(), client.EstablishedAt())
	}
	if _, err = client.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	response := make([]byte, 4)
	if _, err = io.ReadFull(client, response); err != nil {
		t.Fatal(err)
	}
	if string(response) != "ping" {
		t.Fatalf("unexpected response %q", response)
	}
	if err = <-serverErr; err != nil {
		t.Fatal(err)
	}
}
