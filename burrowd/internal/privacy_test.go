package internal

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"burrow/protocol"
	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

func privacyServer(t *testing.T) (*Server, *httptest.Server, *miniredis.Miniredis) {
	t.Helper()
	rdb := miniredis.RunT(t)
	redis, err := NewRedisClient("redis://" + rdb.Addr())
	if err != nil {
		t.Fatal(err)
	}
	s := NewServer(NewTunnelRegistry(redis, "example.test", false), "example.test", false)
	s.gracePeriod = time.Hour // Cleanup races are driven explicitly by tests.
	proxy := httptest.NewServer(s)
	t.Cleanup(func() {
		s.Shutdown(context.Background())
		proxy.Close()
		redis.Close()
	})
	return s, proxy, rdb
}

func dialOwner(t *testing.T, proxy *httptest.Server, id, token string, want int) *yamux.Session {
	t.Helper()
	headers := http.Header{}
	if token != "" {
		headers.Set("X-Tunnel-Token", token)
	}
	conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(proxy.URL, "http")+"/tunnels/connect?tunnel_id="+id+"&port=3000", headers)
	if resp == nil {
		t.Fatalf("dial %s: %v", id, err)
	}
	if resp.StatusCode != want {
		t.Fatalf("dial %s: HTTP %d, want %d (%v)", id, resp.StatusCode, want, err)
	}
	if err != nil {
		resp.Body.Close()
		return nil
	}
	session, err := yamux.Client(&protocol.WsConn{Conn: conn}, nil)
	if err != nil {
		conn.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close(); conn.Close() })
	return session
}

func apiRequest(t *testing.T, proxy *httptest.Server, method, path, token string, want int) []byte {
	t.Helper()
	req, _ := http.NewRequest(method, proxy.URL+path, nil)
	if token != "" {
		req.Header.Set("X-Tunnel-Token", token)
	}
	resp, err := proxy.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != want {
		t.Fatalf("%s %s: HTTP %d, want %d: %s", method, path, resp.StatusCode, want, data)
	}
	if want != http.StatusNotFound && resp.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("%s must not cache private responses", path)
	}
	return data
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for session lifecycle")
		}
		time.Sleep(time.Millisecond)
	}
}

func activeSession(t *testing.T, s *Server, id string) *tunnelSession {
	t.Helper()
	var current *tunnelSession
	waitFor(t, func() bool { s.mu.RLock(); defer s.mu.RUnlock(); current = s.connections[id]; return current != nil })
	return current
}

func TestOwnerIsolationAndPrivateViews(t *testing.T) {
	// A token signed with the former server secret has no privileged meaning.
	t.Setenv("BURROW_SECRET", "former-server-secret")
	ts := fmt.Sprint(time.Now().Unix())
	mac := hmac.New(sha256.New, []byte("former-server-secret"))
	mac.Write([]byte("burrow-auth:" + ts))
	formerAdmin := ts + "." + base64.URLEncoding.EncodeToString(mac.Sum(nil))
	s, proxy, _ := privacyServer(t)
	a := dialOwner(t, proxy, "owner-a", "credential-a", http.StatusSwitchingProtocols)
	dialOwner(t, proxy, "owner-a-second", "credential-a", http.StatusSwitchingProtocols)
	dialOwner(t, proxy, "owner-b", "credential-b", http.StatusSwitchingProtocols)
	original := activeSession(t, s, "owner-a")
	activeSession(t, s, "owner-a-second")
	activeSession(t, s, "owner-b")
	s.addLog("owner-a", original, &protocol.TunnelLog{Path: "/private-a", IP: "192.0.2.1"})
	for _, token := range []string{"", "credential-a", formerAdmin} {
		body := apiRequest(t, proxy, "GET", "/health", token, http.StatusOK)
		if string(body) != `{"status":"ok"}` {
			t.Fatalf("public health leaked metadata: %s", body)
		}
	}
	for _, token := range []string{"", "credential-b", formerAdmin} {
		apiRequest(t, proxy, "GET", "/tunnels/owner-a/logs", token, http.StatusUnauthorized)
		apiRequest(t, proxy, "DELETE", "/tunnels/owner-a", token, http.StatusUnauthorized)
	}
	apiRequest(t, proxy, "GET", "/tunnels/owner-a/logs?token=credential-a", "", http.StatusUnauthorized)
	body := apiRequest(t, proxy, "GET", "/tunnels/owner-a/logs", "credential-a", http.StatusOK)
	if !strings.Contains(string(body), "/private-a") {
		t.Fatalf("owner lost logs: %s", body)
	}
	var entries []*protocol.TunnelLog
	if err := json.Unmarshal(body, &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID == "" {
		t.Fatalf("missing stable log ID: %s", body)
	}
	firstID := entries[0].ID
	s.addLog("owner-a", original, &protocol.TunnelLog{Path: "/private-a", IP: "192.0.2.1"})
	if err := json.Unmarshal(apiRequest(t, proxy, "GET", "/tunnels/owner-a/logs", "credential-a", http.StatusOK), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].ID != firstID || entries[1].ID == "" || entries[1].ID == firstID {
		t.Fatalf("log IDs must be stable and distinguish identical request fields: %+v", entries)
	}
	for _, test := range []struct {
		token string
		ids   []string
	}{
		{"credential-a", []string{"owner-a", "owner-a-second"}}, {"credential-b", []string{"owner-b"}},
		{"unrelated", nil}, {formerAdmin, nil},
	} {
		var result protocol.TunnelList
		body := apiRequest(t, proxy, "GET", "/tunnels", test.token, http.StatusOK)
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), "credential-") || strings.Contains(string(body), hashToken("credential-a")) {
			t.Fatal("list leaked owner credentials")
		}
		if len(result.Tunnels) != len(test.ids) || result.Count != len(test.ids) {
			t.Fatalf("owner list/count leaked tunnels: %+v", result)
		}
		for i, id := range test.ids {
			if result.Tunnels[i].TunnelID != id {
				t.Fatalf("wrong owner list: %+v", result)
			}
		}
	}
	for _, path := range []string{"/tunnels", "/tunnels?all=true"} {
		apiRequest(t, proxy, "GET", path, "", http.StatusUnauthorized)
	}
	for _, token := range []string{"credential-a", formerAdmin} {
		apiRequest(t, proxy, "GET", "/tunnels?all=true", token, http.StatusBadRequest)
		for _, path := range []string{"/admin/tunnels", "/admin/stats", "/stats", "/logs/owner-a", "/tunnel/ws"} {
			apiRequest(t, proxy, "GET", path, token, http.StatusNotFound)
		}
	}
	dialOwner(t, proxy, "owner-a", "credential-b", http.StatusConflict)
	dialOwner(t, proxy, "owner-a", formerAdmin, http.StatusConflict)
	dialOwner(t, proxy, "owner-a", "", http.StatusUnauthorized)
	if activeSession(t, s, "owner-a") != original || a.IsClosed() {
		t.Fatal("denied takeover changed the existing connection")
	}
	apiRequest(t, proxy, "DELETE", "/tunnels/owner-a", "credential-a", http.StatusNoContent)
	waitFor(t, a.IsClosed)
	apiRequest(t, proxy, "GET", "/tunnels/owner-a/logs", "credential-a", http.StatusUnauthorized)
	dialOwner(t, proxy, "owner-a", "credential-a", http.StatusGone)
	dialOwner(t, proxy, "owner-a", "credential-b", http.StatusConflict)
	apiRequest(t, proxy, "DELETE", "/tunnels/owner-b", "credential-b", http.StatusNoContent)
}

func TestReconnectAndStaleCleanupCannotReplaceOwnership(t *testing.T) {
	s, proxy, _ := privacyServer(t)
	first := dialOwner(t, proxy, "reconnecting", "owner", http.StatusSwitchingProtocols)
	original := activeSession(t, s, "reconnecting")
	s.addLog("reconnecting", original, &protocol.TunnelLog{Path: "/before-reconnect"})
	first.Close()
	var pending *pendingRemoval
	waitFor(t, func() bool {
		s.mu.RLock()
		defer s.mu.RUnlock()
		pending = s.pendingRemoves["reconnecting"]
		return pending != nil
	})
	var disconnected protocol.TunnelList
	if err := json.Unmarshal(apiRequest(t, proxy, "GET", "/tunnels", "owner", http.StatusOK), &disconnected); err != nil {
		t.Fatal(err)
	}
	if disconnected.Count != 0 || len(disconnected.Tunnels) != 0 {
		t.Fatalf("disconnected session counted as connected: %+v", disconnected)
	}
	second := dialOwner(t, proxy, "reconnecting", "owner", http.StatusSwitchingProtocols)
	replacement := activeSession(t, s, "reconnecting")
	if replacement == original {
		t.Fatal("connection was not replaced")
	}
	// Emulate already-queued work from the old transport after reconnection.
	s.expireTunnel("reconnecting", original.id, pending)
	s.disconnectTunnel("reconnecting", original)
	s.addLog("reconnecting", original, &protocol.TunnelLog{Path: "/stale"})
	if second.IsClosed() || activeSession(t, s, "reconnecting") != replacement {
		t.Fatal("stale cleanup removed replacement")
	}
	body := apiRequest(t, proxy, "GET", "/tunnels/reconnecting/logs", "owner", http.StatusOK)
	if !strings.Contains(string(body), "/before-reconnect") || strings.Contains(string(body), "/stale") {
		t.Fatalf("incorrect log lifecycle: %s", body)
	}
	// Replacing a still-live connection is allowed only for the same owner.
	third := dialOwner(t, proxy, "reconnecting", "owner", http.StatusSwitchingProtocols)
	waitFor(t, second.IsClosed)
	waitFor(t, func() bool { return activeSession(t, s, "reconnecting") != replacement })
	third.Close()
	waitFor(t, func() bool {
		s.mu.RLock()
		defer s.mu.RUnlock()
		pending = s.pendingRemoves["reconnecting"]
		return pending != nil
	})
	data, err := s.registry.Get(context.Background(), "reconnecting")
	if err != nil || data == nil {
		t.Fatalf("missing disconnected session: %v", err)
	}
	s.expireTunnel("reconnecting", data.SessionID, pending)
	dialOwner(t, proxy, "reconnecting", "different-owner", http.StatusConflict)
	dialOwner(t, proxy, "reconnecting", "owner", http.StatusSwitchingProtocols)
}

type failedHandshakeWriter struct {
	http.ResponseWriter
	entered chan struct{}
	release chan struct{}
}

func (w *failedHandshakeWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	close(w.entered)
	<-w.release
	return nil, nil, errors.New("handshake transport failed")
}

func TestFailedReplacementPreservesLiveGeneration(t *testing.T) {
	for _, transportFailure := range []bool{false, true} {
		name := "missing-websocket-key"
		if transportFailure {
			name = "transport-failure-with-concurrent-refresh"
		}
		t.Run(name, func(t *testing.T) {
			s, proxy, _ := privacyServer(t)
			client := dialOwner(t, proxy, "original", "owner", http.StatusSwitchingProtocols)
			original := activeSession(t, s, "original")
			s.addLog("original", original, &protocol.TunnelLog{Path: "/preserved"})
			req := httptest.NewRequest(http.MethodGet, "/tunnels/connect?tunnel_id=original&port=4000", nil)
			req.Header.Set("X-Tunnel-Token", "owner")
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Sec-WebSocket-Version", "13")
			recorder := httptest.NewRecorder()
			if transportFailure {
				req.Header.Set("Sec-WebSocket-Key", base64.StdEncoding.EncodeToString([]byte("0123456789abcdef")))
				writer := &failedHandshakeWriter{recorder, make(chan struct{}), make(chan struct{})}
				done := make(chan struct{})
				go func() { s.ServeHTTP(writer, req); close(done) }()
				select {
				case <-writer.entered:
				case <-done:
					t.Fatalf("request failed before hijacking: %d", recorder.Code)
				}
				// The original keepalive must remain valid throughout the upgrade,
				// not merely be restored after the attempted replacement fails.
				err := s.registry.Refresh(context.Background(), "original", original.id)
				close(writer.release)
				<-done
				if err != nil {
					t.Fatalf("concurrent refresh invalidated: %v", err)
				}
				if recorder.Code != http.StatusInternalServerError {
					t.Fatalf("handshake status: %d", recorder.Code)
				}
			} else {
				s.ServeHTTP(recorder, req)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("handshake status: %d", recorder.Code)
				}
			}
			data, err := s.registry.Get(context.Background(), "original")
			if err != nil || data == nil || data.SessionID != original.id || data.Port != 3000 {
				t.Fatalf("failed handshake changed the live record: %+v %v", data, err)
			}
			if activeSession(t, s, "original") != original || client.IsClosed() {
				t.Fatal("failed handshake replaced the live connection")
			}
			if err := s.registry.Refresh(context.Background(), "original", original.id); err != nil {
				t.Fatal(err)
			}
			body := apiRequest(t, proxy, "GET", "/tunnels/original/logs", "owner", http.StatusOK)
			if !strings.Contains(string(body), "/preserved") {
				t.Fatalf("failed handshake lost logs: %s", body)
			}
		})
	}
}

func TestRegistryClaimsAreAtomicAndReservationsSurviveExpiry(t *testing.T) {
	s, _, rdb := privacyServer(t)
	ctx := context.Background()
	start := make(chan struct{})
	results := make(chan string, 2)
	var wg sync.WaitGroup
	for _, owner := range []string{"owner-a", "owner-b"} {
		wg.Go(func() {
			<-start
			_, err := s.registry.Register(ctx, "contested", 3000, owner, owner+"-session")
			if err == nil {
				results <- owner
			} else if !errors.Is(err, ErrTunnelOwnership) {
				t.Errorf("claim: %v", err)
			}
		})
	}
	close(start)
	wg.Wait()
	close(results)
	var winners []string
	for owner := range results {
		winners = append(winners, owner)
	}
	if len(winners) != 1 {
		t.Fatalf("expected exactly one owner, got %v", winners)
	}
	owner := winners[0]
	if _, err := s.registry.Register(ctx, "contested", 3000, owner, "new-session"); err != nil {
		t.Fatal(err)
	}
	if err := s.registry.Remove(ctx, "contested", owner+"-session"); err != nil {
		t.Fatal(err)
	}
	if err := s.registry.Refresh(ctx, "contested", owner+"-session"); !errors.Is(err, ErrTunnelRevoked) {
		t.Fatalf("stale refresh: %v", err)
	}
	data, err := s.registry.Get(ctx, "contested")
	if err != nil || data == nil || data.SessionID != "new-session" {
		t.Fatalf("stale generation removed replacement: %+v %v", data, err)
	}
	rdb.FastForward(25 * time.Hour)
	if _, err := s.registry.Register(ctx, "contested", 3000, "stranger", "takeover"); !errors.Is(err, ErrTunnelOwnership) {
		t.Fatalf("expired ID takeover: %v", err)
	}
	if _, err := s.registry.Register(ctx, "contested", 3000, owner, "owner-reconnect"); err != nil {
		t.Fatal(err)
	}
	if err := s.registry.Revoke(ctx, "contested", "stranger"); !errors.Is(err, ErrTunnelOwnership) {
		t.Fatalf("unauthorized revoke: %v", err)
	}
	if err := s.registry.Revoke(ctx, "contested", owner); err != nil {
		t.Fatal(err)
	}
	if err := s.registry.Refresh(ctx, "contested", "owner-reconnect"); !errors.Is(err, ErrTunnelRevoked) {
		t.Fatalf("recreated revoked session: %v", err)
	}
	rdb.FastForward(25 * time.Hour)
	if _, err := s.registry.Register(ctx, "contested", 3000, owner, "revived"); !errors.Is(err, ErrTunnelRevoked) {
		t.Fatalf("revocation expired: %v", err)
	}
}

func TestReservationCannotInstallAfterClosure(t *testing.T) {
	s, _, _ := privacyServer(t)
	ctx := context.Background()
	if _, err := s.registry.Reserve(ctx, "reserved", "owner"); err != nil {
		t.Fatal(err)
	}
	if data, err := s.registry.Get(ctx, "reserved"); err != nil || data != nil {
		t.Fatalf("reservation installed a live record: %+v %v", data, err)
	}
	if _, err := s.registry.Reserve(ctx, "reserved", "other"); !errors.Is(err, ErrTunnelOwnership) {
		t.Fatalf("reservation takeover: %v", err)
	}
	if err := s.registry.Revoke(ctx, "reserved", "owner"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.registry.Register(ctx, "reserved", 3000, "owner", "late-install"); !errors.Is(err, ErrTunnelRevoked) {
		t.Fatalf("installation bypassed revocation after reservation: %v", err)
	}
	if data, err := s.registry.Get(ctx, "reserved"); err != nil || data != nil {
		t.Fatalf("revoked ID has a live record: %+v %v", data, err)
	}
}

func TestLegacyAnonymousSessionsFailClosed(t *testing.T) {
	s, proxy, rdb := privacyServer(t)
	rdb.Set("tunnel:legacy", `{"id":"legacy","port":3000}`)
	rdb.Set("tunnel:legacy-owned", `{"id":"legacy-owned","port":3000,"owner_hash":"`+hashToken("old-owner")+`"}`)
	rdb.SetTTL("tunnel:legacy", time.Hour)
	rdb.SetTTL("tunnel:legacy-owned", time.Hour)
	if err := s.registry.redis.ReserveLegacyOwners(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"", "new-owner"} {
		apiRequest(t, proxy, "GET", "/tunnels/legacy/logs", token, http.StatusUnauthorized)
		apiRequest(t, proxy, "DELETE", "/tunnels/legacy", token, http.StatusUnauthorized)
	}
	dialOwner(t, proxy, "legacy", "new-owner", http.StatusConflict)
	dialOwner(t, proxy, "new-anonymous", "", http.StatusUnauthorized)
	rdb.FastForward(2 * time.Hour)
	dialOwner(t, proxy, "legacy", "new-owner", http.StatusConflict)
	dialOwner(t, proxy, "legacy-owned", "new-owner", http.StatusConflict)
	dialOwner(t, proxy, "legacy-owned", "old-owner", http.StatusSwitchingProtocols)
}

func TestConcurrentReconnectionCannotUndoClosure(t *testing.T) {
	s, _, _ := privacyServer(t)
	ctx := context.Background()
	for i := range 20 {
		id := fmt.Sprintf("close-race-%d", i)
		if _, err := s.registry.Register(ctx, id, 3000, "owner", "original"); err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Go(func() {
			<-start
			_, err := s.registry.Register(ctx, id, 3000, "owner", "reconnect")
			if err != nil && !errors.Is(err, ErrTunnelRevoked) {
				t.Error(err)
			}
		})
		wg.Go(func() {
			<-start
			if err := s.registry.Revoke(ctx, id, "owner"); err != nil {
				t.Error(err)
			}
		})
		close(start)
		wg.Wait()
		if data, err := s.registry.Get(ctx, id); err != nil || data != nil {
			t.Fatalf("closure left a live record: %+v %v", data, err)
		}
		if _, err := s.registry.Register(ctx, id, 3000, "owner", "retry"); !errors.Is(err, ErrTunnelRevoked) {
			t.Fatalf("closure was undone: %v", err)
		}
	}
}
