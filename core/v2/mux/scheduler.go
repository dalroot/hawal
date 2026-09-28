// Package mux provides bounded scheduling primitives for v2 streams.
package mux

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

type Class uint8

const (
	ClassControl Class = iota + 1
	ClassInteractive
	ClassDatagram
	ClassBulk
)

var (
	ErrQueueFull     = errors.New("mux: queue is full")
	ErrFrameTooLarge = errors.New("mux: frame exceeds payload limit")
	ErrInvalidClass  = errors.New("mux: invalid stream class")
)

type Frame struct {
	StreamID uint64
	Class    Class
	Payload  []byte
	Expires  time.Time
}

type SchedulerLimits struct {
	MaxFrames        int
	MaxBytes         int
	MaxPayload       int
	MaxControlFrames int
}

func DefaultSchedulerLimits() SchedulerLimits {
	return SchedulerLimits{MaxFrames: 4096, MaxBytes: 16 << 20, MaxPayload: 64 << 10, MaxControlFrames: 128}
}

// Scheduler uses weighted service while enforcing global and control-plane
// bounds. Enqueue copies payload ownership into the scheduler.
type Scheduler struct {
	mu       sync.Mutex
	queues   map[Class][]Frame
	limits   SchedulerLimits
	frames   int
	bytes    int
	cursor   int
	closed   bool
	notify   chan struct{}
	schedule []Class
}

func NewScheduler(limits SchedulerLimits) (*Scheduler, error) {
	if limits.MaxFrames < 1 || limits.MaxBytes < 1 || limits.MaxPayload < 1 || limits.MaxControlFrames < 1 || limits.MaxControlFrames > limits.MaxFrames {
		return nil, errors.New("mux: invalid scheduler limits")
	}
	return &Scheduler{
		queues:   make(map[Class][]Frame, 4),
		limits:   limits,
		notify:   make(chan struct{}, 1),
		schedule: []Class{ClassControl, ClassInteractive, ClassControl, ClassDatagram, ClassInteractive, ClassControl, ClassBulk, ClassDatagram, ClassInteractive},
	}, nil
}

func (s *Scheduler) TryEnqueue(frame Frame) error {
	if !validClass(frame.Class) {
		return ErrInvalidClass
	}
	if len(frame.Payload) > s.limits.MaxPayload {
		return ErrFrameTooLarge
	}
	owned := frame
	owned.Payload = append([]byte(nil), frame.Payload...)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return io.ErrClosedPipe
	}
	if s.frames == s.limits.MaxFrames || len(owned.Payload) > s.limits.MaxBytes-s.bytes {
		return ErrQueueFull
	}
	if owned.Class == ClassControl && len(s.queues[ClassControl]) >= s.limits.MaxControlFrames {
		return ErrQueueFull
	}
	s.queues[owned.Class] = append(s.queues[owned.Class], owned)
	s.frames++
	s.bytes += len(owned.Payload)
	s.signal()
	return nil
}

func (s *Scheduler) Next(ctx context.Context) (Frame, error) {
	for {
		s.mu.Lock()
		s.dropExpiredDatagrams(time.Now())
		if frame, ok := s.take(); ok {
			s.mu.Unlock()
			return frame, nil
		}
		if s.closed {
			s.mu.Unlock()
			return Frame{}, io.EOF
		}
		notify := s.notify
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return Frame{}, ctx.Err()
		case <-notify:
		}
	}
}

func (s *Scheduler) Close() {
	s.mu.Lock()
	s.closed = true
	s.signal()
	s.mu.Unlock()
}

func (s *Scheduler) Depth() (frames, bytes int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frames, s.bytes
}

func (s *Scheduler) take() (Frame, bool) {
	for checked := 0; checked < len(s.schedule); checked++ {
		class := s.schedule[s.cursor]
		s.cursor = (s.cursor + 1) % len(s.schedule)
		queue := s.queues[class]
		if len(queue) == 0 {
			continue
		}
		frame := queue[0]
		queue[0] = Frame{}
		s.queues[class] = queue[1:]
		s.frames--
		s.bytes -= len(frame.Payload)
		return frame, true
	}
	return Frame{}, false
}

func (s *Scheduler) dropExpiredDatagrams(now time.Time) {
	queue := s.queues[ClassDatagram]
	kept := queue[:0]
	for _, frame := range queue {
		if !frame.Expires.IsZero() && !frame.Expires.After(now) {
			s.frames--
			s.bytes -= len(frame.Payload)
			continue
		}
		kept = append(kept, frame)
	}
	for i := len(kept); i < len(queue); i++ {
		queue[i] = Frame{}
	}
	s.queues[ClassDatagram] = kept
}

func (s *Scheduler) signal() {
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

func validClass(class Class) bool { return class >= ClassControl && class <= ClassBulk }
