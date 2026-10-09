# 📋 Hawal Tunnel Changelog

All notable changes, core capabilities, architecture evolutions, and security hardening for the **Hawal Tunnel (هه‌واڵ)** project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v2.5.3] - 2026-10-09 (Current Stable Milestone)

> **Milestone Focus:** Immediate Keepalive/Handshake Delivery, Idle Reconnect Elimination, Synchronized DeadLink Timeout, Default MTU Hardening (1150), and GitOps Process Cycling.

### ⚡ Hawal Core v2.5.3: Zero-Buffering & Idle Stability
- **Zero-Latency Control Delivery:** Set `WriteDelay = false` and `ACKNoDelay = true` across all Rawpaq KCP modes (`ModeNormal`, `ModeFast`, `ModeFast2`, `ModeFast3`). Control frames (Ping/Pong, Stream Open) and initial TLS handshakes are flushed immediately onto the raw socket wire without waiting for outbound buffer aggregation.
- **Synchronized DeadLink Timeout (75s):** Aligned default `DeadLinkTimeout` in `NewSession` to 75 seconds to match `pingLoop` recovery thresholds, preventing spurious 45-second teardowns during idle periods while maintaining rapid dead-link self-healing.
- **Evidence-Backed Default MTU (1150):** Updated `DefaultConfig` MTU from 1350 to 1150 per Evidence Ledger RAW-06, ensuring raw TCP frames including KCP and Noise framing stay well below Iranian carrier/TIC fragmentation thresholds.
- **GitOps Tunnel Recycling:** Enhanced `scripts/update-node.sh` to cycle active core tunnel processes upon binary replacement, allowing nodes to transition instantly to new releases.

---

## [v2.5.2] - 2026-10-09

> **Milestone Focus:** High-Concurrency Mux Queue Expansion, Safe Process Isolation, MTU/Channel Size Persistence & Dynamic Tuning, and GitOps Automated Deployment.

### ⚡ Hawal Core v2.5.2: Concurrency & Transport Resilience
- **Mux Control Queue Hardening:** Increased `MaxControlFrames` in scheduler from 128 to 2048 to prevent control frame buffer exhaustion and session dropouts under sudden handshake surges (Fixes #6).
- **Atomic Session Eviction:** Implemented clean session eviction (`Evict()`, `IsEvicted()`) and increased `DeadLinkTimeout` to 75 seconds, eliminating false-positive link teardowns during transit jitter (Fixes #6).
- **Deterministic Rawpaq Binding:** Added `SourcePort` configuration to `rawpaq` carrier for symmetric iptables conntrack bypass (`NOTRACK` and `DROP RST`).

### 🛡️ Agent & Panel Hardening
- **Granular Process Tracking:** Replaced blind `pkill -9` with per-tunnel state-file PID tracking (`SIGTERM`), preventing agent restarts from killing adjacent production tunnels (Fixes #6).
- **MTU / Channel Size Persistence:** Full-stack persistence for `channel_size` across Web UI, backend API, and SQLite database; enforced safe 1150 default MTU to eliminate Iranian ISP/DPI KCP packet fragmentation (Fixes #7).
- **GitOps Node Deployment:** Added standardized `scripts/update-node.sh` with cryptographic SHA256 checksum verification and automated GitHub Actions release packaging.

---

## [v2.5.1] - 2026-10-03

> **Milestone Focus:** Hawal Core v2.5.1 Resilient Rawpaq Carrier, Morning Silent-Drop & Freeze Fix, Dead-Link Auto-Recovery, Socket Write Deadlines, and In-Depth DPI / Raw-TCP Research Documentation.

### ⚡ Hawal Core v2.5.1: Anti-Freeze & Resilient Transport Engine
- **Bidirectional Keepalive (`TypePong`):** Introduced first-class `TypePong` (Type = 9) control record with codec verification, allowing real-time round-trip latency tracking and heartbeat validation.
- **Dead-Link Auto-Detection & Self-Healing:** Implemented automatic stale link detection (`lastInboundActivity`). If an active tunnel experiences silent packet absorption or ISP routing blackholing (>45 seconds without inbound traffic/pong), the session cleanly tears down and automatically triggers an instant client reconnection.
- **Socket Write Deadlines & Deadlock Elimination:** Added proactive 10-second `SetWriteDeadline` on the carrier layer and outbound multiplexer pumps, completely eliminating infinite KCP send-window hangs and morning freeze conditions.
- **Native Rawpaq Carrier with Kernel BPF:** Hardened the high-performance Linux Raw-TCP carrier utilizing kernel cBPF socket filters to evade stateful DPI connection tracking without full 3-way handshake fingerprints.
- **Privacy-Preserving Telemetry:** Enforced strict zero-user-connection logging across Hawal Core, logging only system lifecycle metrics without storing or leaking end-user IP addresses.

### 📚 Comprehensive DPI & Censorship Research Documentation
- **Raw-TCP & Paqet Reverse Engineering Study:** Added extensive technical documentation (`docs/research/dpi-2026/raw-tcp-and-paqet-mechanisms.md`) breaking down raw socket packet manipulation, eBPF capture rules, syn-cookie interactions, and DPI bypass mechanisms.
- **Evidence Ledger & Source Catalog:** Updated experimental findings, Black Hat/academic references, and 2026 censorship countermeasures in `docs/research/dpi-2026/`.

---

## [v2.4.0] - 2026-10-01

### 🎨 Ant Design Inspired Web Console & UX
- **Ant Design Enterprise Architecture:** Complete user interface overhaul inspired by the clean, ergonomic Ant Design system with refined typography, high-contrast dark/light themes, responsive navigation drawer, and modular cards.
- **Live Terminal Logs Viewer (`>_`):** Integrated in-browser terminal modal streaming live Paqet Wire transport logs, connection handshakes, and carrier events with auto-scroll and line pruning.
- **Interactive Port Forwarding Chips:** Re-engineered port chip management with event delegation, click-to-remove interactions, red-tinted hover feedback, and two-way synchronization with Quick Port Tags.
- **Extended Session Persistence (Remember-Me):** Introduced 90-day persistent session tokens stored in secure HTTP-only cookies and localStorage, eliminating frequent session timeouts.
- **Dynamic KPI Traffic Telemetry:** Real-time calculation of overall network bandwidth consumption (Total Sent / Received / Cumulative usage) derived directly from live carrier sessions.
- **Dynamic GeoIP & Node Flags:** Automatic destination flag rendering (🇩🇪 Germany, 🇳🇱 Netherlands, etc.) and real-time jitter/latency metrics in the wire tunnel table.

### 🛡️ Core Security & Fleet Port Safeguards
- **Critical Port Collision Guard:** Implemented strict backend and UI validation reserving essential host management ports (`22` SSH, `9090` Panel, `7444` Agent Sync) against accidental port forwarding.
- **Automated Fleet Reloads (Restart Nonces):** Integrated incremental configuration nonces triggering zero-downtime hot reloads across distributed node agents upon tunnel updates.
- **Multi-Node Wire Carrier Hardening:** Validated concurrent operation across Iran edge, Netherlands exit, and Germany exit nodes with 8 parallel multiplexed streams under Paqet Wire (KCP and RawTCP).

---

## [v2.3.0] - 2026-09-29

> **Milestone Focus:** Enterprise Bento Grid Dashboard, Real-time Traffic Telemetry, Hawal Core v2 Outbound TLS Reverse Tunneling, Multi-Arch CI/CD, and Automated CodeQL Security Scanning.

### 🎨 Enterprise UI & Anti-Slop Bento Architecture
- **Modern Bento Grid Layout:** Completely overhauled the web management dashboard following strict anti-slop design principles with modular Bento grid containers, refined dark-mode gradients, smooth micro-interactions, and 100% responsive layout on desktop and mobile.
- **Live Node Resource Meters:** Real-time visual gauges (circular and linear SVG meters) monitoring CPU load, RAM utilization, and disk space for both master and foreign nodes with threshold indicators (Normal / Warning / Critical).
- **Streaming Bandwidth Telemetry:** High-frequency real-time throughput meters displaying instantaneous upload and download transfer speeds (in MB/s and KB/s) powered by reactive WebSocket sync.
- **Interactive Historical Traffic Charts:** Chart.js integration visualizing 24-hour and 30-day cumulative bandwidth consumption across nodes and individual tunnels.
- **SPA URL Routing & State Persistence:** Hash-based single-page application navigation (`#tunnels`, `#nodes`, `#traffic`, `#logs`, `#settings`) allowing direct linking and browser back/forward history retention.
- **DOM Hierarchy Harmonization:** Resolved nested table markup and flexbox alignment inconsistencies across all tunnel inspection modals.

### ⚡ Hawal Core v2 Engine: Reverse Outbound & TLS Carrier
- **Outbound Reverse Tunnel Topology:** Established outbound client-to-server reverse architecture (`Iran = Dialer/Client`, `Abroad = Acceptor/Server`), completely evading strict domestic ISP ingress port filtering and NAT traversal issues.
- **Cryptographic Perfect Forward Secrecy (PFS):** Replaced static token hashing with ephemeral X25519 Diffie-Hellman key agreement, HMAC-SHA256 mutual authentication, and ChaCha20-Poly1305 AEAD stream encryption.
- **Standard TLS 1.3 Carrier Obfuscation:** Camouflaged tunnel transport inside authentic TLS handshakes with customizable Server Name Indication (SNI), eliminating plaintext packet signatures (`HWL1` magic headers removed entirely).
- **Encrypted Frame Metadata:** All lengths, types, stream IDs, and sequence counters are protected by authenticated ciphertexts, preventing statistical payload classification.
- **Dynamic Initial Padding:** Injected variable pseudo-random padding (16 to 48 bytes) into initial handshake frames to mitigate machine-learning packet-length fingerprinting.
- **Kernel-Level Traffic Accounting:** Synchronized iptables socket accounting so Hawal v2 multiplexed streams accurately report directional bytes on both local and remote nodes.

### 🛡️ Automated CI/CD, Security Scanning & Multi-Arch Releases
- **Multi-Architecture Matrix Compilation:** Automated GitHub Actions build pipeline cross-compiling standalone `hawal-core` binaries across four Linux target architectures:
  - `linux/amd64` (Standard 64-bit x86 servers)
  - `linux/arm64` (ARMv8 / AArch64 cloud instances & Raspberry Pi 4/5)
  - `linux/386` (Legacy 32-bit x86 environments)
  - `linux/armv7` (32-bit ARM hardware)
- **32-Bit Integer Overflow Resolution:** Fixed cross-compilation arithmetic bounds in `record/codec.go` to ensure safe operation on 32-bit kernels.
- **Static Code Analysis (CodeQL & Gosec):** Integrated automated GitHub CodeQL scanning for Go and Python alongside `gosec` to detect potential memory safety, cryptographic, and dependency vulnerabilities on every push and pull request.
- **Automated GitHub Release Pipeline:** One-click automated release generation generating release notes, attaching compiled binaries, and publishing cryptographic SHA-256 verification checksums (`checksums.txt`).
- **Community Governance & Rich Templates:** Established interactive GitHub issue forms (`bug_report.yml`, `feature_request.yml`, `dpi_network_report.yml`), `PULL_REQUEST_TEMPLATE.md`, `SECURITY.md`, and `CONTRIBUTING.md`.

### 📚 Documentation & Deployment Manuals
- **Zero-Touch Quickstart:** Overhauled Persian (`README_FA.md`) and English (`README.md`) documentation with structured table of contents, prerequisites, port firewall rules, and copy-paste 1-line installation scripts.
- **Reverse Proxy & Auto-SSL Blueprints:** Published comprehensive deployment guides for production Nginx and Caddy reverse proxies with automated Let's Encrypt SSL certificate issuance.

---

## [v2.2.0] - 2026-09-29

### 🚀 Unified CLI & Diagnostics Tool (`hawal`)
- **Interactive Terminal Menu:** Launched zero-dependency Python-based `hawal` command-line utility with intuitive English menu for node operators.
- **Multi-Tunnel Registry (`hawal tunnels`):** Real-time table viewing active tunnels, core protocols (GOST, Paqet, Hawal Stealth, Backhaul), listening ports, and target destination IP addresses.
- **Latency & Upstream Health Check (`hawal ping`):** Real-time round-trip latency measurements testing panel control sockets, individual tunnel core handshakes, and upstream services (e.g. Xray / 3X-UI on ports 2087 and 2096).
- **Log Streaming Engine (`hawal logs`):** Real-time terminal log viewer with regex filtering by tunnel ID (e.g. `hawal logs tun_742c465f`).
- **Node Operator Handbook:** Added `docs/AGENT_GUIDE.md` and `docs/AGENT_GUIDE_FA.md` detailing operational commands and troubleshooting routines.

### 🛡️ Process Lifecycle & Systemd Hardening
- **Process Group Isolation (`os.killpg`):** Implemented `start_new_session=True` for tunnel child processes across `agent/agent.py` and `app/static/js/agent.py`, guaranteeing all sub-processes terminate cleanly without leaving orphan port-binding daemons.
- **Port Conflict Sanitizer (`_free_ports`):** Automated pre-binding socket release to terminate stale processes occupying requested ports before initializing new tunnels.
- **Dual Subscription Mapping:** Added simultaneous binding for ports `443` and `2096` on Iran nodes to prevent subscription retrieval timeouts across iOS and Android client applications (v2rayNG, Streisand, Shadowrocket).
- **Non-Blocking Socket Drain:** Added `await writer.drain()` to `server.py` static and template handlers to prevent socket reset errors when transferring large agent binaries.

---

## [v2.1.0] - 2026-09-28

### 👻 GOST v3 Relay Core Integration
- **Full TCP & UDP Multi-Protocol Relay:** Integrated GOST v3 relay engine allowing seamless tunneling of UDP-dependent modern protocols including Hysteria2, TUIC, and HTTP/3 QUIC.
- **Encrypted Relay Carrier:** Supported encrypted relay hops between domestic master and foreign agent nodes.

### 🛡️ Paqet Kernel-Level Raw TCP / KCP Engine
- **Raw TCP Packet Injection:** Implemented Paqet core operating with KCP congestion control over raw TCP packets to maintain connection stability over high packet-loss links (10% to 40% loss tolerance).
- **Direct Kernel Raw-Table Accounting:** Built custom iptables parsing engine that reads raw socket counter statistics directly from Linux kernel `raw-table`, accurately tracking actual wire consumption including protocol overhead.
- **TCP-Only Forwarding Sanitization:** Hardened Paqet forwarding rules to restrict binding strictly to valid TCP ingress streams.

### 🔐 Authentication & Session Security
- **3X-UI Inspired Login Interface:** Designed clean, secure authentication portal with first-time administrator initialization and built-in cryptographically secure password generator.
- **Session Expiration & Rate-Limiting:** Added timed session token invalidation and sanitized panel HTTP timeouts to mitigate credential brute-forcing.

---

## [v2.0.0] - 2026-09-25 (Initial Multi-Core Foundation)

### ⚡ Core Architecture & Panel Master
- **Centralized Web Control Panel:** Light, standalone Python asynchronous web server running without heavyweight external databases (embedded SQLite engine).
- **Zero-Touch Foreign Node Enrollment:** One-line `curl` installer provisioning foreign server agents with automatic authentication token generation.
- **Multi-Core Tunnel Engine:** Unified orchestration layer supporting:
  - **Hawal Stealth Core v1:** High-performance Go-based TCP multiplexing tunnel with `TCP_NODELAY`.
  - **Backhaul Core:** Concurrent multiplexing tunnel over WebSocket, TCP, and TLS transports.
- **Docker Compose & Systemd Native Support:** Full compatibility with systemd daemon management or containerized deployment via `docker-compose.yml`.
- **System Resource Telemetry:** Live heartbeat telemetry transmitting CPU and memory metrics from connected nodes to master dashboard.
