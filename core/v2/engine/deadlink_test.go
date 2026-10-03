package engine

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"net"
	"testing"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
	"github.com/dalroot/hawal/core/v2/record"
)

type dummyLink struct {
	net.Conn
	id          string
	established time.Time
}

func (d *dummyLink) ID() string               { return d.id }
func (d *dummyLink) Kind() carrier.Kind       { return carrier.KindTCP }
func (d *dummyLink) EstablishedAt() time.Time { return d.established }

func dummyCodec(t *testing.T) *record.Codec {
	t.Helper()
	key := bytes.Repeat([]byte{1}, 32)
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)
	nonce := bytes.Repeat([]byte{2}, aead.NonceSize())
	var transcript [32]byte
	c, err := record.New(aead, aead, nonce, nonce, transcript, record.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSessionWriteDeadlineTrigger(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	codec := dummyCodec(t)
	link := &dummyLink{
		Conn:        c1,
		id:          "dummy-1",
		established: time.Now(),
	}

	sess, err := NewSession(link, codec, false)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()

	// Verify session initialized lastInboundActivity
	if sess.lastInboundActivity == 0 {
		t.Fatal("lastInboundActivity not initialized")
	}
}
