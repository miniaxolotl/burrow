package routing

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
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func port(t *testing.T, server *httptest.Server) uint16 {
	t.Helper()
	_, p, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	n, _ := strconv.Atoi(p)
	return uint16(n)
}
func stream(t *testing.T, table *Table) net.Conn {
	t.Helper()
	client, server := net.Pipe()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); table.ServeStream(ctx, server) }()
	t.Cleanup(func() {
		client.Close()
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("stream worker did not drain")
		}
	})
	client.SetDeadline(time.Now().Add(3 * time.Second))
	return client
}
func TestHTTPDispatcherPreservesBodiesQueriesHostAndTrailers(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("Forwarded") != "" || r.Header.Get("X-Tunnel-Token") != "" {
			t.Error("untrusted forwarding identity or management header forwarded")
		}
		data, _ := io.ReadAll(r.Body)
		w.Header().Set("Trailer", "X-Complete")
		fmt.Fprintf(w, "%s %s %s %s", r.Method, r.Host, r.URL.RequestURI(), data)
		w.Header().Set("X-Complete", "done")
	}))
	defer upstream.Close()
	table := &Table{Version: 1, Routes: []Route{{Path: "/api", Port: port(t, upstream), StripPrefix: true}}}
	conn := stream(t, table)
	go fmt.Fprint(conn, "POST /api/a%20b?q=a%2Bb HTTP/1.1\r\nHost: public.test\r\nContent-Length: 3\r\nX-Forwarded-For: forged\r\nForwarded: for=forged\r\nX-Tunnel-Token: not-management\r\n\r\nabc")
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || string(data) != "POST public.test /a%20b?q=a%2Bb abc" || resp.Trailer.Get("X-Complete") != "done" {
		t.Fatalf("proxy result %q %v %+v", data, err, resp.Trailer)
	}
}

func TestHTTPDispatcherRejectsConnectAndDoesNotPipeline(t *testing.T) {
	var calls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; fmt.Fprint(w, "one") }))
	defer upstream.Close()
	table := &Table{Version: 1, Routes: []Route{{Path: "/", Port: port(t, upstream)}}}
	conn := stream(t, table)
	go fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: one.test\r\n\r\nGET /tunnels HTTP/1.1\r\nHost: other.test\r\n\r\n")
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if _, err := http.ReadResponse(reader, nil); err == nil {
		t.Fatal("pipelined request crossed stream isolation")
	}
	if calls != 1 {
		t.Fatalf("forwarded %d requests", calls)
	}
	conn = stream(t, table)
	go fmt.Fprint(conn, "CONNECT remote.test:443 HTTP/1.1\r\nHost: remote.test:443\r\n\r\n")
	resp, err = http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil || resp.StatusCode != 405 {
		t.Fatalf("CONNECT result %v %v", resp, err)
	}
	resp.Body.Close()
}

func TestHTTPDispatcherWebSocketAndSSECancellation(t *testing.T) {
	cancelled := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/events" {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: one\n\n")
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			close(cancelled)
			return
		}
		u := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		c, err := u.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		kind, data, err := c.ReadMessage()
		if err == nil {
			c.WriteMessage(kind, data)
		}
	}))
	defer upstream.Close()
	table := &Table{Version: 1, Routes: []Route{{Path: "/", Port: port(t, upstream)}}}
	conn := stream(t, table)
	dial := websocket.Dialer{NetDialContext: func(context.Context, string, string) (net.Conn, error) { return conn, nil }}
	ws, _, err := dial.Dial("ws://public.test/socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	ws.WriteMessage(websocket.TextMessage, []byte("echo"))
	_, data, err := ws.ReadMessage()
	if err != nil || string(data) != "echo" {
		t.Fatalf("upgrade: %q %v", data, err)
	}
	ws.Close()
	conn = stream(t, table)
	go fmt.Fprint(conn, "GET /events HTTP/1.1\r\nHost: public.test\r\n\r\n")
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	data = make([]byte, 11)
	if _, err := io.ReadFull(resp.Body, data); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	resp.Body.Close()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("SSE upstream not cancelled")
	}
}
