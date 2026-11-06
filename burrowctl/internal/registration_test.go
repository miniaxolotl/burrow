package internal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"burrow/protocol"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

func TestClientWaitsForMatchingRegistration(t *testing.T) {
	upgraded, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := websocket.Upgrader{}
		conn, err := u.Upgrade(w, r, http.Header{protocol.RegistrationHeader: {"1"}})
		if err != nil {
			return
		}
		defer conn.Close()
		cfg := yamux.DefaultConfig()
		cfg.LogOutput = io.Discard
		session, err := yamux.Server(&protocol.WsConn{Conn: conn}, cfg)
		if err != nil {
			return
		}
		defer session.Close()
		close(upgraded)
		<-release
		control, err := session.Open()
		if err != nil {
			return
		}
		json.NewEncoder(control).Encode(protocol.Registration{Version: 1, TunnelID: r.URL.Query().Get("tunnel_id"), Generation: "committed"})
		control.Close()
		<-session.CloseChan()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	client := NewClient(strings.TrimPrefix(server.URL, "http://"), "synthetic-owner", "example.test", false)
	defer client.Close()
	events := make(chan LifecycleEvent, 8)
	client.SetEventHandler(func(e LifecycleEvent) { events <- e })
	created := make(chan error, 1)
	go func() { _, err := client.CreateTunnel(ctx, 3000); created <- err }()
	<-upgraded
	select {
	case err := <-created:
		t.Fatalf("startup completed without acknowledgement: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-created:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("ack did not release startup")
	}
	found := false
	for len(events) > 0 {
		e := <-events
		if e.Type == "registered" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing registered observation")
	}
}

func TestTunnelIdentityDoesNotDependOnUpstreamPort(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := websocket.Upgrader{}
		conn, err := u.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		cfg := yamux.DefaultConfig()
		cfg.LogOutput = io.Discard
		session, err := yamux.Server(&protocol.WsConn{Conn: conn}, cfg)
		if err != nil {
			return
		}
		defer session.Close()
		<-session.CloseChan()
	}))
	defer server.Close()
	client := NewClient(strings.TrimPrefix(server.URL, "http://"), "synthetic-owner", "example.test", false)
	defer client.Close()
	for range 2 {
		if _, err := client.CreateTunnel(t.Context(), 3000); err != nil {
			t.Fatal(err)
		}
	}
	if tunnels := client.ListTunnels(); len(tunnels) != 2 || tunnels[0].TunnelID == tunnels[1].TunnelID {
		t.Fatalf("upstream port used as identity: %+v", tunnels)
	}
}
