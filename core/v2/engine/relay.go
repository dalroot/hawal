package engine

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

var bufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 32*1024)
		return &b
	},
}

// Rule defines a port forwarding mapping.
type Rule struct {
	ListenPort string // e.g. "8080" or "0.0.0.0:8080"
	TargetAddr string // e.g. "127.0.0.1:8080"
}

// ParseRules parses strings of format "listenPort=targetAddr" or "port".
func ParseRules(rules []string) ([]Rule, map[string]string) {
	var parsed []Rule
	m := make(map[string]string)

	for _, r := range rules {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}

		parts := strings.Split(r, "=")
		if len(parts) == 2 {
			lPort := strings.TrimSpace(parts[0])
			tAddr := strings.TrimSpace(parts[1])
			parsed = append(parsed, Rule{ListenPort: lPort, TargetAddr: tAddr})
			m[lPort] = tAddr
			// Also index by bare port if listenPort is host:port
			if _, port, err := net.SplitHostPort(lPort); err == nil {
				m[port] = tAddr
			}
		} else if len(parts) == 1 {
			port := strings.TrimSpace(parts[0])
			tAddr := "127.0.0.1:" + port
			parsed = append(parsed, Rule{ListenPort: port, TargetAddr: tAddr})
			m[port] = tAddr
		}
	}
	return parsed, m
}

// ForwardListener listens on a local port and forwards connections to new v2 streams.
type ForwardListener struct {
	rule     Rule
	listener net.Listener
	noDelay  bool
}

func StartForwardListener(rule Rule, noDelay bool, getSession func() *Session) (*ForwardListener, error) {
	bindAddr := rule.ListenPort
	if !strings.Contains(bindAddr, ":") {
		bindAddr = "0.0.0.0:" + bindAddr
	}

	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return nil, fmt.Errorf("engine: listen on %s: %w", bindAddr, err)
	}

	fl := &ForwardListener{
		rule:     rule,
		listener: ln,
		noDelay:  noDelay,
	}

	go fl.serve(getSession)
	return fl, nil
}

func (fl *ForwardListener) Close() error {
	return fl.listener.Close()
}

func (fl *ForwardListener) serve(getSession func() *Session) {
	for {
		conn, err := fl.listener.Accept()
		if err != nil {
			return
		}

		if tcpConn, ok := conn.(*net.TCPConn); ok && fl.noDelay {
			_ = tcpConn.SetNoDelay(true)
			_ = tcpConn.SetKeepAlive(true)
			_ = tcpConn.SetKeepAlivePeriod(30 * time.Second)
		}

		go func(c net.Conn) {
			defer c.Close()
			sess := getSession()
			if sess == nil {
				log.Printf("[Hawal-v2] ⚠️ Dropping client connection on %s: session is not ready", fl.rule.ListenPort)
				return
			}

			// Open stream towards target
			stream, err := sess.OpenStream(fl.rule.TargetAddr)
			if err != nil {
				log.Printf("[Hawal-v2] Failed to open stream for %s: %v", fl.rule.TargetAddr, err)
				return
			}
			defer stream.Close()

			PipeBidirectional(c, stream)
		}(conn)
	}
}

// ServeEgress accepts incoming streams from session and pipes them to local target services.
func ServeEgress(ctx context.Context, session *Session, portMap map[string]string, noDelay bool) error {
	for {
		stream, err := session.AcceptStream(ctx)
		if err != nil {
			return err
		}

		go func(s *Stream) {
			defer s.Close()

			target := s.Target()
			if mapped, ok := portMap[target]; ok {
				target = mapped
			} else if !strings.Contains(target, ":") {
				target = "127.0.0.1:" + target
			}

			log.Printf("[Hawal-v2] 🔌 Serving egress stream #%d -> dialing %s", s.ID(), target)
			outConn, err := net.DialTimeout("tcp", target, 10*time.Second)
			if err != nil {
				log.Printf("[Hawal-v2] ❌ Egress dial to %s failed: %v", target, err)
				_ = s.Reset()
				return
			}
			defer outConn.Close()
			log.Printf("[Hawal-v2] 🟢 Egress connected to %s for stream #%d", target, s.ID())

			if tcpConn, ok := outConn.(*net.TCPConn); ok && noDelay {
				_ = tcpConn.SetNoDelay(true)
				_ = tcpConn.SetKeepAlive(true)
				_ = tcpConn.SetKeepAlivePeriod(30 * time.Second)
			}

			PipeBidirectional(outConn, s)
		}(stream)
	}
}

// PipeBidirectional pipes data between two endpoints using zero-allocation pooled buffers.
func PipeBidirectional(a, b io.ReadWriteCloser) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)
		_, _ = io.CopyBuffer(b, a, *bufPtr)
		_ = b.Close()
	}()

	go func() {
		defer wg.Done()
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)
		_, _ = io.CopyBuffer(a, b, *bufPtr)
		_ = a.Close()
	}()

	wg.Wait()
}
