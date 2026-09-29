# Security Policy

The Hawal Tunnel project takes the security and integrity of tunneling protocols, access controls, and network communications seriously.

## Supported Versions

Only the latest minor release is actively maintained and receives security updates:

| Version | Supported          |
| ------- | ------------------ |
| 2.3.x   | :white_check_mark: |
| 2.2.x   | :x:                |
| < 2.2.0 | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in Hawal (such as authentication bypass, token leakage, arbitrary remote execution, or memory safety corruption in `core/v2`), please do **NOT** open a public GitHub issue.

Instead, please responsibly disclose the vulnerability through:
- **GitHub Private Vulnerability Reporting:** Use the "Report a vulnerability" button under the **Security** tab of this repository.
- **Maintainer Contact:** Reach out directly to the core maintainer team at `security@dalroot.org` or via encrypted channel.

### What to Include in Your Report
1. Description of the vulnerability and its potential security impact.
2. Step-by-step instructions to reproduce the issue (proof-of-concept scripts or commands).
3. Component affected (e.g. Master Web Panel, Agent API, Hawal Core v2, GOST relay).
4. Any proposed mitigation or remediation patch if available.

### Safe Harbor & Responsible Research Rules
- Network experiments must be conducted exclusively between nodes owned and operated by you.
- Do not attempt denial of service against third parties or spoof non-owned IP ranges.
- Do not exploit discovered vulnerabilities to exfiltrate user data.
- We will acknowledge receipt of your report within 48 hours and coordinate a coordinated disclosure timeline.
