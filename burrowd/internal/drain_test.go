package internal

import (
	"context"
	"errors"
	"github.com/gorilla/websocket"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestShutdownPreservesAdmittedPOSTAndRejectsLateTraffic(t *testing.T) {
	s := NewServer(nil, "example.test", false)
	entered, release := make(chan struct{}), make(chan struct{})
	proxyFixture(t, s, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		close(entered)
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		io.WriteString(w, string(body))
	}))
	proxy := httptest.NewServer(s)
	defer proxy.Close()
	client := &http.Client{Timeout: 2 * time.Second}
	defer client.CloseIdleConnections()
	response := make(chan error, 1)
	go func() {
		req, _ := http.NewRequest("POST", proxy.URL+"/effect", strings.NewReader("exactly-once"))
		req.Host = "probe.example.test"
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			body, readErr := io.ReadAll(resp.Body)
			err = readErr
			if resp.StatusCode != 200 || string(body) != "exactly-once" {
				err = errors.New("admitted POST interrupted")
			}
		}
		response <- err
	}()
	<-entered
	shutdown := make(chan error, 1)
	go func() { shutdown <- s.Shutdown(t.Context()) }()
	<-s.intake.Done()
	select {
	case err := <-shutdown:
		close(release)
		t.Fatalf("premature shutdown: %v", err)
	default:
	}
	req := httptest.NewRequest("POST", "http://probe.example.test/late", strings.NewReader("must-not-forward"))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != 503 || w.Header().Get("Retry-After") == "" || w.Header().Get("Cache-Control") != "no-store" {
		close(release)
		t.Fatalf("late admission: status=%d headers=%v", w.Code, w.Header())
	}
	close(release)
	if err := <-response; err != nil {
		t.Fatal(err)
	}
	if err := <-shutdown; err != nil {
		t.Fatal(err)
	}
}

func TestDrainDeadlineInterruptsStreamingAndIsShared(t *testing.T) {
	s := NewServer(nil, "example.test", false)
	if err := s.SetDrainTimeout(25 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	proxyFixture(t, s, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: admitted\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	proxy := httptest.NewServer(s)
	defer proxy.Close()
	req, _ := http.NewRequest("GET", proxy.URL+"/events", nil)
	req.Host = "probe.example.test"
	client := &http.Client{Timeout: time.Second}
	defer client.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	start := time.Now()
	if err := s.Shutdown(t.Context()); !errors.Is(err, ErrDrainTimeout) {
		t.Fatalf("deadline outcome: %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("deadline was not bounded")
	}
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Fatal("interrupted SSE reported clean completion")
	}
	if err := s.Shutdown(t.Context()); !errors.Is(err, ErrDrainTimeout) {
		t.Fatalf("repeated outcome: %v", err)
	}
}

func TestReadinessFailureReplacesExpiredSuccess(t *testing.T) {
	s, _, redis := privacyServer(t)
	get := func() int {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "http://example.test/ready", nil))
		return w.Code
	}
	if code := get(); code != 200 {
		t.Fatalf("ready status %d", code)
	}
	redis.SetError("private failure")
	s.readyMu.Lock()
	s.readyAt = time.Now().Add(-2 * time.Second)
	s.readyMu.Unlock()
	if code := get(); code != 503 {
		t.Fatalf("stale success reused: %d", code)
	}
}

func TestDrainingRegistrationDoesNotReserve(t *testing.T) {
	s, _, redis := privacyServer(t)
	if err := s.Shutdown(t.Context()); err != nil {
		t.Fatal(err)
	}
	before := len(redis.Keys())
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://example.test/tunnels/connect?tunnel_id=late&port=3000", nil)
	req.Header.Set("X-Tunnel-Token", "synthetic-owner")
	s.ServeHTTP(w, req)
	if w.Code != 503 || len(redis.Keys()) != before {
		t.Fatalf("draining admission changed registry: status=%d", w.Code)
	}
}

func TestReadyFailsClosedButHealthRemainsLive(t *testing.T) {
	s, _, redis := privacyServer(t)
	check := func(path string, status int) {
		t.Helper()
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "http://example.test"+path, nil))
		if w.Code != status || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body)
		}
		if strings.Contains(w.Body.String(), "redis") {
			t.Fatal("readiness leaked dependency details")
		}
	}
	check("/ready", 200)
	if err := s.Shutdown(t.Context()); err != nil {
		t.Fatal(err)
	}
	redis.SetError("private redis error")
	check("/ready", 503)
	check("/health", 200)
}

func TestShutdownSharesFirstResult(t *testing.T) {
	s := NewServer(nil, "example.test", false)
	entered := make(chan struct{})
	proxyFixture(t, s, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
	}))
	proxy := httptest.NewServer(s)
	defer proxy.Close()
	go func() {
		req, _ := http.NewRequest("GET", proxy.URL, nil)
		req.Host = "probe.example.test"
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}()
	<-entered
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	first := s.Shutdown(ctx)
	second := s.Shutdown(t.Context())
	if !errors.Is(first, context.Canceled) || !errors.Is(second, context.Canceled) {
		t.Fatalf("inconsistent shutdown: first=%v second=%v", first, second)
	}
}

func TestDrainDeadlineClosesHijackedWebSocket(t *testing.T) {
	s := NewServer(nil, "example.test", false)
	s.SetDrainTimeout(25 * time.Millisecond)
	proxyFixture(t, s, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := u.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			kind, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if conn.WriteMessage(kind, data) != nil {
				return
			}
		}
	}))
	proxy := httptest.NewServer(s)
	defer proxy.Close()
	dialer := websocket.Dialer{NetDialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp", strings.TrimPrefix(proxy.URL, "http://"))
	}}
	conn, _, err := dialer.DialContext(t.Context(), "ws://probe.example.test/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.WriteMessage(websocket.TextMessage, []byte("admitted"))
	if _, body, err := conn.ReadMessage(); err != nil || string(body) != "admitted" {
		t.Fatalf("upgrade failed: %q %v", body, err)
	}
	if err := s.Shutdown(t.Context()); !errors.Is(err, ErrDrainTimeout) {
		t.Fatalf("upgrade lifetime was not drained: %v", err)
	}
	s.mu.RLock()
	active := len(s.requestCancels)
	s.mu.RUnlock()
	if active != 0 {
		t.Fatalf("shutdown retained %d hijacked request handlers", active)
	}
	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("hijacked upgrade survived drain deadline")
	} else if failure, ok := err.(net.Error); ok && failure.Timeout() {
		t.Fatal("hijacked upgrade only stopped at the test's read timeout")
	}
}

func TestShutdownWaiterCanCancelWithoutClaimingCompletion(t *testing.T) {
	s := NewServer(nil, "example.test", false)
	s.requests.Add(1)
	first := make(chan error, 1)
	go func() { first <- s.Shutdown(t.Context()) }()
	<-s.intake.Done()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	second := make(chan error, 1)
	go func() { second <- s.Shutdown(ctx) }()
	select {
	case err := <-second:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled waiter: %v", err)
		}
	case <-time.After(50 * time.Millisecond):
		s.requests.Done()
		<-first
		t.Fatal("shutdown waiter ignored its cancellation")
	}
	s.requests.Done()
	if err := <-first; err != nil {
		t.Fatalf("cancelled waiter changed shared outcome: %v", err)
	}
}

func TestImmediateDrainWithoutRequestsIsClean(t *testing.T) {
	s := NewServer(nil, "example.test", false)
	if err := s.SetDrainTimeout(0); err != nil {
		t.Fatal(err)
	}
	if err := s.Shutdown(t.Context()); err != nil {
		t.Fatalf("idle immediate drain claimed an interruption: %v", err)
	}
}
