package rawpaq

import (
	"bytes"
	"net"
	"testing"
)

func TestIPChecksum(t *testing.T) {
	// Sample IPv4 header: 45 00 00 3c 1c 46 40 00 40 06 00 00 ac 10 0a 63 ac 10 0a 0c
	hdr := []byte{
		0x45, 0x00, 0x00, 0x3c,
		0x1c, 0x46, 0x40, 0x00,
		0x40, 0x06, 0x00, 0x00, // Checksum field is 0x00, 0x00
		0xac, 0x10, 0x0a, 0x63, // 172.16.10.99
		0xac, 0x10, 0x0a, 0x0c, // 172.16.10.12
	}
	csum := ipChecksum(hdr)
	if csum == 0 {
		t.Fatalf("expected non-zero IP checksum")
	}

	// Place calculated checksum in header and recompute; RFC 1071 states result should be 0 (or ~0)
	hdr[10] = byte(csum >> 8)
	hdr[11] = byte(csum & 0xff)
	recomputed := ipChecksum(hdr)
	if recomputed != 0 {
		t.Fatalf("expected 0 for valid checksum verification, got 0x%04x", recomputed)
	}
}

func TestTCPChecksum(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.100").To4()
	dstIP := net.ParseIP("10.0.0.1").To4()
	tcpHdr := []byte{
		0x1f, 0x90, 0x1f, 0x90, // Sport 8080, Dport 8080
		0x00, 0x00, 0x00, 0x01, // Seq 1
		0x00, 0x00, 0x00, 0x02, // Ack 2
		0x50, 0x18, 0xfa, 0xf0, // Offset 5, Flags 0x18 (PA), Win 64240
		0x00, 0x00, 0x00, 0x00, // Checksum 0, Urg 0
	}
	payload := []byte("hello-hawal-v2-raw-packet")

	csum := tcpChecksum(srcIP, dstIP, tcpHdr, payload)
	if csum == 0 {
		t.Fatalf("expected non-zero TCP checksum")
	}

	// Verify non-zero and stable
	csum2 := tcpChecksum(srcIP, dstIP, tcpHdr, payload)
	if csum != csum2 {
		t.Fatalf("checksum is not deterministic: %04x vs %04x", csum, csum2)
	}
}

func TestBuildAndParseRawEthernetFrame(t *testing.T) {
	srcMAC, _ := net.ParseMAC("00:11:22:33:44:55")
	dstMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	srcIP := net.ParseIP("192.168.1.50").To4()
	dstIP := net.ParseIP("1.2.3.4").To4()
	srcPort := uint16(54237)
	dstPort := uint16(443)
	seq := uint32(12345678)
	ack := uint32(87654321)
	flags := uint8(0x18) // PA
	payload := []byte("secret-record-payload-1234")

	frame := buildRawEthernetFrame(srcMAC, dstMAC, srcIP, dstIP, srcPort, dstPort, seq, ack, flags, payload, 46)

	parsedSrcIP, parsedDstIP, parsedSrcPort, parsedDstPort, parsedSeq, parsedAck, parsedFlags, parsedPayload, err := parseRawEthernetFrame(frame)
	if err != nil {
		t.Fatalf("parse frame error: %v", err)
	}

	if !parsedSrcIP.Equal(srcIP) {
		t.Fatalf("src IP mismatch: got %v want %v", parsedSrcIP, srcIP)
	}
	if !parsedDstIP.Equal(dstIP) {
		t.Fatalf("dst IP mismatch: got %v want %v", parsedDstIP, dstIP)
	}
	if parsedSrcPort != srcPort {
		t.Fatalf("src port mismatch: got %d want %d", parsedSrcPort, srcPort)
	}
	if parsedDstPort != dstPort {
		t.Fatalf("dst port mismatch: got %d want %d", parsedDstPort, dstPort)
	}
	if parsedSeq != seq {
		t.Fatalf("seq mismatch: got %d want %d", parsedSeq, seq)
	}
	if parsedAck != ack {
		t.Fatalf("ack mismatch: got %d want %d", parsedAck, ack)
	}
	if parsedFlags != flags {
		t.Fatalf("flags mismatch: got %x want %x", parsedFlags, flags)
	}
	if !bytes.Equal(parsedPayload, payload) {
		t.Fatalf("payload mismatch: got %q want %q", parsedPayload, payload)
	}
}

func TestBPFFilterAssembly(t *testing.T) {
	prog, err := buildBPFFilter(RoleListener, 54237, 0)
	if err != nil {
		t.Fatalf("BPF assemble for listener failed: %v", err)
	}
	if len(prog) == 0 {
		t.Fatalf("assembled BPF program is empty")
	}

	progClient, err := buildBPFFilter(RoleDialer, 20500, 54237)
	if err != nil {
		t.Fatalf("BPF assemble for dialer failed: %v", err)
	}
	if len(progClient) <= len(prog) {
		t.Fatalf("expected dialer BPF filter to have additional instructions for remote port check")
	}
}
