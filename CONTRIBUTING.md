# Contributing to Hawal Tunnel

Thank you for your interest in contributing to **Hawal Tunnel (هه‌واڵ)**! We welcome contributions that improve tunnel reliability, observability, UI/UX aesthetics, and security.

---

## 🏛️ Code & Design Principles

1. **Anti-Slop Design Doctrine:**
   - Keep user interfaces clean, intuitive, and modern (Bento-grid aesthetic, micro-animations, clear telemetry).
   - Avoid visual clutter, redundant status widgets, or misleading metrics.

2. **Network Protocol Discipline:**
   - Follow the guidelines in `docs/research/dpi-2026/`.
   - Never claim a transport is "undetectable" or "unblockable" solely because it uses encryption, TLS, QUIC, or padding.
   - Decouple cryptographic security (Noise, PFS, ChaCha20-Poly1305) from wire camouflage and packet-shaping profiles.
   - Run active network benchmarks only between server endpoints under your direct control.

3. **Zero Orphan Processes:**
   - Process lifecycles must be strictly managed using process group isolation (`os.killpg` / `start_new_session=True`).
   - Listeners must cleanly release bound TCP/UDP ports upon termination.

---

## 🛠️ Local Development Setup

### Core v2 (Go 1.22+)
```bash
cd core
# Run unit and fuzz tests
go test -v ./v2/...

# Build hawal-core standalone binary
go build -o hawal-core ./v2/cmd/hawal-core
```

### Web Panel & Agent (Python 3.10+)
```bash
# Setup virtual environment
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt flake8 pytest

# Start development panel
python3 server.py --port 8080

# Check Python linting
flake8 . --count --select=E9,F63,F7,F82 --show-source --statistics --exclude=.git,__pycache__,docs,ui-v2
```

---

## 🔀 Pull Request Process

1. Fork the repository and create your branch from `master` (`git checkout -b feat/my-feature`).
2. Ensure all Go unit tests pass (`cd core && go test ./v2/...`).
3. Ensure no linting errors are introduced in Python codebase.
4. Update `CHANGELOG.md` with relevant details following the keep-a-changelog format.
5. Open a Pull Request against `dalroot/hawal:master` using the PR template.
6. Verify that the GitHub Actions CI and CodeQL security checks pass.
