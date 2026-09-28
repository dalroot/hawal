// Package record implements bounded, authenticated records after handshake.
package record

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"sync"
)

const headerSize = 24

var ErrKeyExhausted = errors.New("record: traffic key record limit reached")

// Type is encrypted metadata; it never appears in the outer record prefix.
type Type uint8

const (
	TypeOpen Type = iota + 1
	TypeData
	TypeWindowUpdate
	TypeHalfClose
	TypeReset
	TypePing
	TypeGoAway
	TypeKeyUpdate
)

type Flags uint8

const (
	FlagDatagram Flags = 1 << iota
	FlagFinal
)

// Record is the protected unit exchanged by the mux layer.
type Record struct {
	Type     Type
	Flags    Flags
	StreamID uint64
	Payload  []byte
}

// Options applies hard allocation and key-lifetime limits.
type Options struct {
	MaxPlaintext     int
	Buckets          []int
	MaxRecordsPerKey uint64
	MaskLength       bool
}

func DefaultOptions() Options {
	return Options{
		MaxPlaintext:     64 << 10,
		Buckets:          []int{256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
		MaxRecordsPerKey: 1 << 32,
		MaskLength:       true,
	}
}

// Codec owns one send and one receive sequence space. Concurrent reads and
// writes are supported, but there must be only one reader of a reliable link.
type Codec struct {
	send direction
	recv direction
	opts Options
	aad  [32]byte
}

type direction struct {
	mu        sync.Mutex
	aead      cipher.AEAD
	baseNonce []byte
	sequence  uint64
}

func New(sendAEAD, receiveAEAD cipher.AEAD, sendNonce, receiveNonce []byte, transcript [32]byte, options Options) (*Codec, error) {
	if sendAEAD == nil || receiveAEAD == nil {
		return nil, errors.New("record: send and receive AEAD are required")
	}
	if len(sendNonce) != sendAEAD.NonceSize() || len(receiveNonce) != receiveAEAD.NonceSize() || len(sendNonce) < 8 {
		return nil, errors.New("record: invalid base nonce size")
	}
	if options.MaxPlaintext < headerSize || options.MaxPlaintext > math.MaxUint32 {
		return nil, errors.New("record: invalid plaintext limit")
	}
	if options.MaxRecordsPerKey == 0 {
		return nil, errors.New("record: key record limit must be positive")
	}
	buckets := append([]int(nil), options.Buckets...)
	sort.Ints(buckets)
	last := 0
	for _, bucket := range buckets {
		if bucket < headerSize || bucket > options.MaxPlaintext || bucket == last {
			return nil, errors.New("record: invalid padding buckets")
		}
		last = bucket
	}
	if len(buckets) == 0 || buckets[len(buckets)-1] != options.MaxPlaintext {
		return nil, errors.New("record: buckets must include MaxPlaintext")
	}
	options.Buckets = buckets
	return &Codec{
		send: direction{aead: sendAEAD, baseNonce: append([]byte(nil), sendNonce...)},
		recv: direction{aead: receiveAEAD, baseNonce: append([]byte(nil), receiveNonce...)},
		opts: options,
		aad:  transcript,
	}, nil
}

// Write encrypts and writes exactly one record. The visible outer length is a
// configured bucket size plus AEAD overhead, never the actual payload length.
func (c *Codec) Write(w io.Writer, record Record) error {
	if !validType(record.Type) {
		return errors.New("record: invalid type")
	}
	target, ok := c.bucketFor(headerSize + len(record.Payload))
	if !ok {
		return fmt.Errorf("record: payload of %d bytes exceeds limit", len(record.Payload))
	}

	c.send.mu.Lock()
	defer c.send.mu.Unlock()
	if c.send.sequence >= c.opts.MaxRecordsPerKey {
		return ErrKeyExhausted
	}
	plaintext := make([]byte, target)
	plaintext[0] = byte(record.Type)
	plaintext[1] = byte(record.Flags)
	binary.BigEndian.PutUint64(plaintext[4:12], record.StreamID)
	binary.BigEndian.PutUint64(plaintext[12:20], c.send.sequence)
	binary.BigEndian.PutUint32(plaintext[20:24], uint32(len(record.Payload)))
	copy(plaintext[headerSize:], record.Payload)
	if padding := plaintext[headerSize+len(record.Payload):]; len(padding) > 0 {
		if _, err := rand.Read(padding); err != nil {
			return fmt.Errorf("record: padding randomness: %w", err)
		}
	}
	nonce := nonceFor(c.send.baseNonce, c.send.sequence)
	ciphertextLen := len(plaintext) + c.send.aead.Overhead()
	prefix := make([]byte, 4)
	binary.BigEndian.PutUint32(prefix, uint32(ciphertextLen))
	ciphertext := c.send.aead.Seal(nil, nonce, plaintext, c.additionalData(prefix))
	wirePrefix := append([]byte(nil), prefix...)
	if c.opts.MaskLength {
		mask := lengthMask(c.send.baseNonce, c.send.sequence, c.aad)
		for i := 0; i < 4; i++ {
			wirePrefix[i] ^= mask[i]
		}
	}
	if err := writeAll(w, wirePrefix); err != nil {
		return fmt.Errorf("record: write prefix: %w", err)
	}
	if err := writeAll(w, ciphertext); err != nil {
		return fmt.Errorf("record: write ciphertext: %w", err)
	}
	c.send.sequence++
	return nil
}

// Read authenticates and parses one record using bounded allocation.
func (c *Codec) Read(r io.Reader) (Record, error) {
	c.recv.mu.Lock()
	defer c.recv.mu.Unlock()
	if c.recv.sequence >= c.opts.MaxRecordsPerKey {
		return Record{}, ErrKeyExhausted
	}
	wirePrefix := make([]byte, 4)
	if _, err := io.ReadFull(r, wirePrefix); err != nil {
		return Record{}, fmt.Errorf("record: read prefix: %w", err)
	}
	prefix := append([]byte(nil), wirePrefix...)
	if c.opts.MaskLength {
		mask := lengthMask(c.recv.baseNonce, c.recv.sequence, c.aad)
		for i := 0; i < 4; i++ {
			prefix[i] ^= mask[i]
		}
	}
	ciphertextLen := int(binary.BigEndian.Uint32(prefix))
	minCiphertext := headerSize + c.recv.aead.Overhead()
	maxCiphertext := c.opts.MaxPlaintext + c.recv.aead.Overhead()
	if ciphertextLen < minCiphertext || ciphertextLen > maxCiphertext || !c.validCiphertextBucket(ciphertextLen) {
		return Record{}, fmt.Errorf("record: invalid ciphertext length %d", ciphertextLen)
	}
	ciphertext := make([]byte, ciphertextLen)
	if _, err := io.ReadFull(r, ciphertext); err != nil {
		return Record{}, fmt.Errorf("record: read ciphertext: %w", err)
	}
	nonce := nonceFor(c.recv.baseNonce, c.recv.sequence)
	plaintext, err := c.recv.aead.Open(nil, nonce, ciphertext, c.additionalData(prefix))
	if err != nil {
		return Record{}, errors.New("record: authentication failed")
	}
	if len(plaintext) < headerSize {
		return Record{}, errors.New("record: truncated plaintext")
	}
	recordType := Type(plaintext[0])
	if !validType(recordType) {
		return Record{}, errors.New("record: invalid protected type")
	}
	sequence := binary.BigEndian.Uint64(plaintext[12:20])
	if sequence != c.recv.sequence {
		return Record{}, errors.New("record: unexpected protected sequence")
	}
	payloadLen := int(binary.BigEndian.Uint32(plaintext[20:24]))
	if payloadLen > len(plaintext)-headerSize {
		return Record{}, errors.New("record: invalid protected payload length")
	}
	payload := make([]byte, payloadLen)
	copy(payload, plaintext[headerSize:headerSize+payloadLen])
	c.recv.sequence++
	return Record{Type: recordType, Flags: Flags(plaintext[1]), StreamID: binary.BigEndian.Uint64(plaintext[4:12]), Payload: payload}, nil
}

func lengthMask(baseNonce []byte, seq uint64, transcript [32]byte) [4]byte {
	h := sha256.New()
	h.Write(baseNonce)
	h.Write(transcript[:])
	var seqBuf [8]byte
	binary.BigEndian.PutUint64(seqBuf[:], seq)
	h.Write(seqBuf[:])
	sum := h.Sum(nil)
	var mask [4]byte
	copy(mask[:], sum[:4])
	return mask
}

func (c *Codec) bucketFor(required int) (int, bool) {
	i := sort.SearchInts(c.opts.Buckets, required)
	if i == len(c.opts.Buckets) {
		return 0, false
	}
	return c.opts.Buckets[i], true
}

func (c *Codec) validCiphertextBucket(ciphertextLen int) bool {
	plainLen := ciphertextLen - c.recv.aead.Overhead()
	i := sort.SearchInts(c.opts.Buckets, plainLen)
	return i < len(c.opts.Buckets) && c.opts.Buckets[i] == plainLen
}

func (c *Codec) additionalData(prefix []byte) []byte {
	aad := make([]byte, 0, len(c.aad)+len(prefix))
	aad = append(aad, c.aad[:]...)
	return append(aad, prefix...)
}

func nonceFor(base []byte, sequence uint64) []byte {
	nonce := append([]byte(nil), base...)
	start := len(nonce) - 8
	value := binary.BigEndian.Uint64(nonce[start:]) ^ sequence
	binary.BigEndian.PutUint64(nonce[start:], value)
	return nonce
}

func validType(t Type) bool { return t >= TypeOpen && t <= TypeKeyUpdate }

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}
