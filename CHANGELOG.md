# 📋 Hawal Tunnel Changelog

All notable changes to the Hawal Tunnel project are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v2.3.0] - 2026-09-29

### 🎨 Enterprise UI & Anti-Slop Bento Architecture
- **Modern Bento Grid Layout:** Completely redesigned the web control panel following enterprise anti-slop aesthetics with sleek borders, micro-interactions, dark-mode gradients, and responsive mobile adaptation.
- **Server Resource Telemetry Gauges:** Integrated live CPU, RAM, and disk utilization circular/linear meters directly into the node cards with instant threshold alerts.
- **Real-Time Bandwidth & Historical Traffic Charts:** Added streaming throughput visualization (current upload/download speeds in MB/s) powered by Chart.js, coupled with persistent historical traffic accounting across daily and monthly windows.
- **SPA URL Routing & DOM Bug Fixes:** Implemented single-page URL hash and push-state routing (`#tunnels`, `#nodes`, `#traffic`, `#logs`, `#settings`) and fixed HTML DOM table nesting inconsistencies.

### ⚡ Hawal Core v2: Reverse Tunneling & TLS Carrier
- **Outbound Reverse Tunnel Topology:** Configured Hawal v2 outbound architecture where the foreign server acts as the listening acceptor (`mode=server`) and Iran acts as the initiating dialer (`mode=client`), bypassing strict ingress filtering on domestic ISPs.
- **TLS 1.3 Carrier Enforcement:** Enabled standard TLS carrier camouflage with customizable SNI disguise, eliminating raw packet fingerprints and providing authenticated stream multiplexing.
- **Accurate Monitoring Accounting:** Aligned iptables and kernel socket counters so that Hawal v2 multiplexed streams accurately report ingress and egress consumption on both local and foreign nodes.

### 📚 Documentation & Guides Overhaul
- **Comprehensive Installation Guide:** Rewrote both Persian (`README_FA.md`) and English (`README.md`) documentation with structured tables of contents, technical requirements, firewall considerations, and zero-touch 1-line installation scripts.
- **Domain & SSL Reverse Proxy Manuals:** Added production Nginx and Caddy reverse proxy deployment blueprints with automatic Let's Encrypt SSL renewal for master control panels.

### 🛡️ CI/CD, Security & Community Infrastructure
- **Automated Multi-Arch GitHub CI:** Added GitHub Actions pipeline for automated unit testing (`core/v2`) and cross-compilation of standalone `hawal-core` binaries for Linux `amd64`, `arm64`, `386`, and `armv7`.
- **Security Scanning Pipeline:** Integrated GitHub CodeQL analysis (Go & Python) and `gosec` static analysis triggers on pull requests and branch pushes.
- **Automated GitHub Releases:** Implemented automated release publishing on Git tag pushes with SHA256 checksum generation.
- **Community Standards:** Added interactive GitHub issue templates (`bug_report.yml`, `feature_request.yml`, `dpi_network_report.yml`), a structured PR template, `SECURITY.md`, and `CONTRIBUTING.md`.

---

## [v2.2.0] - 2026-09-29

### 🚀 Unified CLI & Diagnostics (`hawal`)
- **Interactive Terminal Dashboard:** Launched `hawal` CLI with zero dependencies, featuring a sleek English interactive menu for both Master Panels and Foreign Node Agents.
- **Multi-Tunnel Registry (`hawal tunnels`):** Added comprehensive table inspection for all registered tunnels, core protocols (GOST, Paqet, Hawal Stealth), port mappings (`Listen -> Target`), and destination nodes.
- **Live Latency & Health Benchmark (`hawal ping`):** Real-time measurement of REX control sockets, individual tunnel core TLS 1.3 handshakes, and local upstream service responsiveness (e.g. Xray/3X-UI on ports 2087 and 2096).
- **Log Streaming (`hawal logs`):** Direct log streaming with support for filtering individual tunnel IDs (e.g. `hawal logs tun_742c465f`).
- **Comprehensive Help & Operator Guides:** Integrated `hawal --help` and added `docs/AGENT_GUIDE.md` (and Persian edition `docs/AGENT_GUIDE_FA.md`) to guide node operators without confusion.

### 🛡️ Reliability & Process Lifecycle
- **Process Group Isolation (`os.killpg`):** Implemented `start_new_session=True` for tunnel child processes across `agent/agent.py` and `app/static/js/agent.py`, ensuring all sub-processes are terminated cleanly without leaving orphan port-binding processes.
- **Port Conflict Sanitizer (`_free_ports`):** Added automated socket release mechanism before binding new tunnel listeners.
- **Subscription Port Dual-Mapping:** Supported simultaneous forwarding of both port `443` and `2096` on Iran master nodes to eliminate subscription timeout issues with client applications (v2rayNG, Streisand, Shadowrocket).
- **Async Socket Drain:** Added `await writer.drain()` to `server.py` static and template file handlers to prevent connection resets when downloading large binaries.

### ⚡ Core & Network Engine
- **Hawal Stealth Core v2:** Upgraded `core/v2` with DPI-resistant dynamic packet framing, TLS 1.3 relay, and multi-stream session pooling.
- **Repository Branding:** Refreshed project namespace and GitHub sync endpoints to `dalroot/hawal`.

---
