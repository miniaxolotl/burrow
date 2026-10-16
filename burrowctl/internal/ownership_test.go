package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"burrow/protocol"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

func TestRevokedTunnelStopsReconnectingAndHealthOmitsCredential(t *testing.T) {
	healthChecked := make(chan struct{})
	var once sync.Once
	var attempts atomic.Int32
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			if r.Header.Get("X-Tunnel-Token") != "" {
				t.Error("public health probe included owner credential")
			}
			w.Write([]byte(`{"status":"ok"}`))
			once.Do(func() { close(healthChecked) })
			return
		}
		if attempts.Add(1) > 1 {
			http.Error(w, "Tunnel has been closed", http.StatusGone)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		session, err := yamux.Server(&protocol.WsConn{Conn: conn}, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer session.Close()
		select {
		case <-healthChecked:
		case <-time.After(5 * time.Second):
			t.Error("no health probe")
		}
	}))
	defer server.Close()
	client := NewClient(strings.TrimPrefix(server.URL, "http://"), "owner-credential", "example.test", false)
	defer client.Close()
	if _, err := client.CreateTunnel(context.Background(), 3000); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		client.mu.RLock()
		remaining := len(client.tunnels)
		client.mu.RUnlock()
		if remaining == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("revoked tunnel kept reconnecting")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if attempts.Load() != 2 {
		t.Fatalf("unexpected connection attempts: %d", attempts.Load())
	}
}
