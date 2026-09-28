# ADR-0001: Build Hawal Core v2 beside v1

- Status: accepted
- Date: 2026-09-03

## Decision

Hawal Core v2 is implemented under `core/v2` and is not connected to the
existing agent, installer, panel, or v1 configuration format until its local
and owned-endpoint acceptance tests pass.

## Why

The existing tunnel is the operator's only management path. Replacing its wire
format in place would make rollback difficult and could sever access. A
side-by-side package also prevents accidental compatibility promises while the
secure session and carrier APIs are still evolving.

## Consequences

- v1 remains deployable and unchanged.
- v2 tests can run without privileged networking or production traffic.
- rollout needs an explicit feature flag and a separate migration ADR.

