package policy

import "testing"

func TestDefaultsAreValid(t *testing.T) {
	if err := Defaults().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsUnboundedQueue(t *testing.T) {
	limits := Defaults()
	limits.MaxQueuedBytes = 2 << 30
	if err := limits.Validate(); err == nil {
		t.Fatal("unsafe queue accepted")
	}
}
