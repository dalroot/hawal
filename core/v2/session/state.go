// Package session owns the authenticated v2 session lifecycle.
package session

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type State uint8

const (
	StateCreated State = iota
	StateHandshaking
	StateActive
	StateMigrating
	StateDraining
	StateClosed
)

func (s State) String() string {
	switch s {
	case StateCreated:
		return "created"
	case StateHandshaking:
		return "handshaking"
	case StateActive:
		return "active"
	case StateMigrating:
		return "migrating"
	case StateDraining:
		return "draining"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

type Transition struct {
	From State
	To   State
	At   time.Time
}

// Machine serializes lifecycle transitions and preserves a bounded history.
type Machine struct {
	mu      sync.RWMutex
	state   State
	history []Transition
	limit   int
}

func NewMachine(historyLimit int) *Machine {
	if historyLimit < 1 {
		historyLimit = 16
	}
	return &Machine{state: StateCreated, limit: historyLimit}
}

func (m *Machine) State() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *Machine) Move(to State) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !allowed(m.state, to) {
		return fmt.Errorf("session: illegal transition %s -> %s", m.state, to)
	}
	transition := Transition{From: m.state, To: to, At: time.Now().UTC()}
	m.state = to
	if len(m.history) == m.limit {
		copy(m.history, m.history[1:])
		m.history[len(m.history)-1] = transition
	} else {
		m.history = append(m.history, transition)
	}
	return nil
}

func (m *Machine) History() []Transition {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Transition, len(m.history))
	copy(result, m.history)
	return result
}

func allowed(from, to State) bool {
	if to == StateClosed && from != StateClosed {
		return true
	}
	switch from {
	case StateCreated:
		return to == StateHandshaking
	case StateHandshaking:
		return to == StateActive
	case StateActive:
		return to == StateMigrating || to == StateDraining
	case StateMigrating:
		return to == StateActive || to == StateDraining
	case StateDraining:
		return to == StateClosed
	default:
		return false
	}
}

var ErrClosed = errors.New("session: closed")
