package secure

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"testing"
)

func testAEAD(t *testing.T) cipher.AEAD {
	t.Helper()
	block, err := aes.NewCipher(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	return aead
}

func TestSecretsDestroy(t *testing.T) {
	aead := testAEAD(t)
	secrets := Secrets{SendAEAD: aead, ReceiveAEAD: aead, SendNonce: bytes.Repeat([]byte{1}, 12), ReceiveNonce: bytes.Repeat([]byte{2}, 12)}
	secrets.Destroy()
	if secrets.SendAEAD != nil || secrets.ReceiveAEAD != nil {
		t.Fatal("AEAD references were retained")
	}
	if !bytes.Equal(secrets.SendNonce, make([]byte, 12)) || !bytes.Equal(secrets.ReceiveNonce, make([]byte, 12)) {
		t.Fatal("nonce material was not overwritten")
	}
}
