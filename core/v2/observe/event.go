// Package observe provides structured, secret-resistant v2 telemetry.
package observe

import (
	"sync"
	"time"
)

// Phase identifies the protocol stage at which an event occurred.
type Phase string

const (
	PhaseResolve   Phase = "resolve"
	PhaseRoute     Phase = "route"
	PhaseConnect   Phase = "connect"
	PhaseHandshake Phase = "handshake"
	PhasePayload   Phase = "payload"
	PhaseClose     Phase = "close"
)

// Kind identifies a stable, machine-readable event type.
type Kind string

const (
	KindAttempt Kind = "attempt"
	KindSuccess Kind = "success"
	KindFailure Kind = "failure"
	KindSample  Kind = "sample"
)

// Event intentionally has no arbitrary message field. This prevents tokens,
// keys, payloads, and full configuration documents from entering normal logs.
type Event struct {
	At         time.Time
	NodeID     string
	TunnelID   string
	CarrierID  string
	Phase      Phase
	Kind       Kind
	Code       string
	Duration   time.Duration
	BytesIn    uint64
	BytesOut   uint64
	Confidence float64
}

// Sink consumes an event without blocking the data plane.
type Sink interface {
	Record(Event)
}

// Ring is an in-memory bounded recorder. It overwrites the oldest entry when
// full, so diagnostic collection cannot grow memory without bound.
type Ring struct {
	mu       sync.RWMutex
	entries  []Event
	next     int
	capacity int
	full     bool
}

// NewRing constructs a recorder. Non-positive capacities are normalized to 1.
func NewRing(capacity int) *Ring {
	if capacity < 1 {
		capacity = 1
	}
	return &Ring{entries: make([]Event, capacity), capacity: capacity}
}

// Record adds an event and normalizes its timestamp and confidence range.
func (r *Ring) Record(event Event) {
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	if event.Confidence < 0 {
		event.Confidence = 0
	} else if event.Confidence > 1 {
		event.Confidence = 1
	}

	r.mu.Lock()
	r.entries[r.next] = event
	r.next = (r.next + 1) % r.capacity
	if r.next == 0 {
		r.full = true
	}
	r.mu.Unlock()
}

// Snapshot returns events in chronological insertion order.
func (r *Ring) Snapshot() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.full {
		result := make([]Event, r.next)
		copy(result, r.entries[:r.next])
		return result
	}

	result := make([]Event, r.capacity)
	n := copy(result, r.entries[r.next:])
	copy(result[n:], r.entries[:r.next])
	return result
}
