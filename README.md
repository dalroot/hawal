<p align="center">
  <a href="https://github.com/dalroot/hawal">
    <img src="docs/assets/hawal_logo.png" alt="Hawal Logo" width="130" style="border-radius: 16px; margin-bottom: 8px;" />
  </a>
</p>

<h1 align="center">⚡ Hawal — Multi-Carrier Stealth Tunnel Management Platform</h1>

<p align="center">
  <em>A high-performance, resilient, and lightweight control plane for deploying, monitoring, and synchronizing multi-carrier stealth tunnels between edge nodes under hostile network environments.</em>
</p>

<p align="center">
  <a href="https://github.com/dalroot/hawal/releases"><img src="https://img.shields.io/github/v/release/dalroot/hawal?color=0284c7&logo=github&label=Release" alt="Release" /></a>
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white" alt="Go Version" /></a>
  <a href="https://www.python.org/"><img src="https://img.shields.io/badge/Python-3.10%2B-3776ab?logo=python&logoColor=white" alt="Python Version" /></a>
  <a href="https://t.me/dallrep"><img src="https://img.shields.io/badge/Telegram-Community%20%26%20Channel-229ED9?logo=telegram&logoColor=white" alt="Telegram" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-2563eb.svg" alt="License" /></a>
  <a href="docker-compose.yml"><img src="https://img.shields.io/badge/Docker-Ready-2496ed?logo=docker&logoColor=white" alt="Docker" /></a>
</p>

<p align="center">
  <b>Hawal</b> (هه‌واڵ, Kurdish for <i>“friend, companion, or messenger”</i>) maintains all tunnel configurations in a single centralized panel and synchronizes them continuously to lightweight agents on your edge servers.
</p>

<p align="center">
  🌐 <b>Languages:</b> <b>English</b> | 🇮🇷 <a href="README_FA.md">فارسی (Persian)</a> | 💬 <b>Telegram Community:</b> <a href="https://t.me/dallrep">t.me/dallrep</a>
</p>

---

## 📑 Table of Contents

- [✨ Highlights & Features](#-highlights--features)
- [🏛️ Architecture Overview](#️-architecture-overview)
- [🎛️ Tunnel Cores & Selection Guide](#️-tunnel-cores--selection-guide)
  - [⚡ Hawal Core (v2.5.1) Technical Deep-Dive](#-hawal-core-v251-technical-deep-dive)
- [📋 System & Network Prerequisites](#-system--network-prerequisites)
  - [1. Server Requirements](#1-server-requirements)
  - [2. Domain, Subdomain & DNS Configuration](#2-domain-subdomain--dns-configuration)
  - [3. Port Management & Firewall Rules](#3-port-management--firewall-rules)
- [⚡ Quick & Easy Installation (1-Line Installer)](#-quick--easy-installation-1-line-installer)
  - [Step 1: Install the Master Panel](#step-1-install-the-master-panel)
  - [Step 2: Enroll Nodes (Iran & Foreign Servers)](#step-2-enroll-nodes-iran--foreign-servers)
  - [Step 3: Create Your First Tunnel](#step-3-create-your-first-tunnel)
- [🐳 Alternative Deployment (Docker Compose)](#-alternative-deployment-docker-compose)
- [🔒 Securing the Panel with SSL & Reverse Proxy](#-securing-the-panel-with-ssl--reverse-proxy)
- [🛡️ Raw-TCP & Kernel Wire Accounting](#️-raw-tcp--kernel-wire-accounting)
- [🛠️ Operations, Maintenance & Troubleshooting](#️-operations-maintenance--troubleshooting)
- [💬 Community & Support](#-community--support)
- [📚 Documentation & Wiki](#-documentation--wiki)
- [🤝 Contributing & License](#-contributing--license)

---

## ✨ Highlights & Features

- **Ant Design Enterprise Web Console:** Manage multi-port tunnels from a responsive, zero-refresh dashboard with dark/light themes and live bandwidth monitors.
- **Zero-Touch Agent Enrollment:** Deploy node agents with a single pre-authenticated `curl` command without transferring manual config files or opening inbound SSH.
- **Four Integrated Production Cores:**
  - ⚡ **Hawal Core (v2.5.1):** High-performance Go systems engine featuring native Linux Raw-TCP (`rawpaq`), socket-level kernel cBPF/eBPF filtering, ephemeral Noise X25519 PFS, ChaCha20-Poly1305 AEAD, bidirectional `TypePong` keepalives, and dead-link autodetection to prevent silent-drop morning freezing.
  - 🚀 **Backhaul:** High-concurrency multiplexed tunnels over WebSocket, TCP, or TLS.
  - 🛡️ **Paqet:** Raw TCP packet manipulation combined with KCP congestion control for unstable links.
  - 👻 **GOST v3:** Authenticated relay forwarding supporting TLS, WebSocket, KCP, and QUIC (Hysteria2 compatible).
- **Privacy First (Zero User Logging):** Hawal Core does not log, write, or leak end-user connection IPs—only structured engine lifecycle events are recorded.
- **Accurate Wire Traffic Accounting:** Exact destination raw-table accounting for Paqet and Rawpaq (measuring actual wire bytes including retransmissions and KCP framing).
- **Real-Time Observability:** Node CPU/RAM meters, active tunnel latency ping, packet-loss tracking, and live connection status.
- **Embedded Database:** Standalone SQLite database with zero external dependencies (no Redis or PostgreSQL needed).
- **Systemd & Docker Ready:** Full lifecycle management via native systemd services or Docker Compose.

---

## 🏛️ Architecture Overview

In Hawal's architecture, the management control plane is completely separated from the high-throughput encrypted data plane:

```mermaid
graph TD
    subgraph ManagementPlane["🖥️ Control Plane (Web Dashboard & State Orchestration)"]
        Admin["👤 Admin / Network Operator"]
        Panel["⚡ Hawal Master Panel<br/><b>FastAPI + SQLite + Web UI</b><br/><i>Default Port :9090 or :443 HTTPS</i>"]
        Admin -->|Browser HTTPS| Panel
    end

    subgraph IranEdge["🇮🇷 Iran Edge Node (Client / Dialer)"]
        AgentIran["🤖 Hawal Node Agent<br/><i>/opt/hawal/agent.py</i>"]
        IngressPort["🚪 Client Ingress Port<br/><i>e.g. :443, :8443 (TCP/UDP)</i>"]
        HawalDialer["⚡ Hawal Core Engine (v2.5.1)<br/><i>Raw-TCP (cBPF) / TLS 1.3 / TCP Mux</i>"]
        ClientUser["👥 End-Users & VPN Clients<br/><i>(Xray / V2Ray / Sing-box)</i>"]
        
        ClientUser -->|User Traffic| IngressPort
        IngressPort -->|Forward Stream| HawalDialer
    end

    subgraph ForeignEdge["🌍 Foreign Exit Node (Server / Acceptor)"]
        AgentForeign["🤖 Hawal Node Agent<br/><i>/opt/hawal/agent.py</i>"]
        HawalAcceptor["⚡ Hawal Core Server (v2.5.1)<br/><i>Kernel BPF / Raw Socket Listener</i>"]
        CorePort["🔒 Stealth Core Port<br/><i>e.g. :3107 / :9999</i>"]
        TargetApp["🎯 Target Service<br/><i>Xray / 3X-UI / GOST (127.0.0.1:443)</i>"]

        CorePort -->|Demux Payloads| HawalAcceptor
        HawalAcceptor -->|Local Loopback| TargetApp
    end

    Panel <-.->|Sync Config & Telemetry (3s)<br/>TLS / Ephemeral Bearer Token| AgentIran
    Panel <-.->|Sync Config & Telemetry (3s)<br/>TLS / Ephemeral Bearer Token| AgentForeign
    HawalDialer ==>|Reverse Outbound Stealth Transport<br/>PFS: Noise X25519 + ChaCha20-Poly1305<br/>Anti-Freeze Heartbeat & Zero-User Logging| CorePort

    classDef panelStyle fill:#0f172a,stroke:#0284c7,stroke-width:2px,color:#f8fafc;
    classDef iranStyle fill:#022c22,stroke:#10b981,stroke-width:2px,color:#f8fafc;
    classDef foreignStyle fill:#1e1b4b,stroke:#8b5cf6,stroke-width:2px,color:#f8fafc;
    
    class Panel panelStyle;
    class HawalDialer,IngressPort,AgentIran,ClientUser iranStyle;
    class HawalAcceptor,CorePort,AgentForeign,TargetApp foreignStyle;
```

```text
╭────────────────────────────────────────────────────────────────────────────────────────╮
│                               Hawal Topology Overview                                  │
╰────────────────────────────────────────────────────────────────────────────────────────╯
       Browser / Admin
              │ (HTTPS)
              ▼
 ┌──────────────────────────┐        Configuration Sync & Heartbeat       ┌──────────────────────────┐
 │ Hawal Master Panel       │◄───────────────────────────────────────────►│ Foreign Edge Node       │
 │ (Web UI, SQLite DB)      │                  (3–5s)                     │ (/opt/hawal/agent.py)    │
 └──────────────────────────┘                                             └────────────┬─────────────┘
              ▲                                                                        │
              │ Sync & Telemetry                                                       │
 ┌────────────┴─────────────┐                                                          │
 │ Iran Edge Node           │              Reverse Outbound Stealth Carrier            │
 │ (/opt/hawal/agent.py)    │══════════════════════════════════════════════════════════╪═════════════════╗
 └────────────┬─────────────┘   (Hawal Core v2.5.1 / Paqet / Backhaul / GOST)          │                 ║
              │                                                                        ▼                 ║
              ▼                                                             Target Service (Xray/3X-UI)  ║
      Client Entry Port                                                      (Listening on 127.0.0.1)    ║
     (e.g., :443, :8443)                                                                                 ║
```

---

## 🎛️ Tunnel Cores & Selection Guide

| Core | Language & Architecture | Supported Transports | Cryptography & Privacy | Operational Scenario |
|---|---|---|---|---|
| ⚡ **Hawal Core (v2.5.1)** | **Go Native** (Standalone) | `rawpaq` (Raw-TCP + cBPF), `tls-http`, `tcp` | Ephemeral Noise X25519 (PFS) + ChaCha20-Poly1305 AEAD + **Zero User Logging** | **Primary & Recommended**; engineered specifically for hostile DPI environments with built-in silent-drop anti-freeze and deadlink auto-recovery. |
| 🛡️ **Paqet Wire** | **C & Go** | Raw TCP packets via KCP congestion control | Symmetric static key | Severe packet-loss routes with volatile latency. Requires `root` and isolated core port. |
| 🚀 **Backhaul** | **Go** | `ws`, `tcp`, `tcpmux`, `tls` | TLS 1.3 / Cleartext | High-throughput web multiplexing when standard TCP streams are unimpeded. |
| 👻 **GOST v3** | **Go** | `tls`, `ws`, `kcp`, `quic` | Multi-protocol customizable | Relaying UDP traffic, gaming protocols, or Hysteria2 / QUIC tunnels. |

### Architectural Decision Strategy:
1. **Hostile Censorship & Default Deployment:** Choose **Hawal Core (v2.5.1)**. With its native `rawpaq` carrier, Linux kernel cBPF socket filtering, bidirectional `TypePong` verification, and active dead-link detection, it automatically recovers from silent routing drops without manual intervention.
2. **High Bandwidth & Web Camouflage:** Use **Backhaul** over WebSocket (`ws`) or `tcpmux` if your ISP does not inject TCP RST or apply severe packet-drop rate limiting.
3. **Extreme Jitter & Lossy Links:** Choose **Paqet Wire** if your link suffers >15% packet loss and you require raw packet injection. Ensure core port is dedicated and separate from entry ports.
4. **QUIC / UDP Gaming & Streaming:** Deploy **GOST v3** when forwarding UDP payloads or Hysteria2 streams alongside standard TCP.

---

### ⚡ Hawal Core (v2.5.1) Technical Deep-Dive

Hawal Core is built from the ground up to counter modern stateful Deep Packet Inspection (DPI) and silent blackholing:

1. **Native Raw-TCP with Kernel BPF (`rawpaq`):**
   - Synthesizes raw TCP packets at Layer 4, bypassing Linux TCP stack state machine handshakes.
   - Attaches classical BPF (cBPF) filter programs to the raw socket, discarding kernel-level noise and RST packets.
2. **Anti-Freeze & Dead-Link Self-Healing:**
   - **Bidirectional Heartbeat (`TypePong = 9`):** True wire round-trip verification between dialer and acceptor.
   - **Silent Drop Auto-Recovery:** If routing blackholing occurs (>45 seconds without inbound traffic/pong), the stale session is cleanly torn down and instantly re-established.
   - **Socket Write Deadlines:** 10-second non-blocking deadlines (`SetWriteDeadline`) eliminate infinite KCP/Mux window deadlocks.
3. **Noise Protocol Handshake with Perfect Forward Secrecy:**
   - Ephemeral X25519 key exchange per session; compromise of long-term credentials does not expose historical traffic.
   - Dynamic pseudo-random length masking (16–48 bytes) on first-flight frames prevents machine learning length classifiers.
4. **Ultra-Low Resource Footprint:**
   - Requires <20 MB RSS memory on edge nodes (~74% less RAM than traditional Paqet setups).

---

## 📋 System & Network Prerequisites

Before starting the installation, ensure your environment meets the following technical requirements:

### 1. Server Requirements

| Component | Minimum Specification | Recommended Specification |
|---|---|---|
| **Operating System** | Ubuntu 20.04+ / Debian 11+ | Ubuntu 22.04 / 24.04 LTS or Debian 12 |
| **CPU / Architecture** | 1 vCPU (x86_64 or aarch64) | 2+ vCPU (x86_64) |
| **RAM** | 512 MB | 1 GB or higher |
| **Privileges** | `root` or `sudo` access | `root` access (required for systemd & iptables) |
| **Dependencies** | Python 3.10+, `curl`, `tar` | Automatically verified and installed by script |

### 2. Domain, Subdomain & DNS Configuration

While accessing the panel via raw IP (`http://SERVER_IP:9090`) works for initial trials, **using a domain or subdomain with SSL is strongly recommended** for production deployments:

1. **Create an A-Record:** Point your chosen subdomain (e.g. `panel.yourdomain.com`) to the public IP of the Master Panel server.
2. **Cloudflare Consideration (DNS-Only Mode):** If managing DNS through Cloudflare, set the record to **DNS-Only (Grey Cloud ⚪)**. Direct TCP connectivity avoids HTTP timeout disconnects on agent sync heartbeats.
3. **Reverse Proxy (SSL/TLS):** Run Nginx or Caddy on the panel server to terminate TLS on standard port `443` and forward requests to `http://127.0.0.1:9090`. See our detailed [Domain & SSL Reverse Proxy Guide](docs/DOMAIN_AND_SSL_GUIDE.md).

### 3. Port Management & Firewall Rules

| Port Type | Example Ports | Direction & Protocol | Purpose |
|---|---|---|---|
| **Panel Web Port** | `9090` (or `443` via Reverse Proxy) | Inbound TCP | Web dashboard access & Agent sync |
| **Dedicated Core Port** | `3107`, `9999` | Inbound TCP/UDP on Foreign VPS | Encrypted tunnel backbone between Iran and Kharej |
| **User Forwarded Ports** | `443`, `8443`, `2083` | Inbound TCP/UDP on Iran VPS | Client-facing traffic entry points |

> ⚠️ **Important:** Never use the same port number for both the **Core Port** and a **Forwarded Port**. The core port handles the internal tunnel transport; forwarded ports are the external entry points.

---

## ⚡ Quick & Easy Installation (1-Line Installer)

Hawal provides an automated, zero-configuration installation script. Can you install easily? **Yes, in under 60 seconds.**

### Step 1: Install the Master Panel

Run the following command on the server chosen to host the Hawal control plane:

```bash
curl -fsSL https://raw.githubusercontent.com/dalroot/hawal/master/install-panel.sh | bash
```

*Want to run on a custom port?* Use the `--port` parameter:

```bash
curl -fsSL https://raw.githubusercontent.com/dalroot/hawal/master/install-panel.sh | bash -s -- --port 9090
```

Once complete, open your browser and navigate to:
`http://YOUR_SERVER_IP:9090`

---

### Step 2: Enroll Nodes (Iran & Foreign Servers)

No configuration files need to be manually edited on your server nodes:

1. Log into the panel and navigate to **Node Management**.
2. Click **Add Node**, enter a friendly name, and choose the role (`iran` or `kharej`).
3. Copy the generated `curl` command and run it in your edge server's terminal:

```bash
curl -fsSL "http://PANEL_IP:9090/install?token=NODE_TOKEN&role=kharej&name=Germany" | bash
```

The node agent installs automatically as a managed systemd service (`hawal-agent.service`) and switches to **Online** within seconds.

---

### Step 3: Create Your First Tunnel

1. In the web dashboard, open **Tunnel Management** and click **Create Tunnel**.
2. Select your Iran node and Foreign node.
3. Choose your preferred tunnel core (e.g. **Hawal Core**).
4. Assign a unique **Core Port** (e.g. `3107`).
5. Configure your port mapping:
   ```text
   443=127.0.0.1:443
   8443=127.0.0.1:8443
   ```
6. Click **Save & Deploy**. Both node agents synchronize within seconds and activate the encrypted tunnel.

---

## 🐳 Alternative Deployment (Docker Compose)

For containerized environments:

```bash
git clone https://github.com/dalroot/hawal.git
cd hawal
docker compose up -d --build
docker compose logs -f hawal-panel
```

---

## 🔒 Securing the Panel with SSL & Reverse Proxy

Terminate TLS on standard HTTPS port 443 with Nginx and Let's Encrypt:

```bash
sudo apt-get update && sudo apt-get install -y nginx certbot python3-certbot-nginx

sudo bash -c 'cat > /etc/nginx/sites-available/hawal << EOF
server {
    server_name panel.yourdomain.com;
    location / {
        proxy_pass http://127.0.0.1:9090;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF'

sudo ln -sf /etc/nginx/sites-available/hawal /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d panel.yourdomain.com
```

---

## 🛡️ Raw-TCP & Kernel Wire Accounting

Both **Hawal Core (Rawpaq)** and **Paqet Wire** operate directly at the packet-filter layer without standard Linux socket descriptors:
- Standard utilities like `ss` or `/proc/PID/io` cannot measure raw socket traffic.
- Hawal solves this by directly harvesting byte counters from the dedicated `raw` table of `iptables` on the destination node.
- The reported volume represents **genuine physical wire usage**, including retransmissions, padding, and KCP framing overhead.
- Rules for `NOTRACK` and RST suppression are automatically provisioned and maintained by `hawal-agent`.

---

## 🛠️ Operations, Maintenance & Troubleshooting

```bash
# Interactive CLI dashboard (available on all nodes)
hawal

# Inspect service statuses
systemctl status hawal-panel --no-pager
systemctl status hawal-agent --no-pager

# Stream live service logs
journalctl -u hawal-panel -f
journalctl -u hawal-agent -f

# Inspect listening ports
ss -lntup | grep -E '9090|3107'
```

---

## 💬 Community & Support

Join our technical community to discuss network resilience, report emerging DPI behaviors, and collaborate with developers:

- 📢 **Telegram Channel & Community:** **[t.me/dallrep](https://t.me/dallrep)**
- 💡 **Network Evidence & Issue Reports:** Use our [DPI Network Report Form](https://github.com/dalroot/hawal/issues/new?template=dpi_network_report.yml) or open a [GitHub Issue](https://github.com/dalroot/hawal/issues).
- 🔬 **DPI & Censorship Research:** In-depth architectural and packet-manipulation documentation in [`docs/research/dpi-2026/`](docs/research/dpi-2026/).

---

## 📚 Documentation & Wiki

Explore our dedicated documentation guides in the [`docs/`](docs/) directory:

- 📖 **[Domain, Subdomain & SSL Setup Guide](docs/DOMAIN_AND_SSL_GUIDE.md)** — Nginx, Caddy, Cloudflare, and SSL configuration.
- 📖 **[Node Agent Operator Guide](docs/AGENT_GUIDE.md)** — In-depth agent lifecycle, sync internals, and firewall requirements.
- 🇮🇷 **[مستندات فارسی ایجنت](docs/AGENT_GUIDE_FA.md)** — راهنمای جامع فارسی معماری ایجنت نودها.
- 🇮🇷 **[راهنمای فارسی دامنه و اس‌اس‌ال](docs/DOMAIN_AND_SSL_GUIDE_FA.md)** — راهنمای کامل فارسی راه‌اندازی ساب‌دامین و ریورس پروکسی.

---

## 🤝 Contributing & License

Contributions, bug reports, and enhancements are warmly welcome! Please submit issues and pull requests following our community standards. When reporting an issue, please include your OS version, selected core, sanitized systemd logs, and reproduction steps.

Released under the **[MIT License](LICENSE)** © 2026 [dalroot](https://github.com/dalroot) and Hawal contributors.
