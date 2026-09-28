package rawpaq

import (
	"context"
	"errors"
	"net"
)

// UDPBackend is a rootless diagnostic backend. It validates KCP lifecycle and
// integration tests but is not the production raw-packet implementation.
type UDPBackend struct{}

func (UDPBackend) Name() string { return "udp-diagnostic" }

func (UDPBackend) Preflight(_ context.Context, request PacketRequest) (PreflightReport, error) {
	if request.LocalAddress == "" {
		return PreflightReport{Checks: []Check{{Code: CheckAddress, Detail: "local address missing"}}}, nil
	}
	if request.Role != RoleDialer && request.Role != RoleListener {
		return PreflightReport{}, errors.New("unknown packet role")
	}
	return PreflightReport{Ready: true, Checks: []Check{{Code: CheckAddress, OK: true}}}, nil
}

func (UDPBackend) Open(ctx context.Context, request PacketRequest) (net.PacketConn, error) {
	listenConfig := net.ListenConfig{}
	return listenConfig.ListenPacket(ctx, "udp", request.LocalAddress)
}
