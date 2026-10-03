package rawpaq

import (
	"context"
	"net"
)

type Role uint8

const (
	RoleDialer Role = iota + 1
	RoleListener
)

// PacketRequest is the minimum information a packet backend needs. A PCAP
// implementation may use InterfaceName and RouterMAC; a rootless test backend
// can use LocalAddress only.
type PacketRequest struct {
	Role          Role
	InterfaceName string
	LocalAddress  string
	RouterMAC     string
	RemoteAddress string
}

type CheckCode string

const (
	CheckPlatform   CheckCode = "platform"
	CheckPrivileges CheckCode = "privileges"
	CheckInterface  CheckCode = "interface"
	CheckAddress    CheckCode = "address"
	CheckGateway    CheckCode = "gateway"
	CheckPacketIO   CheckCode = "packet-io"
)

type Check struct {
	Code   CheckCode
	OK     bool
	Detail string
}

type PreflightReport struct {
	Ready  bool
	Checks []Check
}

// PacketBackend owns low-level packet capture/injection. Open must either
// return a fully initialized PacketConn or clean up every partial resource.
type PacketBackend interface {
	Name() string
	Preflight(context.Context, PacketRequest) (PreflightReport, error)
	Open(context.Context, PacketRequest) (net.PacketConn, error)
}

// DefaultBackend returns the native LinuxRawBackend on Linux when running with
// appropriate capabilities/root, falling back to UDPBackend otherwise.
func DefaultBackend() PacketBackend {
	return defaultPlatformBackend()
}

