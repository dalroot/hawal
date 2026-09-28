package carrier_test

import (
	"testing"

	"github.com/dalroot/hawal/core/v2/carrier"
	tcpcarrier "github.com/dalroot/hawal/core/v2/carrier/tcp"
	tlscarrier "github.com/dalroot/hawal/core/v2/carrier/tls"
)

func TestRegistry(t *testing.T) {
	r := carrier.NewRegistry()
	if err := r.Register(carrier.KindTCP, func() (carrier.Carrier, error) { return tcpcarrier.Carrier{}, nil }); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(carrier.KindTLSHTTP, func() (carrier.Carrier, error) { return tlscarrier.New(tlscarrier.Config{Insecure: true}), nil }); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(carrier.KindTCP, func() (carrier.Carrier, error) { return tcpcarrier.Carrier{}, nil }); err == nil {
		t.Fatal("duplicate registration succeeded")
	}
	got, err := r.Build(carrier.KindTCP)
	if err != nil || got.Kind() != carrier.KindTCP {
		t.Fatalf("Build(TCP) = %#v, %v", got, err)
	}
	gotTLS, err := r.Build(carrier.KindTLSHTTP)
	if err != nil || gotTLS.Kind() != carrier.KindTLSHTTP {
		t.Fatalf("Build(TLS) = %#v, %v", gotTLS, err)
	}
	if _, err = r.Build(carrier.KindQUIC); err == nil {
		t.Fatal("unregistered carrier build succeeded")
	}
}
