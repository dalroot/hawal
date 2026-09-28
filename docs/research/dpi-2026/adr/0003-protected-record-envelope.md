# ADR-0003: Protected record envelope

- Status: accepted for prototype
- Date: 2026-09-03

## Decision

The reliable-stream record envelope has a four-byte ciphertext length followed
by AEAD ciphertext. There is no plaintext magic value or version. Record type,
flags, stream ID, direction sequence, actual payload length, payload, and random
padding are all authenticated and encrypted.

The visible ciphertext length must correspond to a configured bucket. The
handshake transcript and visible bucket length are AEAD additional data. Each
direction receives a distinct AEAD and base nonce from the secure handshaker.
The final eight nonce bytes are XORed with a monotonic direction sequence.

## Limits

- ciphertext length is checked before allocation;
- configured buckets are sorted, unique, and capped;
- an authenticated internal sequence must match the expected sequence;
- each traffic key has an explicit record limit;
- reaching that limit fails closed until the session performs key update.

## Deferred

The key derivation, handshake pattern, key-update transcript, and resumption
replay cache remain blocked on the secure-session ADR and library review. The
record package accepts `cipher.AEAD`; it does not construct keys or invent a
cipher.

