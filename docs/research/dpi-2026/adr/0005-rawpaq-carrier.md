# ADR-0005: Integrate Paqet below Hawal session and mux

- Status: accepted for prototype
- Date: 2026-09-03
- Upstream baseline: Paqet `v1.0.0-alpha.21`, commit `4b81ce0`

## Decision

Hawal will not nest its session inside the complete Paqet application. A
`RawPaq` carrier instead uses this layering:

```text
Hawal mux and secure records
            |
       KCP byte stream
            |
 injected net.PacketConn
            |
 Linux PCAP raw packet backend
```

Paqet's smux, forwarding, SOCKS, CLI, configuration parser, and logger are not
part of the carrier. This prevents duplicate multiplexing, duplicate stream
buffers, and conflicting connection lifecycle logic.

## Dependency and licensing

The prototype pins `github.com/xtaci/kcp-go/v5` to `v5.6.72`, matching the
reviewed Paqet release. Paqet is MIT licensed. Its license and an upstream
commit manifest are stored under `third_party/paqet` even though this milestone
does not yet copy Paqet source files.

The KCP dependency raises the core build requirement from Go 1.20 to Go 1.24.
This affects the release builder, not installed nodes receiving a static
binary, and must be reflected in CI before release.

## Security boundary

KCP supplies reliability, ordering, retransmission, congestion behavior, and
optional FEC. It is not the Hawal peer-authentication boundary. Authentication,
forward secrecy, replay protection, and protected record metadata remain in
the Hawal secure-session layer.

An unauthenticated raw listener can still consume CPU and memory before the
inner handshake. The production PCAP backend therefore requires a bounded
pre-auth admission policy, rate limit, and cleanup tests before deployment.

## Rollout

The packet backend is injected. Rootless tests use an ordinary UDP PacketConn;
the Linux PCAP backend is feature-gated and will only be enabled on explicitly
selected experimental nodes. Existing Paqet tunnels and Hawal v1 remain
unchanged.

