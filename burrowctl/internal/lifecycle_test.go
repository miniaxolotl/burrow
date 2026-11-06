package internal

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"burrow/protocol"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

func TestParentContextStopsTunnelTransport(t *testing.T) {
	opened := make(chan *yamux.Session, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		config := yamux.DefaultConfig()
		config.LogOutput = io.Discard
		session, err := yamux.Server(&protocol.WsConn{Conn: conn}, config)
		if err != nil {
			t.Error(err)
			return
		}
		defer session.Close()
		opened <- session
		<-session.CloseChan()
	}))
	defer server.Close()
	client := NewClient(strings.TrimPrefix(server.URL, "http://"), "synthetic-owner", "example.test", false)
	defer client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := client.CreateTunnel(ctx, 3000); err != nil {
		t.Fatal(err)
	}
	session := <-opened
	cancel()
	select {
	case <-session.CloseChan():
	case <-time.After(time.Second):
		t.Fatal("tunnel outlived its parent context")
	}
}

func TestReconnectEventsCountAttemptsNotBackoffSeconds(t *testing.T) {
	var connections atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if connections.Add(1) > 1 {
			http.Error(w, "temporary unavailability", http.StatusServiceUnavailable)
			return
		}
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			conn.Close()
		}
	}))
	defer server.Close()
	client := NewClient(strings.TrimPrefix(server.URL, "http://"), "synthetic-owner", "example.test", false)
	defer client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	attempts := make(chan int, 3)
	client.SetEventHandler(func(event LifecycleEvent) {
		if event.Type == "reconnecting" {
			attempts <- event.Attempt
		}
	})
	if _, err := client.CreateTunnel(ctx, 3000); err != nil {
		t.Fatal(err)
	}
	for expected := 1; expected <= 3; expected++ {
		select {
		case actual := <-attempts:
			if actual != expected {
				t.Fatalf("attempt %d reported as %d", expected, actual)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("missing reconnect attempt")
		}
	}
}

func TestClientCloseCancelsInFlightHandshake(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	defer server.CloseClientConnections()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := NewClient(strings.TrimPrefix(server.URL, "http://"), "synthetic-owner", "example.test", false)
	defer client.Close()
	created := make(chan error, 1)
	go func() {
		_, err := client.CreateTunnel(ctx, 3000)
		created <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("handshake did not start")
	}
	closed := make(chan struct{})
	go func() { client.Close(); close(closed) }()
	select {
	case err := <-created:
		if err == nil {
			t.Fatal("cancelled handshake succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("client close did not cancel its pending handshake")
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("client close did not drain")
	}
}
