# Hawal Core v2.5.1 (Production-Ready Stealth Engine)

This directory contains the core implementation of **Hawal Core v2.5.1**, a high-performance, censorship-resistant multi-carrier stealth tunneling engine written in Go.

## Architectural Foundation & Core Features

- **Multi-Carrier Architecture:**
  - `tcp`: Baseline streaming carrier with bidirectional multiplexing.
  - `tls-http`: TLS 1.3 masquerade with customizable SNI and HTTP-like handshakes.
  - `rawpaq`: Native Linux Raw-TCP carrier with socket-level kernel cBPF/eBPF packet filtering and custom SYN/ACK packet generation, bypassing stateful middlebox tracking.
- **Cryptographic Security & Privacy:**
  - Ephemeral X25519 Diffie-Hellman key exchange providing **Perfect Forward Secrecy (PFS)**.
  - **ChaCha20-Poly1305 AEAD** frame encryption with encrypted record lengths, headers, and sequence counters.
  - Variable pseudo-random length masking on handshake flights to foil ML packet-length classifiers.
  - **Zero User Logging:** Hawal Core logs only structured system and lifecycle events; user connection IPs are never recorded.
- **Anti-Freeze & Resilient Transport (v2.5.1):**
  - **Bidirectional Heartbeats:** Periodic `TypePing` and `TypePong` (Record Type 9) exchange for true wire reachability verification.
  - **Dead-Link Auto-Detection:** Automatically detects silent packet absorption / ISP route dropping (>45s without inbound traffic) and cleanly triggers client-side reconnection.
  - **Socket Write Deadlines:** 10s deadlines on carrier writes and mux pumps to eliminate permanent KCP/socket send-window blocking.
- **High-Throughput Multiplexing:**
  - Weighted fair scheduling with strict byte-credit flow control windows.
  - Bounded ring buffers and non-blocking IO.

## Package Structure

```text
application ingress (SOCKS / TCP Forward)
        │
       mux  ◄──── policy / shaping
        │
      record (ChaCha20-Poly1305 + Length Masking)
        │
  secure session ──── Noise Handshake (PFS: X25519)
        │
      carrier (Rawpaq / TLS / TCP) ──── observe / controller
```

## Building & Testing

```bash
# Run all unit, integration, and fuzz tests
go test -v -race ./v2/...

# Build standalone binary
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.Version=2.5.1" -o ../bin/hawal-core ./v2/cmd/hawal-core
```
