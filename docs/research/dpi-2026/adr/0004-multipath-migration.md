# ADR-0004: Idempotent multi-path migration

- Status: accepted for prototype
- Date: 2026-09-03

## Decision

A session owns a bounded path set. The first attached link becomes primary;
subsequent links remain standby. Attaching a link never closes or replaces the
current primary.

Promotion is explicit and uses an expected topology generation. A repeated
promotion to the already-primary path succeeds without another state change.
A competing attach, detach, or health transition invalidates stale promotion
decisions and requires a fresh snapshot.

The former primary remains open as standby. Detaching the primary is rejected;
the caller must first promote a verified alternative and drain outstanding
records.

## Controller guardrails

- stale measurements are ignored;
- a healthy path is retained for a minimum dwell period;
- performance migration requires a configured improvement margin;
- failure threshold may trigger failover without waiting for dwell;
- a path decision never labels its cause as censorship or DPI detection.

