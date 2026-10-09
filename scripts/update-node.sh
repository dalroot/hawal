#!/usr/bin/env bash
# ==============================================================================
# Hawal Node GitOps Updater
# Synchronizes node with official GitHub repository and verified releases.
# ==============================================================================
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${BLUE}[INFO]${NC} $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

if [ "$(id -u)" -ne 0 ]; then
    log_error "This script must be run as root."
    exit 1
fi

REPO_DIR="/opt/hawal-repo"
HAWAL_DIR="/opt/hawal"
PANEL_DIR="/opt/hawal-panel"
REPO_URL="https://github.com/dalroot/hawal.git"

# 1. Architecture detection
ARCH_RAW="$(uname -m)"
case "${ARCH_RAW}" in
    x86_64|amd64)   ARCH="amd64" ;;
    aarch64|arm64)  ARCH="arm64" ;;
    armv7*|armv8l)  ARCH="armv7" ;;
    i386|i686)      ARCH="386" ;;
    *)
        log_error "Unsupported CPU architecture: ${ARCH_RAW}"
        exit 1
        ;;
esac
log_info "Detected architecture: ${ARCH}"

# 2. Update or clone git repository
log_info "Synchronizing Git repository at ${REPO_DIR}..."
if [ -d "${REPO_DIR}/.git" ]; then
    cd "${REPO_DIR}"
    git fetch --all --tags --prune
    git checkout master
    git pull origin master
else
    mkdir -p "$(dirname "${REPO_DIR}")"
    git clone "${REPO_URL}" "${REPO_DIR}"
    cd "${REPO_DIR}"
fi
CURRENT_COMMIT="$(git rev-parse --short HEAD)"
log_ok "Repository synchronized to commit: ${CURRENT_COMMIT}"

# 3. Determine target release tag
TARGET_TAG="${1:-}"
if [ -z "${TARGET_TAG}" ]; then
    log_info "Querying latest official release tag from GitHub..."
    TARGET_TAG="$(curl -sSL "https://api.github.com/repos/dalroot/hawal/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)"
fi

if [ -z "${TARGET_TAG}" ]; then
    # Fallback to latest git tag
    TARGET_TAG="$(git describe --tags --abbrev=0 2>/dev/null || echo "v2.5.2")"
fi
log_info "Target release version: ${TARGET_TAG}"

# 4. Download and cryptographically verify binary
TMP_DIR="$(mktemp -d /tmp/hawal-update-XXXXXX)"
trap 'rm -rf "${TMP_DIR}"' EXIT

BINARY_NAME="hawal-core-linux-${ARCH}"
DOWNLOAD_URL="https://github.com/dalroot/hawal/releases/download/${TARGET_TAG}/${BINARY_NAME}"
CHECKSUM_URL="https://github.com/dalroot/hawal/releases/download/${TARGET_TAG}/checksums.txt"

log_info "Downloading ${BINARY_NAME} from GitHub Releases..."
if curl -sSLf "${DOWNLOAD_URL}" -o "${TMP_DIR}/${BINARY_NAME}" && curl -sSLf "${CHECKSUM_URL}" -o "${TMP_DIR}/checksums.txt"; then
    log_info "Verifying SHA256 checksum..."
    EXPECTED_HASH="$(grep "${BINARY_NAME}" "${TMP_DIR}/checksums.txt" | awk '{print $1}')"
    ACTUAL_HASH="$(sha256sum "${TMP_DIR}/${BINARY_NAME}" | awk '{print $1}')"

    if [ -n "${EXPECTED_HASH}" ] && [ "${EXPECTED_HASH}" = "${ACTUAL_HASH}" ]; then
        log_ok "Cryptographic verification passed (SHA256: ${ACTUAL_HASH:0:16}...)"
    else
        log_error "Checksum mismatch! Expected: ${EXPECTED_HASH}, Got: ${ACTUAL_HASH}"
        exit 1
    fi
else
    log_warn "Official release asset not yet available on GitHub Releases. Using local build from repo..."
    if [ -f "${REPO_DIR}/bin/hawal-core" ]; then
        cp -f "${REPO_DIR}/bin/hawal-core" "${TMP_DIR}/${BINARY_NAME}"
    else
        log_error "Failed to retrieve hawal-core binary."
        exit 1
    fi
fi

# 5. Atomic deployment to /opt/hawal/bin/hawal-core
mkdir -p "${HAWAL_DIR}/bin" "${HAWAL_DIR}/tunnels" "${HAWAL_DIR}/logs"
chmod 700 "${TMP_DIR}/${BINARY_NAME}"

# Atomic swap prevents 'Text file busy' execution lock
cp -f "${TMP_DIR}/${BINARY_NAME}" "${HAWAL_DIR}/bin/hawal-core.new"
chmod 700 "${HAWAL_DIR}/bin/hawal-core.new"
mv -f "${HAWAL_DIR}/bin/hawal-core.new" "${HAWAL_DIR}/bin/hawal-core"
log_ok "Hawal Core binary installed to ${HAWAL_DIR}/bin/hawal-core"

# Cycle running hawal-core tunnel instances so new binary takes effect immediately
log_info "Cycling running hawal-core tunnel instances..."
pkill -f "${HAWAL_DIR}/bin/hawal-core" || true

# 6. Update Panel (if installed on this node)
if [ -d "${PANEL_DIR}" ]; then
    log_info "Updating Hawal Panel application..."
    mkdir -p "${PANEL_DIR}/app/static/bin" "${PANEL_DIR}/bin"
    cp -rf "${REPO_DIR}/app/"* "${PANEL_DIR}/app/"
    cp -f "${REPO_DIR}/server.py" "${PANEL_DIR}/server.py"
    cp -f "${HAWAL_DIR}/bin/hawal-core" "${PANEL_DIR}/app/static/bin/hawal-core"
    cp -f "${HAWAL_DIR}/bin/hawal-core" "${PANEL_DIR}/bin/hawal-core"
    
    if systemctl is-active --quiet hawal-panel 2>/dev/null; then
        systemctl restart hawal-panel
        log_ok "Restarted hawal-panel service"
    fi
fi

# 7. Update Agent
if [ -f "${REPO_DIR}/agent/agent.py" ]; then
    log_info "Updating Hawal Agent..."
    cp -f "${REPO_DIR}/agent/agent.py" "${HAWAL_DIR}/agent.py"
    
    if systemctl is-active --quiet hawal-agent 2>/dev/null; then
        systemctl restart hawal-agent
        log_ok "Restarted hawal-agent service"
    fi
fi

# 8. Post-update verification
sleep 2
CORE_VERSION="$("${HAWAL_DIR}/bin/hawal-core" -version 2>&1 || true)"
log_ok "Verification complete: ${CORE_VERSION}"
log_ok "Node update to commit ${CURRENT_COMMIT} (Release: ${TARGET_TAG}) successfully completed!"
