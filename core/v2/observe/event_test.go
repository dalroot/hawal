package observe

import "testing"

func TestRingKeepsNewestInOrder(t *testing.T) {
	ring := NewRing(2)
	ring.Record(Event{Code: "one"})
	ring.Record(Event{Code: "two"})
	ring.Record(Event{Code: "three"})

	got := ring.Snapshot()
	if len(got) != 2 || got[0].Code != "two" || got[1].Code != "three" {
		t.Fatalf("unexpected snapshot: %#v", got)
	}
}

func TestRingNormalizesEvent(t *testing.T) {
	ring := NewRing(0)
	ring.Record(Event{Confidence: 4})

	got := ring.Snapshot()
	if len(got) != 1 || got[0].At.IsZero() || got[0].Confidence != 1 {
		t.Fatalf("event was not normalized: %#v", got)
	}
}
