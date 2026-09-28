package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/dalroot/hawal/core/v2/mux"
	"github.com/dalroot/hawal/core/v2/record"
)

const (
	maxChunkSize     = 16384
	streamChanBuffer = 256
)

var (
	ErrStreamClosed = errors.New("engine: stream closed")
	ErrStreamReset  = errors.New("engine: stream reset by peer")
)

// Stream implements net.Conn over a Hawal v2 multiplexed session.
type Stream struct {
	id       uint64
	target   string
	session  *Session
	inbound  chan []byte
	currBuf  []byte
	mu       sync.Mutex
	closed   bool
	closeErr error
	doneChan chan struct{}

	readDeadline  time.Time
	writeDeadline time.Time
}

func newStream(id uint64, target string, session *Session) *Stream {
	return &Stream{
		id:       id,
		target:   target,
		session:  session,
		inbound:  make(chan []byte, streamChanBuffer),
		doneChan: make(chan struct{}),
	}
}

func (s *Stream) ID() uint64     { return s.id }
func (s *Stream) Target() string { return s.target }

func (s *Stream) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	s.mu.Lock()
	if len(s.currBuf) > 0 {
		n := copy(p, s.currBuf)
		s.currBuf = s.currBuf[n:]
		s.mu.Unlock()
		return n, nil
	}

	if s.closed {
		err := s.closeErr
		if err == nil {
			err = io.EOF
		}
		s.mu.Unlock()
		return 0, err
	}

	deadline := s.readDeadline
	s.mu.Unlock()

	var timer *time.Timer
	var timeoutChan <-chan time.Time
	if !deadline.IsZero() {
		dur := time.Until(deadline)
		if dur <= 0 {
			return 0, context.DeadlineExceeded
		}
		timer = time.NewTimer(dur)
		defer timer.Stop()
		timeoutChan = timer.C
	}

	select {
	case <-s.session.closed:
		return 0, io.ErrClosedPipe
	case <-s.doneChan:
		s.mu.Lock()
		err := s.closeErr
		if err == nil {
			err = io.EOF
		}
		s.mu.Unlock()
		return 0, err
	case <-timeoutChan:
		return 0, context.DeadlineExceeded
	case chunk, ok := <-s.inbound:
		if !ok {
			s.mu.Lock()
			err := s.closeErr
			if err == nil {
				err = io.EOF
			}
			s.mu.Unlock()
			return 0, err
		}
		n := copy(p, chunk)
		if n < len(chunk) {
			s.mu.Lock()
			s.currBuf = append([]byte(nil), chunk[n:]...)
			s.mu.Unlock()
		}
		return n, nil
	}
}

func (s *Stream) Write(p []byte) (int, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	s.mu.Unlock()

	written := 0
	for len(p) > 0 {
		chunkSize := len(p)
		if chunkSize > maxChunkSize {
			chunkSize = maxChunkSize
		}
		chunk := p[:chunkSize]

		// Encode record.TypeData as first byte in payload
		payload := make([]byte, 1+len(chunk))
		payload[0] = byte(record.TypeData)
		copy(payload[1:], chunk)

		frame := mux.Frame{
			StreamID: s.id,
			Class:    mux.ClassInteractive,
			Payload:  payload,
		}

		if err := s.session.enqueueWithRetry(frame); err != nil {
			return written, fmt.Errorf("engine: write frame: %w", err)
		}

		written += chunkSize
		p = p[chunkSize:]
	}

	return written, nil
}

func (s *Stream) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.closeErr = io.EOF
	close(s.doneChan)
	s.mu.Unlock()

	// Notify peer of stream close
	payload := []byte{byte(record.TypeHalfClose)}
	_ = s.session.enqueueControl(mux.Frame{
		StreamID: s.id,
		Class:    mux.ClassControl,
		Payload:  payload,
	})

	s.session.removeStream(s.id)
	return nil
}

func (s *Stream) Reset() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.closeErr = ErrStreamReset
	close(s.doneChan)
	s.mu.Unlock()

	payload := []byte{byte(record.TypeReset)}
	_ = s.session.enqueueControl(mux.Frame{
		StreamID: s.id,
		Class:    mux.ClassControl,
		Payload:  payload,
	})

	s.session.removeStream(s.id)
	return nil
}

func (s *Stream) onInboundData(data []byte) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	owned := append([]byte(nil), data...)
	select {
	case s.inbound <- owned:
	case <-s.doneChan:
	case <-s.session.closed:
	default:
		// Queue full backpressure: wait briefly
		select {
		case s.inbound <- owned:
		case <-time.After(50 * time.Millisecond):
		case <-s.doneChan:
		case <-s.session.closed:
		}
	}
}

func (s *Stream) onRemoteClose() {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		s.closeErr = io.EOF
		close(s.doneChan)
	}
	s.mu.Unlock()
}

func (s *Stream) onRemoteReset() {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		s.closeErr = ErrStreamReset
		close(s.doneChan)
	}
	s.mu.Unlock()
}

// net.Conn interface implementation
type streamAddr struct{ target string }

func (a streamAddr) Network() string { return "hawal-v2" }
func (a streamAddr) String() string  { return a.target }

func (s *Stream) LocalAddr() net.Addr  { return streamAddr{target: fmt.Sprintf("stream:%d", s.id)} }
func (s *Stream) RemoteAddr() net.Addr { return streamAddr{target: s.target} }

func (s *Stream) SetDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readDeadline = t
	s.writeDeadline = t
	return nil
}

func (s *Stream) SetReadDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readDeadline = t
	return nil
}

func (s *Stream) SetWriteDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeDeadline = t
	return nil
}
