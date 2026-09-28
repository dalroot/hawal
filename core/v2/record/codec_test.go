package record

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"testing"
)

func newAEAD(t *testing.T, fill byte) cipher.AEAD {
	t.Helper()
	key := bytes.Repeat([]byte{fill}, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	return aead
}

func codecPair(t *testing.T, options Options) (*Codec, *Codec) {
	t.Helper()
	c2s, s2c := newAEAD(t, 1), newAEAD(t, 2)
	cNonce := bytes.Repeat([]byte{3}, c2s.NonceSize())
	sNonce := bytes.Repeat([]byte{4}, s2c.NonceSize())
	var transcript [32]byte
	copy(transcript[:], "bound-to-secure-handshake")
	client, err := New(c2s, s2c, cNonce, sNonce, transcript, options)
	if err != nil {
		t.Fatal(err)
	}
	server, err := New(s2c, c2s, sNonce, cNonce, transcript, options)
	if err != nil {
		t.Fatal(err)
	}
	return client, server
}

func TestCodecRoundTrip(t *testing.T) {
	client, server := codecPair(t, DefaultOptions())
	var wire bytes.Buffer
	want := Record{Type: TypeData, Flags: FlagFinal, StreamID: 42, Payload: []byte("secret payload")}
	if err := client.Write(&wire, want); err != nil {
		t.Fatal(err)
	}
	got, err := server.Read(&wire)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != want.Type || got.Flags != want.Flags || got.StreamID != want.StreamID || !bytes.Equal(got.Payload, want.Payload) {
		t.Fatalf("Read() = %#v, want %#v", got, want)
	}
}

func TestCodecUsesBucketsAndNoMagicPrefix(t *testing.T) {
	opts := DefaultOptions()
	opts.MaskLength = false
	client, _ := codecPair(t, opts)
	var first, second bytes.Buffer
	if err := client.Write(&first, Record{Type: TypeData, Payload: []byte{1}}); err != nil {
		t.Fatal(err)
	}
	if err := client.Write(&second, Record{Type: TypeData, Payload: bytes.Repeat([]byte{2}, 100)}); err != nil {
		t.Fatal(err)
	}
	firstLen := binary.BigEndian.Uint32(first.Bytes()[:4])
	secondLen := binary.BigEndian.Uint32(second.Bytes()[:4])
	if firstLen != secondLen {
		t.Fatalf("payloads in same bucket leaked different lengths: %d != %d", firstLen, secondLen)
	}
	if first.Len() != second.Len() {
		t.Fatalf("payloads in same bucket leaked different wire lengths: %d != %d", first.Len(), second.Len())
	}
	if bytes.Contains(first.Bytes(), []byte("HWL1")) {
		t.Fatal("legacy magic leaked into v2 record")
	}
}

func TestCodecLengthMaskingConcealsPrefix(t *testing.T) {
	opts := DefaultOptions()
	opts.MaskLength = true
	client, server := codecPair(t, opts)
	var wire bytes.Buffer
	want := Record{Type: TypeData, Flags: FlagFinal, StreamID: 10, Payload: []byte("masked wire payload")}
	if err := client.Write(&wire, want); err != nil {
		t.Fatal(err)
	}
	if wire.Bytes()[0] == 0x00 && wire.Bytes()[1] == 0x00 {
		t.Fatalf("masked wire prefix leaked zero-padded length: %x", wire.Bytes()[:4])
	}
	got, err := server.Read(&wire)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Payload, want.Payload) {
		t.Fatalf("got %q, want %q", got.Payload, want.Payload)
	}
}

func TestCodecRejectsTamperingAndOversizeBeforeAllocation(t *testing.T) {
	client, server := codecPair(t, DefaultOptions())
	var wire bytes.Buffer
	if err := client.Write(&wire, Record{Type: TypeData, Payload: []byte("hello")}); err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte(nil), wire.Bytes()...)
	tampered[len(tampered)-1] ^= 0xff
	if _, err := server.Read(bytes.NewReader(tampered)); err == nil {
		t.Fatal("tampered ciphertext accepted")
	}

	oversize := make([]byte, 4)
	binary.BigEndian.PutUint32(oversize, uint32(DefaultOptions().MaxPlaintext+1024))
	_, freshServer := codecPair(t, DefaultOptions())
	if _, err := freshServer.Read(bytes.NewReader(oversize)); err == nil {
		t.Fatal("oversized record accepted")
	}
}

func TestCodecEnforcesKeyLifetime(t *testing.T) {
	opts := DefaultOptions()
	opts.MaxRecordsPerKey = 1
	client, _ := codecPair(t, opts)
	var wire bytes.Buffer
	if err := client.Write(&wire, Record{Type: TypePing}); err != nil {
		t.Fatal(err)
	}
	if err := client.Write(&wire, Record{Type: TypePing}); err != ErrKeyExhausted {
		t.Fatalf("second Write() error = %v, want %v", err, ErrKeyExhausted)
	}
}
