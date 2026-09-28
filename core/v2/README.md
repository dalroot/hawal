# Hawal Core v2 (experimental)

This directory contains the side-by-side implementation of Hawal Core v2.
It is intentionally not wired into the agent, installer, or web panel yet.

Implemented foundation:

- transport-independent carrier contracts and a concurrency-safe registry;
- a baseline TCP carrier and loopback integration test;
- encrypted version/capability negotiation data structures;
- a secure-handshake boundary that accepts only vetted implementations;
- a bounded authenticated record codec with encrypted metadata, length buckets,
  direction-separated nonces, transcript AAD, and a per-key record limit;
- a weighted, bounded mux scheduler and byte-credit flow-control window;
- an explicit session lifecycle and idempotent multi-path promotion;
- path selection with dwell time, hysteresis, and stale-sample rejection;
- a shaping plugin boundary with enforced overhead and delay budgets;
- a bounded, structured event recorder that does not accept arbitrary log text;
- conservative failure classification (a failed connection is never treated as
  proof of DPI interference); and
- central operational limits plus stable typed failure categories.

Package direction:

```text
application adapters
        |
       mux  <---- policy / shaping
        |
      record
        |
 secure session ---- protocol negotiation
        |
     carrier ---- observe / controller
```

Hard invariants:

1. Carriers never receive tokens, databases, or port mappings.
2. Negotiation occurs after authentication and is not a cleartext magic prefix.
3. Record allocation, queues, windows, paths, and key lifetime are bounded.
4. Attaching a path never closes the current primary path.
5. Logs have stable structured fields and no arbitrary payload text.
6. Adaptive decisions require fresh measurements, dwell time, and hysteresis.

Not implemented yet:

- a concrete TLS 1.3 or audited Noise handshaker;
- key update execution and replay cache for resumptions;
- the complete stream state machine and record pump;
- TLS/HTTP, QUIC, and owned raw-packet carriers;
- agent/panel configuration and production rollout.

The v1 wire format and running production tunnels remain unchanged.
