package mux

import (
	"context"
	"errors"
	"io"
	"sync"
)

// Window implements bounded byte credit for stream and connection flow
// control. Acquire blocks without spinning until credit is released.
type Window struct {
	mu        sync.Mutex
	available uint64
	limit     uint64
	closed    bool
	notify    chan struct{}
}

func NewWindow(limit uint64) (*Window, error) {
	if limit == 0 {
		return nil, errors.New("mux: window limit must be positive")
	}
	return &Window{available: limit, limit: limit, notify: make(chan struct{}, 1)}, nil
}

func (w *Window) Acquire(ctx context.Context, amount uint64) error {
	if amount == 0 {
		return nil
	}
	if amount > w.limit {
		return errors.New("mux: requested credit exceeds window limit")
	}
	for {
		w.mu.Lock()
		if w.closed {
			w.mu.Unlock()
			return io.ErrClosedPipe
		}
		if amount <= w.available {
			w.available -= amount
			w.mu.Unlock()
			return nil
		}
		notify := w.notify
		w.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-notify:
		}
	}
}

func (w *Window) Release(amount uint64) {
	if amount == 0 {
		return
	}
	w.mu.Lock()
	if amount > w.limit-w.available {
		w.available = w.limit
	} else {
		w.available += amount
	}
	w.signal()
	w.mu.Unlock()
}

func (w *Window) Available() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.available
}

func (w *Window) Close() {
	w.mu.Lock()
	w.closed = true
	w.signal()
	w.mu.Unlock()
}

func (w *Window) signal() {
	select {
	case w.notify <- struct{}{}:
	default:
	}
}
