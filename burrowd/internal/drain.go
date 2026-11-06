package internal

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"burrow/protocol"
)

var ErrDrainTimeout = errors.New("request drain deadline exceeded; remaining streams interrupted")

// SetDrainTimeout is a startup-only option. Zero explicitly interrupts immediately.
func (s *Server) SetDrainTimeout(timeout time.Duration) error {
	if timeout < 0 || timeout > 15*time.Second {
		return errors.New("drain timeout must be between 0s and 15s")
	}
	s.drainTimeout = timeout
	return nil
}

func drainRejected(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Retry-After", "1")
	jsonError(w, "Server draining", http.StatusServiceUnavailable)
}

// inflightRequests reports whether the admission barrier still owns live
// request work. The map is the synchronous source of truth: entries are removed
// by each handler's release path before its WaitGroup counter drops.
func (s *Server) inflightRequests() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.requestCancels) > 0
}

func (s *Server) shutdown(parent context.Context, started chan<- struct{}) error {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	s.mu.Lock()
	s.closing = true // Same barrier as request/session admission, with no network I/O.
	s.stopIntake()
	for _, pending := range s.pendingRemoves {
		pending.timer.Stop()
	}
	s.pendingRemoves = make(map[string]*pendingRemoval)
	srv := s.httpServer
	deadline := time.Now().Add(s.drainTimeout)
	if end, ok := ctx.Deadline(); ok && end.Before(deadline) {
		deadline = end
	}
	notices := make([]*tunnelSession, 0, len(s.connections))
	for _, session := range s.connections {
		if session.lifecycle != nil {
			notices = append(notices, session)
		}
	}
	s.mu.Unlock()
	close(started)
	if srv != nil {
		srv.SetKeepAlivesEnabled(false)
	}
	// A fixed-size worker set, not one goroutine or serial timeout per tunnel.
	notified := make(chan struct{})
	noticeCtx, cancelNotices := context.WithDeadline(ctx, deadline)
	defer cancelNotices()
	go func() { s.notifyDraining(noticeCtx, notices, deadline); close(notified) }()

	drained := make(chan struct{})
	go func() { s.requests.Wait(); close(drained) }()
	var result error
	switch {
	case ctx.Err() != nil:
		result = ctx.Err()
	case s.drainTimeout <= 0:
		// Zero means no waiting budget. Only claim an interruption when an
		// admitted request is actually still in flight.
		if s.inflightRequests() {
			result = ErrDrainTimeout
		}
	default:
		timer := time.NewTimer(s.drainTimeout)
		select {
		case <-drained:
		case <-timer.C:
			// Requests that finished exactly at the boundary were not
			// interrupted; only in-flight work owned by the barrier was.
			if s.inflightRequests() {
				result = ErrDrainTimeout
			}
		case <-ctx.Done():
			result = ctx.Err()
		}
		timer.Stop()
	}
	s.mu.Lock()
	s.forceClosing = true
	sessions := make([]*tunnelSession, 0, len(s.connections))
	for id, session := range s.connections {
		sessions = append(sessions, session)
		delete(s.connections, id)
	}
	for _, stop := range s.requestCancels {
		stop()
	}
	upgrades := make([]net.Conn, 0, len(s.hijacked))
	for conn := range s.hijacked {
		upgrades = append(upgrades, conn)
	}
	s.mu.Unlock()
	// Closing yamux ends both normal and hijacked upgrade lifetimes. Never close
	// transports while holding the map/admission lock.
	<-notified
	for _, conn := range upgrades {
		conn.Close()
	}
	for _, session := range sessions {
		session.conn.Close()
	}
	cleanup, stopCleanup := context.WithTimeout(ctx, 5*time.Second)
	defer stopCleanup()
	if srv != nil {
		if err := srv.Shutdown(cleanup); err != nil {
			srv.Close()
			if result == nil {
				result = err
			}
		}
	}
	joined := make(chan struct{})
	go func() { s.handlers.Wait(); s.requests.Wait(); close(joined) }()
	select {
	case <-joined:
	case <-cleanup.Done():
		if result == nil {
			result = cleanup.Err()
		}
	}
	return result
}

// Preserve ResponseController's access to streaming/trailers/informational
// responses. Intercept only successful public upgrade hijacks for forced cleanup.
type drainResponseWriter struct {
	http.ResponseWriter
	server *Server
}

func (w *drainResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *drainResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, buffer, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err != nil {
		return nil, nil, err
	}
	tracked := &drainConn{Conn: conn, server: w.server}
	w.server.mu.Lock()
	forcing := w.server.forceClosing
	if !forcing {
		w.server.hijacked[tracked] = struct{}{}
	}
	w.server.mu.Unlock()
	if forcing {
		tracked.Close()
	}
	return tracked, buffer, nil
}

type drainConn struct {
	net.Conn
	server *Server
	once   sync.Once
	err    error
}

func (c *drainConn) Close() error {
	c.once.Do(func() { c.err = c.Conn.Close(); c.server.mu.Lock(); delete(c.server.hijacked, c); c.server.mu.Unlock() })
	return c.err
}

func (s *Server) notifyDraining(ctx context.Context, sessions []*tunnelSession, deadline time.Time) {
	var workers sync.WaitGroup
	work := make(chan *tunnelSession)
	for range min(8, len(sessions)) {
		workers.Go(func() {
			for session := range work {
				select {
				case <-session.lifecycleReady:
				case <-ctx.Done():
				}
				if ctx.Err() != nil || !time.Now().Before(deadline) {
					continue
				}
				end := minTime(deadline, time.Now().Add(time.Second))
				session.lifecycle.SetWriteDeadline(end)
				stop := context.AfterFunc(ctx, func() { session.lifecycle.SetWriteDeadline(time.Now()) })
				json.NewEncoder(session.lifecycle).Encode(protocol.Lifecycle{Version: 1, Type: "draining", TunnelID: session.tunnelID, Generation: session.id, Reason: "server_shutdown", RemainingMS: max(0, time.Until(deadline).Milliseconds()), NoticeID: rand.Text()})
				stop()
			}
		})
	}
	for _, session := range sessions {
		work <- session
	}
	close(work)
	workers.Wait()
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	s.mu.RLock()
	serving := !s.closing
	s.mu.RUnlock()
	ok := serving && s.registry != nil && s.redisReady(ctx)
	s.mu.RLock()
	ok = ok && !s.closing
	s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	if !ok {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unavailable"}`))
		return
	}
	w.Write([]byte(`{"status":"ok"}`))
}

// One in-flight bounded Redis read; expired successes are never stale fallbacks.
func (s *Server) redisReady(ctx context.Context) bool {
	s.readyMu.Lock()
	if time.Since(s.readyAt) < time.Second {
		ok := s.readyOK
		s.readyMu.Unlock()
		return ok
	}
	if flight := s.readyFlight; flight != nil {
		s.readyMu.Unlock()
		select {
		case <-flight:
		case <-ctx.Done():
			return false
		}
		s.readyMu.Lock()
		ok := time.Since(s.readyAt) < time.Second && s.readyOK
		s.readyMu.Unlock()
		return ok
	}
	flight := make(chan struct{})
	s.readyFlight = flight
	s.readyMu.Unlock()
	err := s.registry.redis.client.Ping(ctx).Err()
	s.readyMu.Lock()
	s.readyOK = err == nil && ctx.Err() == nil
	s.readyAt = time.Now()
	s.readyFlight = nil
	close(flight)
	ok := s.readyOK
	s.readyMu.Unlock()
	return ok
}
