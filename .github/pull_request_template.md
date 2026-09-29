## 📝 Description

Please provide a concise description of the changes introduced by this pull request.

- What problem does this solve?
- What are the design considerations or trade-offs?

## 🔗 Related Issues
Fixes #(issue number)

## 🛡️ DPI & Network Invariants Checklist
Before submitting network or core protocol changes, please review and confirm:
- [ ] No changes to the live production tunnel wire behavior without explicit rollout plan.
- [ ] No third-party IP spoofing, port amplification, or non-controlled target scanning.
- [ ] Cryptographic security is decoupled from wire camouflage / shaping policies.
- [ ] Structured telemetry does not leak sensitive user payloads, tokens, or plaintext keys.

## 🧪 Testing & Verification
- [ ] Unit tests added / updated and passing (`go test ./v2/...`).
- [ ] Validated on both Linux client and server endpoints.
- [ ] No unbounded memory allocations or CPU starvation.

## 📋 Checklist
- [ ] My code follows the repository style and clean architecture guidelines.
- [ ] I have updated documentation or `CHANGELOG.md` where appropriate.
- [ ] CI pipeline and security checks pass without errors.
