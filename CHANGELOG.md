# 📋 Hawal Tunnel Changelog

All notable changes to the Hawal Tunnel project will be documented in this file.

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
