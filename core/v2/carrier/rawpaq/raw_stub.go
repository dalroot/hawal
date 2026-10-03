//go:build !linux

package rawpaq

func defaultPlatformBackend() PacketBackend {
	return UDPBackend{}
}
