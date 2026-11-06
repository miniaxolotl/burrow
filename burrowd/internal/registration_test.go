package internal

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"burrow/protocol"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

func TestRegistrationControlFollowsCommittedSession(t *testing.T) {
	s, proxy, _ := privacyServer(t)
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(proxy.URL, "http")+"/tunnels/connect?tunnel_id=registered&port=3000", http.Header{
		"X-Tunnel-Token": {"synthetic-owner"}, "X-Burrow-Registration": {"1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if response.Header.Get("X-Burrow-Registration") != "1" {
		t.Fatal("registration not negotiated")
	}
	session, err := yamux.Client(&protocol.WsConn{Conn: conn}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	control, err := session.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	defer control.Close()
	control.SetReadDeadline(time.Now().Add(time.Second))
	var ack struct {
		Version    int    `json:"version"`
		TunnelID   string `json:"tunnel_id"`
		Generation string `json:"generation"`
	}
	if err := json.NewDecoder(bufio.NewReader(control)).Decode(&ack); err != nil {
		t.Fatal(err)
	}
	s.mu.RLock()
	active := s.connections["registered"]
	s.mu.RUnlock()
	if active == nil || ack.Version != 1 || ack.TunnelID != "registered" || ack.Generation != active.id {
		t.Fatalf("uncommitted acknowledgement: %+v", ack)
	}
	stored, err := s.registry.Get(t.Context(), "registered")
	if err != nil || stored.SessionID != ack.Generation {
		t.Fatalf("ack precedes Redis: %+v %v", stored, err)
	}
}

type failRegistrationWriter struct {
	http.ResponseWriter
	fail func()
}

func (w failRegistrationWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := w.ResponseWriter.(http.Hijacker).Hijack()
	if err == nil {
		w.fail()
	}
	return conn, rw, err
}

func TestRegistrationFailureAfterUpgradeCannotAcknowledge(t *testing.T) {
	s, _, redis := privacyServer(t)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.ServeHTTP(failRegistrationWriter{w, func() { redis.SetError("registration unavailable") }}, r)
	}))
	defer proxy.Close()
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(proxy.URL, "http")+"/tunnels/connect?tunnel_id=uncommitted&port=3000", http.Header{"X-Tunnel-Token": {"synthetic-owner"}, protocol.RegistrationHeader: {"1"}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if response.StatusCode != 101 {
		t.Fatal("fixture did not pass upgrade")
	}
	session, err := yamux.Client(&protocol.WsConn{Conn: conn}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	control, err := session.AcceptStream()
	if err == nil {
		defer control.Close()
		control.SetReadDeadline(time.Now().Add(time.Second))
		if _, err := protocol.ReadRegistration(control, "uncommitted"); err == nil {
			t.Fatal("failed registration acknowledged")
		}
	}
	s.mu.RLock()
	active := s.connections["uncommitted"]
	s.mu.RUnlock()
	if active != nil {
		t.Fatal("failed registration installed")
	}
}

func TestUnknownRegistrationVersionDoesNotReserve(t *testing.T) {
	_, proxy, redis := privacyServer(t)
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(proxy.URL, "http")+"/tunnels/connect?tunnel_id=unsupported&port=3000", http.Header{
		"X-Tunnel-Token": {"synthetic-owner"}, "X-Burrow-Registration": {"999"},
	})
	if conn != nil {
		conn.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusBadRequest {
		t.Fatalf("unsupported version accepted: %v", err)
	}
	if redis.Exists("owner:unsupported") {
		t.Fatal("invalid capability reserved ownership")
	}
}
