//go:build linux

package rawpaq

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"
)

func defaultPlatformBackend() PacketBackend {
	if os.Geteuid() == 0 {
		return &LinuxRawBackend{}
	}
	return UDPBackend{}
}

// LinuxRawBackend implements PacketBackend using AF_PACKET raw sockets on Linux.
// It bypasses conntrack and injects fake TCP Push/Ack ("PA") frames directly to the NIC.
type LinuxRawBackend struct{}

func (b *LinuxRawBackend) Name() string {
	return "linux-raw-tcp"
}

func (b *LinuxRawBackend) Preflight(ctx context.Context, req PacketRequest) (PreflightReport, error) {
	checks := []Check{
		{Code: CheckPlatform, OK: true, Detail: "linux"},
	}

	// Check privileges
	if os.Geteuid() != 0 {
		checks = append(checks, Check{
			Code:   CheckPrivileges,
			OK:     false,
			Detail: "root privileges or CAP_NET_RAW capability required",
		})
		return PreflightReport{Ready: false, Checks: checks}, nil
	}
	checks = append(checks, Check{Code: CheckPrivileges, OK: true, Detail: "uid 0 (root)"})

	// Check interface
	iface, _, err := resolveInterface(req.InterfaceName)
	if err != nil {
		checks = append(checks, Check{
			Code:   CheckInterface,
			OK:     false,
			Detail: fmt.Sprintf("resolve interface: %v", err),
		})
		return PreflightReport{Ready: false, Checks: checks}, nil
	}
	checks = append(checks, Check{Code: CheckInterface, OK: true, Detail: iface.Name})

	// Check local address
	_, portStr, err := net.SplitHostPort(req.LocalAddress)
	if err != nil && req.LocalAddress != "" {
		checks = append(checks, Check{
			Code:   CheckAddress,
			OK:     false,
			Detail: fmt.Sprintf("invalid local address: %v", err),
		})
		return PreflightReport{Ready: false, Checks: checks}, nil
	}
	port, _ := strconv.Atoi(portStr)
	if req.Role == RoleListener && port <= 0 {
		checks = append(checks, Check{
			Code:   CheckAddress,
			OK:     false,
			Detail: "listen port must be greater than 0",
		})
		return PreflightReport{Ready: false, Checks: checks}, nil
	}
	checks = append(checks, Check{Code: CheckAddress, OK: true, Detail: req.LocalAddress})

	// Probe test socket
	testFD, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, int(htons(unix.ETH_P_ALL)))
	if err != nil {
		checks = append(checks, Check{
			Code:   CheckPacketIO,
			OK:     false,
			Detail: fmt.Sprintf("probe AF_PACKET failed: %v", err),
		})
		return PreflightReport{Ready: false, Checks: checks}, nil
	}
	_ = unix.Close(testFD)
	checks = append(checks, Check{Code: CheckPacketIO, OK: true, Detail: "AF_PACKET socket created successfully"})

	return PreflightReport{Ready: true, Checks: checks}, nil
}

func (b *LinuxRawBackend) Open(ctx context.Context, req PacketRequest) (net.PacketConn, error) {
	iface, gwIP, err := resolveInterface(req.InterfaceName)
	if err != nil {
		return nil, fmt.Errorf("rawpaq: resolve interface: %w", err)
	}

	// Resolve local IP
	localIP, err := resolveLocalIPv4(iface, req.LocalAddress)
	if err != nil {
		return nil, fmt.Errorf("rawpaq: resolve local IP: %w", err)
	}

	// Resolve local port
	var localPort uint16
	if req.LocalAddress != "" {
		_, pStr, err := net.SplitHostPort(req.LocalAddress)
		if err == nil {
			if p, err := strconv.ParseUint(pStr, 10, 16); err == nil {
				localPort = uint16(p)
			}
		}
	}
	if localPort == 0 && req.SourcePort > 0 && req.SourcePort <= 65535 {
		localPort = uint16(req.SourcePort)
	}
	if localPort == 0 {
		ephem := pickEphemeralPort()
		if ephem > 0 && ephem <= 65535 {
			localPort = uint16(ephem)
		} else {
			localPort = 45000
		}
	}

	// Resolve remote endpoint (if dialer)
	var remoteIP net.IP
	var remotePort uint16
	if req.Role == RoleDialer && req.RemoteAddress != "" {
		host, pStr, err := net.SplitHostPort(req.RemoteAddress)
		if err == nil {
			ips, _ := net.LookupIP(host)
			for _, ip := range ips {
				if ip4 := ip.To4(); ip4 != nil {
					remoteIP = ip4
					break
				}
			}
			if p, err := strconv.ParseUint(pStr, 10, 16); err == nil {
				remotePort = uint16(p)
			}
		}
	}

	// Resolve router MAC
	var routerMAC net.HardwareAddr
	if req.RouterMAC != "" {
		routerMAC, err = net.ParseMAC(req.RouterMAC)
		if err != nil {
			return nil, fmt.Errorf("rawpaq: invalid router MAC %q: %w", req.RouterMAC, err)
		}
	} else {
		routerMAC, err = resolveGatewayMAC(iface.Name, gwIP)
		if err != nil {
			// Fallback: probe gateway and retry once
			probeGateway(gwIP)
			routerMAC, err = resolveGatewayMAC(iface.Name, gwIP)
			if err != nil {
				return nil, fmt.Errorf("rawpaq: detect gateway MAC on %s: %w", iface.Name, err)
			}
		}
	}

	// Create AF_PACKET raw socket
	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, int(htons(unix.ETH_P_IP)))
	if err != nil {
		return nil, fmt.Errorf("rawpaq: create AF_PACKET raw socket: %w", err)
	}

	// Bind to the specified network interface
	sll := &unix.SockaddrLinklayer{
		Protocol: htons(unix.ETH_P_IP),
		Ifindex:  iface.Index,
	}
	if err := unix.Bind(fd, sll); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("rawpaq: bind AF_PACKET to %s: %w", iface.Name, err)
	}

	// Set large kernel socket buffers (4 MB)
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, 4*1024*1024)
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, 4*1024*1024)

	// Attach in-kernel BPF filter to drop all unrelated traffic at kernel layer
	if prog, err := buildBPFFilter(req.Role, localPort, remotePort); err == nil {
		sockFilter := make([]unix.SockFilter, len(prog))
		for i, inst := range prog {
			sockFilter[i] = unix.SockFilter{Code: inst.Op, Jt: inst.Jt, Jf: inst.Jf, K: inst.K}
		}
		fprog := &unix.SockFprog{
			Len:    uint16(len(sockFilter)),
			Filter: &sockFilter[0],
		}
		_ = unix.SetsockoptSockFprog(fd, unix.SOL_SOCKET, unix.SO_ATTACH_FILTER, fprog)
	}

	conn := &rawTCPPacketConn{
		fd:         fd,
		iface:      iface,
		localIP:    localIP,
		localPort:  localPort,
		remoteIP:   remoteIP,
		remotePort: remotePort,
		routerMAC:  routerMAC,
		role:       req.Role,
	}
	var isn [4]byte
	_, _ = rand.Read(isn[:])
	conn.seqCounter.Store(binary.BigEndian.Uint32(isn[:]))

	return conn, nil
}

type rawTCPPacketConn struct {
	fd          int
	iface       *net.Interface
	localIP     net.IP
	localPort   uint16
	remoteIP    net.IP
	remotePort  uint16
	routerMAC   net.HardwareAddr
	role        Role
	seqCounter  atomic.Uint32
	lastRecvSeq atomic.Uint32
	closed      atomic.Bool
	mu          sync.Mutex
}

func (c *rawTCPPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	if c.closed.Load() {
		return 0, nil, net.ErrClosed
	}

	buf := make([]byte, 65535)
	for {
		if c.closed.Load() {
			return 0, nil, net.ErrClosed
		}

		n, _, err := unix.Recvfrom(c.fd, buf, 0)
		if err != nil {
			if c.closed.Load() || errors.Is(err, unix.EBADF) {
				return 0, nil, net.ErrClosed
			}
			if errors.Is(err, unix.EINTR) {
				continue
			}
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
				time.Sleep(1 * time.Millisecond)
				continue
			}
			return 0, nil, err
		}

		srcIP, _, srcPort, dstPort, seq, _, _, payload, err := parseRawEthernetFrame(buf[:n])
		if err != nil {
			continue
		}

		// Port filtering verification (backup in case BPF is bypassed)
		if dstPort != c.localPort {
			continue
		}
		if c.role == RoleDialer && c.remotePort > 0 && srcPort != c.remotePort {
			continue
		}

		// Update acknowledgment counter
		c.lastRecvSeq.Store(seq + uint32(len(payload)))

		if len(payload) == 0 {
			// Pure ACK or KeepAlive frame, skip payload return
			continue
		}

		copied := copy(p, payload)
		addr := &net.UDPAddr{IP: srcIP, Port: int(srcPort)}
		return copied, addr, nil
	}
}

func (c *rawTCPPacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	if c.closed.Load() {
		return 0, net.ErrClosed
	}

	var dstIP net.IP
	dstPort := int(c.remotePort)

	switch a := addr.(type) {
	case *net.UDPAddr:
		dstIP = a.IP
		dstPort = a.Port
	case *net.TCPAddr:
		dstIP = a.IP
		dstPort = a.Port
	default:
		host, pStr, err := net.SplitHostPort(addr.String())
		if err == nil {
			dstIP = net.ParseIP(host)
			dstPort, _ = strconv.Atoi(pStr)
		}
	}

	if dstIP == nil || dstPort <= 0 || dstPort > 65535 {
		return 0, errors.New("rawpaq: invalid destination address for raw write")
	}

	seq := c.seqCounter.Add(uint32(len(p))) - uint32(len(p))
	ack := c.lastRecvSeq.Load()

	// Flags: PSH | ACK (0x18) - matching Paqet "PA" signature
	flags := uint8(0x18)
	frame := buildRawEthernetFrame(
		c.iface.HardwareAddr,
		c.routerMAC,
		c.localIP,
		dstIP,
		c.localPort,
		uint16(dstPort),
		seq,
		ack,
		flags,
		p,
		46, // DSCP Expedited Forwarding
	)

	sll := &unix.SockaddrLinklayer{
		Protocol: htons(unix.ETH_P_IP),
		Ifindex:  c.iface.Index,
	}

	if err := unix.Sendto(c.fd, frame, 0, sll); err != nil {
		return 0, err
	}

	return len(p), nil
}

func (c *rawTCPPacketConn) Close() error {
	if c.closed.CompareAndSwap(false, true) {
		return unix.Close(c.fd)
	}
	return nil
}

func (c *rawTCPPacketConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: c.localIP, Port: int(c.localPort)}
}

func (c *rawTCPPacketConn) SetDeadline(t time.Time) error {
	_ = c.SetReadDeadline(t)
	return c.SetWriteDeadline(t)
}

func (c *rawTCPPacketConn) SetReadDeadline(t time.Time) error {
	var tv unix.Timeval
	if !t.IsZero() {
		dur := time.Until(t)
		if dur < 0 {
			dur = 1 * time.Microsecond
		}
		tv = unix.NsecToTimeval(dur.Nanoseconds())
	}
	return unix.SetsockoptTimeval(c.fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &tv)
}

func (c *rawTCPPacketConn) SetWriteDeadline(t time.Time) error {
	var tv unix.Timeval
	if !t.IsZero() {
		dur := time.Until(t)
		if dur < 0 {
			dur = 1 * time.Microsecond
		}
		tv = unix.NsecToTimeval(dur.Nanoseconds())
	}
	return unix.SetsockoptTimeval(c.fd, unix.SOL_SOCKET, unix.SO_SNDTIMEO, &tv)
}

// --- Wire Formatting & Checksum Helpers ---

func htons(v uint16) uint16 {
	return (v << 8) | (v >> 8)
}

func ntohs(v uint16) uint16 {
	return (v << 8) | (v >> 8)
}

func ipChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	return ^uint16(sum)
}

func tcpChecksum(srcIP, dstIP net.IP, tcpHeader, payload []byte) uint16 {
	var sum uint32
	src := srcIP.To4()
	dst := dstIP.To4()
	if src == nil || dst == nil {
		return 0
	}

	// Pseudo-header: SrcIP (4), DstIP (4), Zero (1), Protocol 6 (1), TCP Length (2)
	sum += uint32(binary.BigEndian.Uint16(src[0:2]))
	sum += uint32(binary.BigEndian.Uint16(src[2:4]))
	sum += uint32(binary.BigEndian.Uint16(dst[0:2]))
	sum += uint32(binary.BigEndian.Uint16(dst[2:4]))
	sum += uint32(unix.IPPROTO_TCP)
	tcpLen := len(tcpHeader) + len(payload)
	sum += uint32(tcpLen)

	// TCP Header
	for i := 0; i < len(tcpHeader)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(tcpHeader[i : i+2]))
	}
	if len(tcpHeader)%2 == 1 {
		sum += uint32(tcpHeader[len(tcpHeader)-1]) << 8
	}

	// TCP Payload
	for i := 0; i < len(payload)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(payload[i : i+2]))
	}
	if len(payload)%2 == 1 {
		sum += uint32(payload[len(payload)-1]) << 8
	}

	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	return ^uint16(sum)
}

func buildRawEthernetFrame(
	srcMAC, dstMAC net.HardwareAddr,
	srcIP, dstIP net.IP,
	srcPort, dstPort uint16,
	seq, ack uint32,
	flags uint8,
	payload []byte,
	dscp int,
) []byte {
	totalLen := 14 + 20 + 20 + len(payload)
	frame := make([]byte, totalLen)

	// 1. Ethernet Header (14 bytes)
	copy(frame[0:6], dstMAC)
	copy(frame[6:12], srcMAC)
	binary.BigEndian.PutUint16(frame[12:14], 0x0800) // EtherType IPv4

	// 2. IPv4 Header (20 bytes)
	ipHdr := frame[14:34]
	ipHdr[0] = 0x45 // Version 4, IHL 5 (20 bytes)
	ipHdr[1] = uint8(dscp << 2)
	binary.BigEndian.PutUint16(ipHdr[2:4], uint16(20+20+len(payload)))
	binary.BigEndian.PutUint16(ipHdr[4:6], uint16(seq&0xffff)) // Packet ID
	binary.BigEndian.PutUint16(ipHdr[6:8], 0x4000)             // Flags: Don't Fragment (DF)
	ipHdr[8] = 64                                              // TTL
	ipHdr[9] = unix.IPPROTO_TCP                                // Protocol: TCP (6)
	// Checksum initially 0
	copy(ipHdr[12:16], srcIP.To4())
	copy(ipHdr[16:20], dstIP.To4())
	binary.BigEndian.PutUint16(ipHdr[10:12], ipChecksum(ipHdr))

	// 3. TCP Header (20 bytes)
	tcpHdr := frame[34:54]
	binary.BigEndian.PutUint16(tcpHdr[0:2], srcPort)
	binary.BigEndian.PutUint16(tcpHdr[2:4], dstPort)
	binary.BigEndian.PutUint32(tcpHdr[4:8], seq)
	binary.BigEndian.PutUint32(tcpHdr[8:12], ack)
	tcpHdr[12] = 0x50 // Data Offset: 5 (20 bytes)
	tcpHdr[13] = flags
	binary.BigEndian.PutUint16(tcpHdr[14:16], 64240) // Window size
	// Checksum initially 0
	binary.BigEndian.PutUint16(tcpHdr[18:20], 0) // Urgent pointer

	// Compute TCP Checksum with pseudo-header
	binary.BigEndian.PutUint16(tcpHdr[16:18], tcpChecksum(srcIP, dstIP, tcpHdr, payload))

	// 4. Payload
	if len(payload) > 0 {
		copy(frame[54:], payload)
	}

	return frame
}

func parseRawEthernetFrame(frame []byte) (
	srcIP, dstIP net.IP,
	srcPort, dstPort uint16,
	seq, ack uint32,
	flags uint8,
	payload []byte,
	err error,
) {
	if len(frame) < 54 {
		return nil, nil, 0, 0, 0, 0, 0, nil, errors.New("frame too short")
	}

	// Ethernet check
	etherType := binary.BigEndian.Uint16(frame[12:14])
	if etherType != 0x0800 {
		return nil, nil, 0, 0, 0, 0, 0, nil, errors.New("not IPv4")
	}

	// IPv4 check
	ipStart := 14
	if frame[ipStart]>>4 != 4 {
		return nil, nil, 0, 0, 0, 0, 0, nil, errors.New("not IPv4")
	}
	ipHdrLen := int(frame[ipStart]&0x0F) * 4
	if ipHdrLen < 20 || len(frame) < ipStart+ipHdrLen+20 {
		return nil, nil, 0, 0, 0, 0, 0, nil, errors.New("invalid IP header length")
	}

	proto := frame[ipStart+9]
	if proto != unix.IPPROTO_TCP {
		return nil, nil, 0, 0, 0, 0, 0, nil, errors.New("not TCP")
	}

	srcIP = net.IP(frame[ipStart+12 : ipStart+16])
	dstIP = net.IP(frame[ipStart+16 : ipStart+20])
	totalIPLen := int(binary.BigEndian.Uint16(frame[ipStart+2 : ipStart+4]))

	// TCP check
	tcpStart := ipStart + ipHdrLen
	srcPort = binary.BigEndian.Uint16(frame[tcpStart : tcpStart+2])
	dstPort = binary.BigEndian.Uint16(frame[tcpStart+2 : tcpStart+4])
	seq = binary.BigEndian.Uint32(frame[tcpStart+4 : tcpStart+8])
	ack = binary.BigEndian.Uint32(frame[tcpStart+8 : tcpStart+12])
	tcpHdrLen := int(frame[tcpStart+12]>>4) * 4
	if tcpHdrLen < 20 {
		return nil, nil, 0, 0, 0, 0, 0, nil, errors.New("invalid TCP header length")
	}
	flags = frame[tcpStart+13]

	payloadStart := tcpStart + tcpHdrLen
	payloadEnd := ipStart + totalIPLen
	if payloadEnd > len(frame) {
		payloadEnd = len(frame)
	}

	if payloadStart < payloadEnd {
		payload = frame[payloadStart:payloadEnd]
	}

	return srcIP, dstIP, srcPort, dstPort, seq, ack, flags, payload, nil
}

func buildBPFFilter(role Role, localPort, remotePort uint16) ([]bpf.RawInstruction, error) {
	var filter []bpf.Instruction

	// 1. EtherType == 0x0800 (IPv4)
	filter = append(filter,
		bpf.LoadAbsolute{Off: 12, Size: 2},
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: 0x0800, SkipTrue: 1},
		bpf.RetConstant{Val: 0},
	)

	// 2. IPv4 Protocol == 6 (TCP)
	filter = append(filter,
		bpf.LoadAbsolute{Off: 23, Size: 1},
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: unix.IPPROTO_TCP, SkipTrue: 1},
		bpf.RetConstant{Val: 0},
	)

	// 3. Load IP Header Length (lower nibble of byte 14 shifted by 2) into index register X
	filter = append(filter,
		bpf.LoadMemShift{Off: 14},
	)

	// 4. TCP Destination Port == localPort
	filter = append(filter,
		bpf.LoadIndirect{Off: 16, Size: 2}, // Offset 14 + X + 2 = TCP Dst Port
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: uint32(localPort), SkipTrue: 1},
		bpf.RetConstant{Val: 0},
	)

	// 5. If Dialer with RemotePort, also match TCP Source Port == remotePort
	if role == RoleDialer && remotePort > 0 {
		filter = append(filter,
			bpf.LoadIndirect{Off: 14, Size: 2}, // Offset 14 + X + 0 = TCP Src Port
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: uint32(remotePort), SkipTrue: 1},
			bpf.RetConstant{Val: 0},
		)
	}

	// 6. Match: Pass up to 65535 bytes to userspace
	filter = append(filter,
		bpf.RetConstant{Val: 65535},
	)

	return bpf.Assemble(filter)
}

// --- Platform Network Discovery ---

func resolveInterface(requestedName string) (*net.Interface, net.IP, error) {
	if requestedName != "" {
		iface, err := net.InterfaceByName(requestedName)
		if err == nil {
			gwIP := findGatewayIPForInterface(iface.Name)
			return iface, gwIP, nil
		}
	}

	// Read /proc/net/route to find default gateway interface
	f, err := os.Open("/proc/net/route")
	if err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 8 && fields[1] == "00000000" { // Destination 0.0.0.0
				flags, _ := strconv.ParseUint(fields[3], 16, 32)
				if flags&0x2 != 0 { // RTF_GATEWAY flag
					ifaceName := fields[0]
					if iface, err := net.InterfaceByName(ifaceName); err == nil {
						gwHex := fields[2]
						gwIP := parseHexIP(gwHex)
						return iface, gwIP, nil
					}
				}
			}
		}
	}

	// Fallback: search interfaces for first active global unicast IPv4
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ip4 := ipNet.IP.To4(); ip4 != nil && ip4.IsGlobalUnicast() {
					return &iface, nil, nil
				}
			}
		}
	}

	return nil, nil, errors.New("no default network interface found")
}

func parseHexIP(hexStr string) net.IP {
	if len(hexStr) != 8 {
		return nil
	}
	b, err := hex.DecodeString(hexStr)
	if err != nil || len(b) != 4 {
		return nil
	}
	// Little endian in /proc/net/route
	return net.IPv4(b[3], b[2], b[1], b[0])
}

func findGatewayIPForInterface(ifaceName string) net.IP {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 && fields[0] == ifaceName && fields[1] == "00000000" {
			return parseHexIP(fields[2])
		}
	}
	return nil
}

func resolveGatewayMAC(ifaceName string, gwIP net.IP) (net.HardwareAddr, error) {
	if gwIP == nil {
		return nil, errors.New("gateway IP not specified")
	}

	f, err := os.Open("/proc/net/arp")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	gwStr := gwIP.String()
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 6 && fields[0] == gwStr && fields[5] == ifaceName {
			macStr := fields[3]
			if macStr != "00:00:00:00:00:00" {
				return net.ParseMAC(macStr)
			}
		}
	}
	return nil, fmt.Errorf("gateway %s MAC not found in ARP table", gwStr)
}

func probeGateway(gwIP net.IP) {
	if gwIP == nil {
		return
	}
	// Trigger kernel ARP request by sending an empty UDP probe
	conn, err := net.DialTimeout("udp", net.JoinHostPort(gwIP.String(), "80"), 100*time.Millisecond)
	if err == nil {
		_, _ = conn.Write([]byte{0})
		_ = conn.Close()
	}
	time.Sleep(50 * time.Millisecond)
}

func resolveLocalIPv4(iface *net.Interface, requestedAddr string) (net.IP, error) {
	if requestedAddr != "" {
		host, _, err := net.SplitHostPort(requestedAddr)
		if err == nil && host != "" && host != "0.0.0.0" {
			if ip := net.ParseIP(host).To4(); ip != nil {
				return ip, nil
			}
		}
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ip4 := ipNet.IP.To4(); ip4 != nil && !ip4.IsLoopback() {
				return ip4, nil
			}
		}
	}
	return nil, fmt.Errorf("no IPv4 address configured on interface %s", iface.Name)
}

func pickEphemeralPort() int {
	var b [2]byte
	_, _ = rand.Read(b[:])
	p := int(binary.BigEndian.Uint16(b[:]))
	// Clamp between 20000 and 55000 (Random Dynamic Port Policy)
	return 20000 + (p % 35000)
}
