#!/usr/bin/env bash
# ⚡ Hawal Tunnel (هه‌واڵ) - One-Line Master Panel Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/dalroot/hawal/master/install-panel.sh | bash

set -e

PORT="9090"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --port) PORT="$2"; shift 2 ;;
    *) shift ;;
  esac
done

echo "🚀 ==============================================="
echo "⚡ Installing Hawal Tunnel Control Panel..."
echo "🌐 Web Port: ${PORT}"
echo "==============================================="

INSTALL_DIR="/opt/hawal-panel"
mkdir -p "$INSTALL_DIR" /etc/hawal

# Check Python 3
if ! command -v python3 &> /dev/null; then
  echo "📦 Installing Python3..."
  apt-get update -y && apt-get install -y python3 curl tar
fi

# Clone or download repository
echo "📥 Fetching Hawal Tunnel source..."
rm -rf /tmp/hawal-temp
curl -fsSL https://github.com/dalroot/hawal/archive/refs/heads/master.tar.gz -o /tmp/hawal.tar.gz 2>/dev/null || curl -fsSL https://github.com/dalroot/hawal/archive/refs/heads/main.tar.gz -o /tmp/hawal.tar.gz
mkdir -p /tmp/hawal-temp
tar -xzf /tmp/hawal.tar.gz -C /tmp/hawal-temp --strip-components=1
cp -r /tmp/hawal-temp/* "$INSTALL_DIR/"
rm -rf /tmp/hawal-temp /tmp/hawal.tar.gz

chmod +x "$INSTALL_DIR/server.py" "$INSTALL_DIR/start.sh"

# Create Systemd Service for Panel
cat > /etc/systemd/system/hawal-panel.service << EOF
[Unit]
Description=Hawal Tunnel Master Control Panel
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=/usr/bin/python3 ${INSTALL_DIR}/server.py --port ${PORT}
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable hawal-panel
systemctl restart hawal-panel

SERVER_IP=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{print $7}' || true)
if [[ -z "$SERVER_IP" ]]; then
  SERVER_IP=$(hostname -I 2>/dev/null | awk '{print $1}' || true)
fi
if [[ -z "$SERVER_IP" ]] || [[ "$SERVER_IP" =~ ^10\. ]] || [[ "$SERVER_IP" =~ ^192\.168\. ]] || [[ "$SERVER_IP" =~ ^172\.(1[6-9]|2[0-9]|3[0-1])\. ]]; then
  CANDIDATE_IP=$(curl -s -4 --connect-timeout 2 https://icanhazip.com 2>/dev/null || curl -s -4 --connect-timeout 2 https://api.ipify.org 2>/dev/null || true)
  if echo "$CANDIDATE_IP" | grep -E -q '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'; then
    SERVER_IP="$CANDIDATE_IP"
  fi
fi
SERVER_IP="${SERVER_IP:-YOUR_SERVER_IP}"

echo "==============================================="
echo "🎉 Hawal Tunnel Panel successfully installed & running!"
echo "👉 Dashboard URL: http://${SERVER_IP}:${PORT}"
echo "==============================================="
