package rawpaq

import (
	"context"
	"net"
	"testing"
)

func TestSourcePortValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SourcePort = 54241
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config with SourcePort 54241, got error: %v", err)
	}

	cfg.SourcePort = -1
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for negative SourcePort, got nil")
	}

	cfg.SourcePort = 70000
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for SourcePort > 65535, got nil")
	}
}

func TestSourcePortCarrierRequest(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SourcePort = 54241
	backend := &mockBackend{}

	c, err := New(cfg, backend, nil)
	if err != nil {
		t.Fatalf("New carrier error: %v", err)
	}

	req := c.request(RoleDialer, "0.0.0.0:0", "192.209.62.115:54241")
	if req.SourcePort != 54241 {
		t.Fatalf("expected req.SourcePort == 54241, got %d", req.SourcePort)
	}
}

type mockBackend struct{}

func (m *mockBackend) Name() string { return "mock" }
func (m *mockBackend) Preflight(_ context.Context, _ PacketRequest) (PreflightReport, error) {
	return PreflightReport{Ready: true}, nil
}
func (m *mockBackend) Open(_ context.Context, _ PacketRequest) (net.PacketConn, error) {
	return nil, nil
}
