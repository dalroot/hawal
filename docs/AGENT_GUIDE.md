# 📖 Hawal Node Agent Architecture & Operator Guide

This document is for server administrators and DevOps operators running Hawal Foreign VPS Node Agents.

---

### 1. Zero-Touch Automatic Synchronization

* **No Manual Configuration Required:**
  All tunnel definitions, entry ports, certificates, and credentials are centrally defined on the Hawal Master Panel (Iran).
* **Sync Mechanism:**
  The agent daemon (`/opt/hawal/agent.py`) connects to the Master Panel every 3–5 seconds using its unique node token to sync tunnel configurations.
* **Process Lifecycle:**
  When tunnels are created, updated, or removed in the web panel, the agent automatically spawns, updates, or gracefully terminates the corresponding relay processes.

---

### 2. Firewall Requirements (Foreign VPS)

1. **Relay Core Ports (Inbound TCP):**
   * Examples: Port `3095` (Subscription Tunnel) or Port `3101` (VLESS Traffic Tunnel).
   * These ports must be open inbound in your cloud provider firewall (Hetzner, OVH, DigitalOcean, AWS, etc.) so that the Iran server can establish TLS 1.3 encrypted relay sessions.
2. **Local Target Ports (127.0.0.1 Only):**
   * Examples: Port `2096` (3X-UI subscription web service) or Ports `2087/2088` (Xray inbounds).
   * These ports listen locally on `127.0.0.1`. The Hawal tunnel terminates relay traffic directly to these loopback ports. They do not need to be exposed to the public internet.

---

### 3. Essential CLI Commands (`hawal`)

* **Interactive Control Dashboard:**
  ```bash
  hawal
  ```
* **Inspect Active Tunnels:**
  ```bash
  hawal tunnels
  ```
* **Benchmark Latency & Connectivity to Master:**
  ```bash
  hawal ping
  ```
* **Stream Live Agent & Tunnel Logs:**
  ```bash
  hawal logs
  ```
* **Display Node Credentials & Panel URL:**
  ```bash
  hawal info
  ```
* **Safely Restart Agent & Tunnels:**
  ```bash
  hawal restart
  ```

---

### 4. Troubleshooting Checklist

| Symptom | Probable Cause | Quick Resolution |
| :--- | :--- | :--- |
| **Node shows "Offline" in panel** | Network timeout or port 9090 unreachable on Iran master | Run `hawal ping` on node to verify TCP reachability to master |
| **Tunnel error `i/o timeout`** | Relay core port blocked by cloud firewall | Open the relay port (e.g. 3095 or 3101) inbound on the foreign VPS |
| **Orphan processes on port** | Residual process from killed service | Run `hawal restart` to clean up old processes cleanly |

---
