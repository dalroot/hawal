# Paqet attribution and integration boundary

Hawal's experimental `RawPaq` carrier is informed by and interoperates with
ideas from the Paqet project:

- Upstream: <https://github.com/hanselime/paqet>
- Reviewed release: `v1.0.0-alpha.21`
- Reviewed commit: `4b81ce0ab56b7cc9aec7501c87c4804676cdb1b4`
- Review date: 2026-09-03
- License: MIT

The copyright and permission notice from Paqet is preserved in `LICENSE` in
this directory. At this milestone no Paqet source file has been copied into
Hawal. The carrier uses the same upstream `kcp-go` dependency and reproduces
documented tuning values, while deliberately excluding Paqet's smux,
forwarding, SOCKS, CLI, configuration, and logging layers.

If the Linux PCAP packet backend is later derived from Paqet source, every
derived file must carry an SPDX/MIT attribution header and this notice must be
updated with the exact upstream paths and local modifications.

