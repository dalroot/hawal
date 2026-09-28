package session

import (
	"net"
	"testing"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
)

type fakeLink struct {
	net.Conn
	id   string
	kind carrier.Kind
}

func (f *fakeLink) ID() string               { return f.id }
func (f *fakeLink) Kind() carrier.Kind       { return f.kind }
func (f *fakeLink) EstablishedAt() time.Time { return time.Now() }

func linkPair(id string) (*fakeLink, net.Conn) {
	a, b := net.Pipe()
	return &fakeLink{Conn: a, id: id, kind: carrier.KindTCP}, b
}

func TestPathSetMigrationDoesNotCloseOldPath(t *testing.T) {
	paths, _ := NewPathSet(2)
	first, firstPeer := linkPair("first")
	defer first.Close()
	defer firstPeer.Close()
	second, secondPeer := linkPair("second")
	defer second.Close()
	defer secondPeer.Close()

	if added, err := paths.Attach(first); err != nil || !added {
		t.Fatalf("Attach(first) = %v, %v", added, err)
	}
	_, generation, _ := paths.Snapshot()
	if added, err := paths.Attach(second); err != nil || !added {
		t.Fatalf("Attach(second) = %v, %v", added, err)
	}
	_, generation, _ = paths.Snapshot()
	if _, err := paths.Promote("second", generation); err != nil {
		t.Fatal(err)
	}
	primary, generation, snapshot := paths.Snapshot()
	if primary != "second" || len(snapshot) != 2 {
		t.Fatalf("unexpected snapshot: %q %d %#v", primary, generation, snapshot)
	}
	if _, err := paths.Promote("second", generation-1); err != nil {
		t.Fatalf("idempotent promotion failed: %v", err)
	}
	if _, err := paths.Detach("second"); err != ErrDetachPrimary {
		t.Fatalf("Detach(primary) = %v", err)
	}
	readDone := make(chan error, 1)
	go func() {
		buffer := make([]byte, 1)
		_, err := firstPeer.Read(buffer)
		readDone <- err
	}()
	if _, err := first.Write([]byte("x")); err != nil {
		t.Fatalf("old path was closed during migration: %v", err)
	}
	if err := <-readDone; err != nil {
		t.Fatalf("old path peer could not read: %v", err)
	}
}
