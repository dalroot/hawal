# Hawal project context

When a task concerns DPI, censorship behavior, tunnel reliability, Hawal Core v2,
traffic shaping, active probing, packet transports, or country-specific filtering,
read these files before proposing or implementing a change:

1. `docs/research/dpi-2026/README_FA.md`
2. `docs/research/dpi-2026/evidence-ledger.md`
3. `docs/research/dpi-2026/hawal-v2-requirements.md`
4. `docs/research/dpi-2026/owned-lab-test-matrix.md`

Treat sourced observations, inferences, and experimental hypotheses as different
confidence classes. Do not generalize one ISP, port, server pair, or test date to a
country. Do not claim a transport is undetectable or censorship-resistant solely
because it adds encryption, fragmentation, padding, QUIC, TLS, or a new header.

Run active network experiments only between endpoints controlled by the user. Do
not spoof third-party addresses, scan unrelated networks, generate amplification,
or train noisy evasion strategies against public infrastructure. Protect the
production management path and require an explicit rollout decision before changing
live tunnel wire behavior.
