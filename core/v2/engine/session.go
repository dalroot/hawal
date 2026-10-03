package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dalroot/hawal/core/v2/carrier"
	"github.com/dalroot/hawal/core/v2/mux"
	"github.com/dalroot/hawal/core/v2/record"
)

var (
	ErrSessionClosed = errors.New("engine: session closed")
)

// Session coordinates bidirectional multiplexed traffic over an authenticated carrier.Link.
type Session struct {
	link     carrier.Link
	codec    *record.Codec
	sched    *mux.Scheduler
	isServer bool

	streamsMu sync.RWMutex
	streams   map[uint64]*Stream

	nextStreamID uint64
	incoming     chan *Stream

	closed    chan struct{}
	closeOnce sync.Once
	closeErr  error

	ctx    context.Context
	cancel context.CancelFunc

	lastInboundActivity int64
}

func NewSession(link carrier.Link, codec *record.Codec, isServer bool) (*Session, error) {
	if link == nil || codec == nil {
		return nil, errors.New("engine: link and codec are required")
	}

	sched, err := mux.NewScheduler(mux.DefaultSchedulerLimits())
	if err != nil {
		return nil, fmt.Errorf("engine: create scheduler: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var initStreamID uint64
	if isServer {
		initStreamID = 0 // Server will allocate 2, 4, 6...
	} else {
		initStreamID = 1 // Client will allocate 3, 5, 7... (first is 1)
	}

	s := &Session{
		link:         link,
		codec:        codec,
		sched:        sched,
		isServer:     isServer,
		streams:      make(map[uint64]*Stream),
		nextStreamID: initStreamID,
		incoming:            make(chan *Stream, 128),
		closed:              make(chan struct{}),
		ctx:                 ctx,
		cancel:              cancel,
		lastInboundActivity: time.Now().UnixNano(),
	}

	go s.outboundPump()
	go s.inboundPump()
	go s.pingLoop()

	return s, nil
}

func (s *Session) OpenStream(target string) (*Stream, error) {
	select {
	case <-s.closed:
		return nil, ErrSessionClosed
	default:
	}

	var id uint64
	if s.isServer {
		id = atomic.AddUint64(&s.nextStreamID, 2)
	} else {
		// First stream will be 1, subsequent 3, 5, 7...
		cur := atomic.LoadUint64(&s.nextStreamID)
		if cur == 1 && atomic.CompareAndSwapUint64(&s.nextStreamID, 1, 3) {
			id = 1
		} else {
			id = atomic.AddUint64(&s.nextStreamID, 2)
		}
	}

	st := newStream(id, target, s)

	s.streamsMu.Lock()
	s.streams[id] = st
	s.streamsMu.Unlock()

	// Enqueue TypeOpen record
	openPayload := make([]byte, 1+len(target))
	openPayload[0] = byte(record.TypeOpen)
	copy(openPayload[1:], target)

	frame := mux.Frame{
		StreamID: id,
		Class:    mux.ClassControl,
		Payload:  openPayload,
	}

	if err := s.enqueueControl(frame); err != nil {
		s.removeStream(id)
		return nil, fmt.Errorf("engine: send open frame: %w", err)
	}

	return st, nil
}

func (s *Session) AcceptStream(ctx context.Context) (*Stream, error) {
	select {
	case <-s.closed:
		return nil, ErrSessionClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	case st, ok := <-s.incoming:
		if !ok {
			return nil, ErrSessionClosed
		}
		return st, nil
	}
}

func (s *Session) Close() error {
	return s.CloseWithError(nil)
}

func (s *Session) CloseWithError(err error) error {
	s.closeOnce.Do(func() {
		if err == nil {
			err = ErrSessionClosed
		}
		s.closeErr = err
		s.cancel()
		s.sched.Close()
		_ = s.link.Close()
		close(s.closed)

		s.streamsMu.Lock()
		for _, st := range s.streams {
			st.onRemoteReset()
		}
		s.streams = make(map[uint64]*Stream)
		s.streamsMu.Unlock()
	})
	return nil
}

func (s *Session) removeStream(id uint64) {
	s.streamsMu.Lock()
	delete(s.streams, id)
	s.streamsMu.Unlock()
}

func (s *Session) enqueueControl(frame mux.Frame) error {
	select {
	case <-s.closed:
		return ErrSessionClosed
	default:
	}

	for i := 0; i < 50; i++ {
		err := s.sched.TryEnqueue(frame)
		if err == nil {
			return nil
		}
		if errors.Is(err, io.ErrClosedPipe) {
			return ErrSessionClosed
		}
		time.Sleep(5 * time.Millisecond)
	}
	return mux.ErrQueueFull
}

func (s *Session) enqueueWithRetry(frame mux.Frame) error {
	select {
	case <-s.closed:
		return ErrSessionClosed
	default:
	}

	for i := 0; i < 100; i++ {
		err := s.sched.TryEnqueue(frame)
		if err == nil {
			return nil
		}
		if errors.Is(err, io.ErrClosedPipe) {
			return ErrSessionClosed
		}
		time.Sleep(5 * time.Millisecond)
	}
	return mux.ErrQueueFull
}

func (s *Session) outboundPump() {
	for {
		frame, err := s.sched.Next(s.ctx)
		if err != nil {
			_ = s.CloseWithError(err)
			return
		}

		if len(frame.Payload) == 0 {
			continue
		}

		recType := record.Type(frame.Payload[0])
		payload := frame.Payload[1:]

		rec := record.Record{
			Type:     recType,
			StreamID: frame.StreamID,
			Payload:  payload,
		}

		_ = s.link.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := s.codec.Write(s.link, rec); err != nil {
			_ = s.CloseWithError(fmt.Errorf("engine: write record: %w", err))
			return
		}
	}
}

func (s *Session) inboundPump() {
	for {
		rec, err := s.codec.Read(s.link)
		if err != nil {
			_ = s.CloseWithError(fmt.Errorf("engine: read record: %w", err))
			return
		}

		atomic.StoreInt64(&s.lastInboundActivity, time.Now().UnixNano())

		switch rec.Type {
		case record.TypeOpen:
			target := string(rec.Payload)
			st := newStream(rec.StreamID, target, s)

			s.streamsMu.Lock()
			s.streams[rec.StreamID] = st
			s.streamsMu.Unlock()

			select {
			case s.incoming <- st:
			case <-s.closed:
				return
			default:
				// Queue full: reset stream
				st.Reset()
			}

		case record.TypeData:
			s.streamsMu.RLock()
			st, ok := s.streams[rec.StreamID]
			s.streamsMu.RUnlock()
			if ok {
				st.onInboundData(rec.Payload)
			}

		case record.TypeHalfClose:
			s.streamsMu.RLock()
			st, ok := s.streams[rec.StreamID]
			s.streamsMu.RUnlock()
			if ok {
				st.onRemoteClose()
			}

		case record.TypeReset:
			s.streamsMu.RLock()
			st, ok := s.streams[rec.StreamID]
			s.streamsMu.RUnlock()
			if ok {
				st.onRemoteReset()
				s.removeStream(rec.StreamID)
			}

		case record.TypePing:
			// Peer is probing liveness; respond immediately with Pong
			pongPayload := []byte{byte(record.TypePong)}
			_ = s.enqueueControl(mux.Frame{
				StreamID: 0,
				Class:    mux.ClassControl,
				Payload:  pongPayload,
			})

		case record.TypePong:
			// Pong received; peer confirmed liveness (lastInboundActivity was updated)
		}
	}
}

func (s *Session) pingLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// 1. Send Ping
			payload := []byte{byte(record.TypePing)}
			if err := s.enqueueControl(mux.Frame{
				StreamID: 0,
				Class:    mux.ClassControl,
				Payload:  payload,
			}); err != nil {
				_ = s.CloseWithError(fmt.Errorf("engine: ping enqueue failed: %w", err))
				return
			}

			// 2. Dead-link autodetection: if no response/activity for > 45s, tear down session
			last := time.Unix(0, atomic.LoadInt64(&s.lastInboundActivity))
			if time.Since(last) > 45*time.Second {
				_ = s.CloseWithError(fmt.Errorf("engine: dead link detected (no activity for %v)", time.Since(last).Round(time.Second)))
				return
			}
		}
	}
}
