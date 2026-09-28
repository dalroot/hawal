# 🌐 Domain, Subdomain & SSL Reverse Proxy Guide for Hawal

This guide explains how to bind a custom domain or subdomain (e.g. `panel.yourdomain.com`) to the Hawal Master Panel, secure it with free Let's Encrypt SSL/TLS certificates, and configure reverse proxies using **Nginx** or **Caddy**.

---

## Table of Contents
1. [Why Use a Domain & Reverse Proxy?](#1-why-use-a-domain--reverse-proxy)
2. [DNS Configuration (A-Record Setup)](#2-dns-configuration-a-record-setup)
3. [Cloudflare DNS vs Proxy Mode](#3-cloudflare-dns-vs-proxy-mode)
4. [Option A: Nginx + Certbot (Recommended for Standard Linux)](#option-a-nginx--certbot-recommended)
5. [Option B: Caddy (Automatic SSL with Zero Config)](#option-b-caddy-automatic-ssl)
6. [Firewall & Security Considerations](#6-firewall--security-considerations)
7. [Updating Agent Install URL](#7-updating-agent-install-url)

---

## 1. Why Use a Domain & Reverse Proxy?

By default, the Hawal panel listens on plain HTTP at `http://SERVER_IP:9090`. While fine for private internal testing, deploying on a public VPS benefits significantly from a domain and reverse proxy:
- **Encrypted Transmission:** All credentials, node tokens, and tunnel control commands are protected with TLS 1.3 encryption.
- **Port Masking:** Serve the web interface over standard HTTPS (`443`) rather than exposing non-standard port `9090`.
- **IP Protection:** Easily point your subdomain to a new server IP without reconfiguring bookmarks.
- **WebSockets / Keep-Alive Support:** Long-polling and live stat streams remain stable behind enterprise reverse proxies.

---

## 2. DNS Configuration (A-Record Setup)

1. Log in to your DNS provider (e.g., Cloudflare, ArvanCloud, Namecheap, Hetzner DNS).
2. Create an **A Record**:
   - **Type:** `A`
   - **Name / Host:** `panel` (or `@` for root domain, or any subdomain of choice)
   - **Target / Value:** `<YOUR_PANEL_SERVER_IP>`
   - **TTL:** `Auto` or `2 min`
3. Verify DNS propagation:
   ```bash
   dig +short panel.yourdomain.com
   # or
   ping -c 2 panel.yourdomain.com
   ```

---

## 3. Cloudflare DNS vs Proxy Mode

If your domain DNS is managed by Cloudflare:
- **Recommended: DNS-Only (Grey Cloud ⚪ / Deactivated Proxy)**
  Set the proxy status of `panel.yourdomain.com` to **DNS only**.
  *Reason:* Direct TCP connectivity ensures zero timeout interference with continuous panel polling, agent heartbeats, and raw WebSocket streams.
- **If Proxied (Orange Cloud 🟠):**
  If you must hide the server IP behind Cloudflare's CDN:
  - Make sure **WebSockets** is toggled ON under `Cloudflare Dashboard -> Network`.
  - Set SSL mode to **Full (Strict)** under `SSL/TLS -> Overview`.
  - Keep in mind: Cloudflare CDN proxies only HTTP/HTTPS ports (80, 443, 8443, 2053, 2083, 2087, 2096). **Tunnel core traffic (e.g. 3107) between Iran and foreign nodes MUST NOT pass through Cloudflare CDN** and must connect directly.

---

## 4. Option A: Nginx + Certbot (Recommended)

### Step 1: Install Nginx and Certbot
On Ubuntu / Debian:
```bash
sudo apt-get update
sudo apt-get install -y nginx certbot python3-certbot-nginx
```

### Step 2: Configure Nginx Server Block
Create `/etc/nginx/sites-available/hawal`:
```nginx
server {
    server_name panel.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:9090;
        proxy_http_version 1.1;

        # WebSocket & Streaming support
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Standard proxy headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts for real-time polling
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
```

Enable the configuration:
```bash
sudo ln -sf /etc/nginx/sites-available/hawal /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Step 3: Issue SSL Certificate via Let's Encrypt
```bash
sudo certbot --nginx -d panel.yourdomain.com
```
Follow the interactive prompts to enable automatic HTTPS redirection.

---

## 5. Option B: Caddy (Automatic SSL)

Caddy automatically provisions, verifies, and renews Let's Encrypt SSL certificates with zero manual intervention.

### Step 1: Install Caddy
```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install -y caddy
```

### Step 2: Configure Caddyfile
Edit `/etc/caddy/Caddyfile`:
```caddy
panel.yourdomain.com {
    reverse_proxy 127.0.0.1:9090
}
```

Restart Caddy:
```bash
sudo systemctl restart caddy
```
Your panel is now live at `https://panel.yourdomain.com` with valid TLS!

---

## 6. Firewall & Security Considerations

Once your reverse proxy is active:
1. Allow public HTTPS traffic:
   ```bash
   sudo ufw allow 80/tcp
   sudo ufw allow 443/tcp
   ```
2. (Optional) Restrict raw port `9090` so it only accepts local loopback connections:
   ```bash
   sudo ufw delete allow 9090/tcp 2>/dev/null || true
   ```
3. Remember to leave your tunnel core ports (e.g. `3107`) open for node-to-node communication.

---

## 7. Updating Agent Install URL

When you add a node in the Hawal web UI, the panel generates an install command.
If you have set up a domain with HTTPS, you can invoke the installer pointing to your domain:
```bash
curl -fsSL "https://panel.yourdomain.com/install?token=YOUR_NODE_TOKEN&role=kharej&name=Germany" | bash
```
The node agent will securely connect over HTTPS, synchronizing its state with complete cryptographic protection.
