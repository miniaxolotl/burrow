package internal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"burrow/protocol"
	"burrow/protocol/telemetry"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

type observedTelemetry struct {
	mu         sync.Mutex
	events     []telemetry.Event
	operations []telemetry.Outcome
	forwarded  chan struct{}
}

func (o *observedTelemetry) Track(event telemetry.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
}
func (o *observedTelemetry) Operation(operation telemetry.Operation, _ time.Time, outcome telemetry.Outcome) {
	if operation != telemetry.Forward {
		return
	}
	o.mu.Lock()
	o.operations = append(o.operations, outcome)
	o.mu.Unlock()
	o.forwarded <- struct{}{}
}

func TestTelemetryOwnsFirstCompletedForwardingWithoutReadingContent(t *testing.T) {
	observer := &observedTelemetry{forwarded: make(chan struct{}, 4)}
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("sentry-trace") != "" || req.Header.Get("baggage") != "" {
			t.Error("telemetry propagated into a local service")
		}
		body, _ := io.ReadAll(req.Body)
		fmt.Fprintf(w, "%s %s", req.URL.RequestURI(), body)
	}))
	defer local.Close()
	_, portString, _ := net.SplitHostPort(local.Listener.Addr().String())
	port, _ := strconv.Atoi(portString)
	sessions := make(chan *yamux.Session, 3)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/health" {
			w.WriteHeader(200)
			return
		}
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			t.Error(err)
			return
		}
		session, err := yamux.Server(&protocol.WsConn{Conn: conn}, nil)
		if err != nil {
			t.Error(err)
			conn.Close()
			return
		}
		defer session.Close()
		sessions <- session
		<-session.CloseChan()
	}))
	defer server.Close()
	client := NewClient(strings.TrimPrefix(server.URL, "http://"), "PRIVATE_CREDENTIAL", "private.example", false, observer)
	defer client.Close()
	if _, err := client.CreateTunnel(context.Background(), uint16(port)); err != nil {
		t.Fatal(err)
	}
	var session *yamux.Session
	select {
	case session = <-sessions:
	case <-time.After(5 * time.Second):
		t.Fatal("missing session")
	}
	forward := func(expected bool) {
		t.Helper()
		stream, err := session.OpenStream()
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		_ = stream.SetDeadline(time.Now().Add(5 * time.Second))
		req, _ := http.NewRequest(http.MethodPost, "http://private.example/PRIVATE?token=PRIVATE", strings.NewReader("PRIVATE_BODY"))
		req.Close = true
		_ = req.Write(stream)
		resp, err := http.ReadResponse(bufio.NewReader(stream), req)
		if expected {
			if err != nil {
				t.Fatal(err)
			}
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil || string(body) != "/PRIVATE?token=PRIVATE PRIVATE_BODY" {
				t.Fatalf("forwarding changed: %s %v", body, readErr)
			}
		} else if err == nil {
			resp.Body.Close()
			t.Fatal("closed local service unexpectedly responded")
		}
		select {
		case <-observer.forwarded:
		case <-time.After(5 * time.Second):
			t.Fatal("missing forwarding outcome")
		}
	}
	forward(true)
	forward(true)
	local.Close()
	forward(false)
	observer.mu.Lock()
	defer observer.mu.Unlock()
	if fmt.Sprint(observer.events) != "[tunnel_opened tunnel_request_completed]" {
		t.Fatalf("unexpected milestone ownership: %v", observer.events)
	}
	if fmt.Sprint(observer.operations) != "[success success unavailable]" {
		t.Fatalf("incorrect outcomes: %v", observer.operations)
	}
}
