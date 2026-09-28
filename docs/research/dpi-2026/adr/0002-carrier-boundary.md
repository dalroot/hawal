# ADR-0002: Separate carriers from the secure session

- Status: accepted
- Date: 2026-09-03

## Decision

A carrier is responsible only for creating an ordered stream or datagram path.
Peer authentication, forward secrecy, replay protection, encrypted record
metadata, stream multiplexing, and reconnect policy belong above it.

The first carrier is plain TCP and acts as a diagnostic control. Future TLS,
HTTP, QUIC, and raw transports must implement the same narrow interface.

## Why

This boundary lets tests compare paths without duplicating cryptography. It also
keeps camouflage concerns out of key management and prevents a transport plugin
from silently weakening the secure-session guarantees.

## Security note

The TCP carrier alone is not a secure tunnel. No carrier is advertised as
undetectable, and a failed path is not labeled as DPI interference without
controlled endpoint evidence.

