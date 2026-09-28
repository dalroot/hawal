package secure

import (
	"context"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

var (
	ErrHandshakeTimeout = errors.New("secure: handshake timed out")
	ErrAuthFailed       = errors.New("secure: authentication failed")
	ErrInvalidPeerKey   = errors.New("secure: invalid peer public key")
)

// NoiseHandshaker implements Handshaker using X25519 Ephemeral Diffie-Hellman,
// HMAC-SHA256 authentication, variable random padding, and ChaCha20-Poly1305.
type NoiseHandshaker struct {
	token []byte
}

func NewNoiseHandshaker(token string) *NoiseHandshaker {
	return &NoiseHandshaker{token: []byte(token)}
}

func (h *NoiseHandshaker) Handshake(ctx context.Context, role Role, link carrier.Link, identity IdentityRef, limits Limits) (Result, error) {
	timeout := limits.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_ = link.SetDeadline(time.Now().Add(timeout))
	defer func() { _ = link.SetDeadline(time.Time{}) }()

	curve := ecdh.X25519()
	ephemPriv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return Result{}, fmt.Errorf("secure: generate ephemeral key: %w", err)
	}
	myPub := ephemPriv.PublicKey().Bytes()

	var (
		sharedSecret []byte
		transcript   [32]byte
		sendKey      []byte
		recvKey      []byte
		sendNonce    []byte
		recvNonce    []byte
	)

	switch role {
	case RoleInitiator:
		// 1. Client generates 16-48 bytes random padding
		padLen := randomPaddingLen(16, 48)
		padding := make([]byte, padLen)
		if _, err := io.ReadFull(rand.Reader, padding); err != nil {
			return Result{}, fmt.Errorf("secure: generate padding: %w", err)
		}

		// 2. Auth tag over client ephemeral public key
		authTag := computeHMAC(h.token, myPub)

		// Flight 1: [32B myPub] + [32B authTag] + [1B padLen] + [padLen padding]
		msg := make([]byte, 0, 65+padLen)
		msg = append(msg, myPub...)
		msg = append(msg, authTag...)
		msg = append(msg, byte(padLen))
		msg = append(msg, padding...)

		if _, err := link.Write(msg); err != nil {
			return Result{}, fmt.Errorf("secure: write initiator flight: %w", err)
		}

		// Read Flight 2 from Server: [32B serverPub] + [32B serverAuthTag] + [1B sPadLen]
		var sHdr [65]byte
		if _, err := io.ReadFull(link, sHdr[:]); err != nil {
			return Result{}, fmt.Errorf("secure: read responder flight header: %w", err)
		}
		serverPub := sHdr[:32]
		serverAuthTag := sHdr[32:64]
		sPadLen := int(sHdr[64])

		if sPadLen > 0 {
			discard := make([]byte, sPadLen)
			if _, err := io.ReadFull(link, discard); err != nil {
				return Result{}, fmt.Errorf("secure: read responder padding: %w", err)
			}
		}

		// Verify server authentication tag over (myPub || serverPub)
		expectedServerTag := computeHMAC(h.token, append(append([]byte(nil), myPub...), serverPub...))
		if !hmac.Equal(serverAuthTag, expectedServerTag) {
			return Result{}, ErrAuthFailed
		}

		serverPeerKey, err := curve.NewPublicKey(serverPub)
		if err != nil {
			return Result{}, ErrInvalidPeerKey
		}
		sharedSecret, err = ephemPriv.ECDH(serverPeerKey)
		if err != nil {
			return Result{}, fmt.Errorf("secure: ecdh computation: %w", err)
		}

		transcript = computeTranscript(h.token, myPub, serverPub)
		sendKey, recvKey, sendNonce, recvNonce = deriveKeys(sharedSecret, transcript)

	case RoleResponder:
		// Read Flight 1 from Client: [32B clientPub] + [32B clientAuthTag] + [1B cPadLen]
		var cHdr [65]byte
		if _, err := io.ReadFull(link, cHdr[:]); err != nil {
			return Result{}, fmt.Errorf("secure: read initiator flight header: %w", err)
		}
		clientPub := cHdr[:32]
		clientAuthTag := cHdr[32:64]
		cPadLen := int(cHdr[64])

		// Anti-DoS check before reading padding or allocating ephemeral crypto
		expectedClientTag := computeHMAC(h.token, clientPub)
		if !hmac.Equal(clientAuthTag, expectedClientTag) {
			return Result{}, ErrAuthFailed
		}

		if cPadLen > 0 {
			discard := make([]byte, cPadLen)
			if _, err := io.ReadFull(link, discard); err != nil {
				return Result{}, fmt.Errorf("secure: read initiator padding: %w", err)
			}
		}

		clientPeerKey, err := curve.NewPublicKey(clientPub)
		if err != nil {
			return Result{}, ErrInvalidPeerKey
		}
		sharedSecret, err = ephemPriv.ECDH(clientPeerKey)
		if err != nil {
			return Result{}, fmt.Errorf("secure: ecdh computation: %w", err)
		}

		// Compute response auth tag over (clientPub || myPub)
		serverAuthTag := computeHMAC(h.token, append(append([]byte(nil), clientPub...), myPub...))
		padLen := randomPaddingLen(16, 48)
		padding := make([]byte, padLen)
		if _, err := io.ReadFull(rand.Reader, padding); err != nil {
			return Result{}, fmt.Errorf("secure: generate padding: %w", err)
		}

		// Flight 2: [32B myPub] + [32B serverAuthTag] + [1B padLen] + [padLen padding]
		msg := make([]byte, 0, 65+padLen)
		msg = append(msg, myPub...)
		msg = append(msg, serverAuthTag...)
		msg = append(msg, byte(padLen))
		msg = append(msg, padding...)

		if _, err := link.Write(msg); err != nil {
			return Result{}, fmt.Errorf("secure: write responder flight: %w", err)
		}

		transcript = computeTranscript(h.token, clientPub, myPub)
		// For responder: sendKey is s2cKey (which is initiator's recvKey), recvKey is c2sKey
		c2sKey, s2cKey, c2sNonce, s2cNonce := deriveKeys(sharedSecret, transcript)
		sendKey, recvKey = s2cKey, c2sKey
		sendNonce, recvNonce = s2cNonce, c2sNonce

	default:
		return Result{}, errors.New("secure: unknown role")
	}

	sendAEAD, err := chacha20poly1305.New(sendKey)
	if err != nil {
		return Result{}, fmt.Errorf("secure: create send AEAD: %w", err)
	}
	recvAEAD, err := chacha20poly1305.New(recvKey)
	if err != nil {
		return Result{}, fmt.Errorf("secure: create recv AEAD: %w", err)
	}

	return Result{
		Peer: identity,
		Secrets: Secrets{
			SendAEAD:     sendAEAD,
			ReceiveAEAD:  recvAEAD,
			SendNonce:    sendNonce,
			ReceiveNonce: recvNonce,
			Transcript:   transcript,
		},
		ExportedAt: time.Now(),
	}, nil
}

func computeHMAC(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}

func computeTranscript(token, cPub, sPub []byte) [32]byte {
	h := sha256.New()
	h.Write([]byte("hawal-v2-transcript"))
	h.Write(token)
	h.Write(cPub)
	h.Write(sPub)
	var t [32]byte
	copy(t[:], h.Sum(nil))
	return t
}

func deriveKeys(sharedSecret []byte, transcript [32]byte) (c2sKey, s2cKey, c2sNonce, s2cNonce []byte) {
	kdf := hkdf.New(sha256.New, sharedSecret, transcript[:], []byte("hawal-v2-key-expansion"))
	material := make([]byte, 32+32+12+12)
	_, _ = io.ReadFull(kdf, material)
	c2sKey = material[0:32]
	s2cKey = material[32:64]
	c2sNonce = material[64:76]
	s2cNonce = material[76:88]
	return
}

func randomPaddingLen(min, max int) int {
	var b [1]byte
	_, _ = rand.Read(b[:])
	return min + int(b[0])%(max-min+1)
}
