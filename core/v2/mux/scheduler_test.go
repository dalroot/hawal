package mux

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestSchedulerBoundsAndOwnership(t *testing.T) {
	s, err := NewScheduler(SchedulerLimits{MaxFrames: 2, MaxBytes: 8, MaxPayload: 4, MaxControlFrames: 1})
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("one")
	if err := s.TryEnqueue(Frame{Class: ClassControl, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	payload[0] = 'X'
	if err := s.TryEnqueue(Frame{Class: ClassControl, Payload: []byte("two")}); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("control cap error = %v", err)
	}
	got, err := s.Next(context.Background())
	if err != nil || string(got.Payload) != "one" {
		t.Fatalf("Next() = %#v, %v", got, err)
	}
}

func TestSchedulerDropsExpiredDatagram(t *testing.T) {
	s, _ := NewScheduler(DefaultSchedulerLimits())
	_ = s.TryEnqueue(Frame{Class: ClassDatagram, Payload: []byte("old"), Expires: time.Now().Add(-time.Second)})
	_ = s.TryEnqueue(Frame{Class: ClassBulk, Payload: []byte("live")})
	got, err := s.Next(context.Background())
	if err != nil || string(got.Payload) != "live" {
		t.Fatalf("Next() = %#v, %v", got, err)
	}
	if frames, bytes := s.Depth(); frames != 0 || bytes != 0 {
		t.Fatalf("depth = %d/%d", frames, bytes)
	}
}

func TestSchedulerCloseUnblocksReader(t *testing.T) {
	s, _ := NewScheduler(DefaultSchedulerLimits())
	done := make(chan error, 1)
	go func() {
		_, err := s.Next(context.Background())
		done <- err
	}()
	s.Close()
	select {
	case err := <-done:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("Next() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Next did not unblock")
	}
}

func TestWindowBackpressure(t *testing.T) {
	w, _ := NewWindow(10)
	if err := w.Acquire(context.Background(), 10); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- w.Acquire(ctx, 5) }()
	w.Release(5)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := w.Available(); got != 0 {
		t.Fatalf("available = %d, want 0", got)
	}
}
