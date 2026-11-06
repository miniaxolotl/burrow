package internal

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"burrow/protocol"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

type partitionConn struct {
	net.Conn
	paused atomic.Bool
}

func (c *partitionConn) Write(data []byte) (int, error) {
	if c.paused.Load() {
		return len(data), nil
	}
	return c.Conn.Write(data)
}
func (c *partitionConn) Read(data []byte) (int, error) {
	for {
		n, err := c.Conn.Read(data)
		if !c.paused.Load() || err != nil {
			return n, err
		}
	}
}

func TestPersistentPartitionExpiresWithinTransportBound(t *testing.T) {
	if os.Getenv("BURROW_TEST_LIVENESS") != "1" {
		t.Skip("explicit real-time 30-second transport partition acceptance")
	}
	s, proxy, _ := privacyServer(t)
	var transport *partitionConn
	dialer := *websocket.DefaultDialer
	dialer.NetDialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}
		transport = &partitionConn{Conn: conn}
		return transport, nil
	}
	endpoint := "ws" + strings.TrimPrefix(proxy.URL, "http") + "/tunnels/connect?tunnel_id=partitioned&port=3000"
	headers := http.Header{"X-Tunnel-Token": {"synthetic-owner"}, protocol.RegistrationHeader: {"1"}, protocol.AttachmentHeader: {"1"}, protocol.AttachmentIDHeader: {"attachment"}, protocol.AttemptHeader: {"attempt1"}}
	conn, _, err := dialer.Dial(endpoint, headers)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	cfg := yamux.DefaultConfig()
	cfg.EnableKeepAlive = false
	cfg.LogOutput = io.Discard
	session, err := yamux.Client(&protocol.WsConn{Conn: conn}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	control, err := session.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	control.SetReadDeadline(time.Now().Add(time.Second))
	ack, err := protocol.ReadRegistration(control, "partitioned")
	control.Close()
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	transport.paused.Store(true)
	other, response, err := websocket.DefaultDialer.Dial(endpoint, headers)
	if other != nil {
		other.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusLocked {
		t.Fatal("partition was treated as disconnected before transport expiry")
	}
	response.Body.Close()
	deadline := started.Add(32 * time.Second)
	for {
		s.mu.RLock()
		active := s.connections["partitioned"]
		s.mu.RUnlock()
		if active == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("half-open transport outlived 10s heartbeat + 20s timeout")
		}
		time.Sleep(25 * time.Millisecond)
	}
	headers.Set(protocol.ExpectedGenerationHeader, ack.Generation)
	headers.Set(protocol.AttemptHeader, "attempt2")
	conn2, _, err := websocket.DefaultDialer.Dial(endpoint, headers)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()
	resumed, err := yamux.Client(&protocol.WsConn{Conn: conn2}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer resumed.Close()
	control, err = resumed.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	control.SetReadDeadline(time.Now().Add(time.Second))
	next, err := protocol.ReadRegistration(control, "partitioned")
	control.Close()
	if err != nil || next.Generation == ack.Generation {
		t.Fatalf("recovery did not conditionally advance: %v", err)
	}
}
