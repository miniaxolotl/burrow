package internal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

func proxyFixture(t *testing.T, server *Server, id string, handler http.Handler) {
	t.Helper()
	a, b := net.Pipe()
	cfg := yamux.DefaultConfig()
	cfg.LogOutput = io.Discard
	upstream, err := yamux.Server(a, cfg)
	if err != nil {
		t.Fatal(err)
	}
	downstream, err := yamux.Client(b, cfg)
	if err != nil {
		upstream.Close()
		t.Fatal(err)
	}
	local := &http.Server{Handler: handler}
	go local.Serve(downstream)
	t.Cleanup(func() { local.Close(); upstream.Close(); downstream.Close() })
	server.connections[id] = &tunnelSession{conn: upstream}
}

func TestProxyRedispatchesKeepAliveHosts(t *testing.T) {
	server := NewServer(nil, "example.test", false)
	for _, id := range []string{"first", "second"} {
		proxyFixture(t, server, id, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Tunnel-Token") != "" {
				t.Error("management credential reached a local service")
			}
			io.Copy(io.Discard, r.Body)
			fmt.Fprint(w, id)
		}))
	}
	proxy := httptest.NewServer(server)
	defer proxy.Close()
	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	var firstConn net.Conn
	for _, tc := range []struct {
		host, path, want string
		status           int
	}{
		{"first.example.test", "/nested?probe=1", "first", 200},
		{"example.test", "/health", `{"status":"ok"}`, 200},
		{"second.example.test", "/", "second", 200},
		{"example.test", "/tunnels", `"count":0`, 200},
		{"missing.example.test", "/", "Tunnel not connected", 503},
	} {
		req, _ := http.NewRequest(http.MethodPost, proxy.URL+tc.path, strings.NewReader("body"))
		if tc.host == "example.test" {
			req.Method = http.MethodGet
		}
		req.Host = tc.host
		if tc.host == "example.test" && tc.path == "/tunnels" {
			req.Header.Set("X-Tunnel-Token", "synthetic-management-only")
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{GotConn: func(info httptrace.GotConnInfo) {
			if firstConn == nil {
				firstConn = info.Conn
			} else if info.Conn != firstConn {
				t.Error("test did not reuse the same ingress-origin connection")
			}
		}}))
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != tc.status || !strings.Contains(string(body), tc.want) {
			t.Fatalf("host=%s path=%s status=%d body=%q; want status=%d body=%q", tc.host, tc.path, resp.StatusCode, body, tc.status, tc.want)
		}
	}
}

func TestProxyPipelinedHosts(t *testing.T) {
	server := NewServer(nil, "example.test", false)
	proxyFixture(t, server, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.Copy(io.Discard, r.Body); fmt.Fprint(w, "local-app") }))
	proxy := httptest.NewServer(server)
	defer proxy.Close()
	conn, err := net.DialTimeout("tcp", strings.TrimPrefix(proxy.URL, "http://"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	fmt.Fprint(conn, "POST /first HTTP/1.1\r\nHost: probe.example.test\r\nContent-Length: 4\r\n\r\nbodyGET /health HTTP/1.1\r\nHost: example.test\r\nConnection: close\r\n\r\n")
	reader := bufio.NewReader(conn)
	for _, want := range []string{"local-app", `{"status":"ok"}`} {
		resp, err := http.ReadResponse(reader, nil)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 || string(body) != want {
			t.Fatalf("pipelined body=%q status=%d want=%q err=%v", body, resp.StatusCode, want, err)
		}
	}
}

func TestProxyStreamingTrailers(t *testing.T) {
	server := NewServer(nil, "example.test", false)
	release := make(chan struct{})
	proxyFixture(t, server, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Trailer", "X-Fixture-Complete")
		fmt.Fprint(w, "first\n")
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		fmt.Fprint(w, "last\n")
		w.Header().Set("X-Fixture-Complete", "yes")
	}))
	proxy := httptest.NewServer(server)
	defer proxy.Close()
	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, proxy.URL+"/stream", nil)
	req.Host = "probe.example.test"
	resp, err := client.Do(req)
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	close(release)
	if err != nil || line != "first\n" {
		t.Fatalf("first streaming chunk=%q err=%v", line, err)
	}
	body, err := io.ReadAll(reader)
	if err != nil || string(body) != "last\n" || resp.Trailer.Get("X-Fixture-Complete") != "yes" {
		t.Fatalf("stream/trailer: body=%q trailer=%q err=%v", body, resp.Trailer.Get("X-Fixture-Complete"), err)
	}
}

func TestProxyCancellation(t *testing.T) {
	server := NewServer(nil, "example.test", false)
	done := make(chan struct{})
	proxyFixture(t, server, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(done)
	}))
	proxy := httptest.NewServer(server)
	defer proxy.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, proxy.URL+"/cancel", nil)
	req.Host = "probe.example.test"
	client := &http.Client{Timeout: 3 * time.Second}
	defer client.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	resp.Body.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("cancelled request retained its upstream stream")
	}
}

func TestProxyLogsEachKeepAliveRequest(t *testing.T) {
	server := NewServer(nil, "example.test", false)
	proxyFixture(t, server, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") }))
	proxy := httptest.NewServer(server)
	defer proxy.Close()
	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/request/%d", proxy.URL, i), nil)
		req.Host = "probe.example.test"
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	deadline := time.Now().Add(time.Second)
	for {
		server.mu.RLock()
		count := len(server.logs["probe"])
		size := server.totalSize["probe"]
		server.mu.RUnlock()
		if count == 3 && size > 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("keep-alive requests not individually accounted: logs=%d response_bytes=%d", count, size)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestProxyHTTP2(t *testing.T) {
	server := NewServer(nil, "example.test", false)
	proxyFixture(t, server, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Probe", "ok")
		io.Copy(w, r.Body)
	}))
	proxy := httptest.NewUnstartedServer(server)
	proxy.EnableHTTP2 = true
	proxy.StartTLS()
	defer proxy.Close()
	req, _ := http.NewRequest(http.MethodPost, proxy.URL+"/echo?probe=1", strings.NewReader("http2-body"))
	req.Host = "probe.example.test"
	resp, err := proxy.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.ProtoMajor != 2 || resp.StatusCode != 200 || string(body) != "http2-body" || resp.Header.Get("X-Probe") != "ok" {
		t.Fatalf("HTTP/2 forwarding: protocol=%s status=%d body=%q err=%v", resp.Proto, resp.StatusCode, body, err)
	}
}

func TestProxyWebSocket(t *testing.T) {
	server := NewServer(nil, "example.test", false)
	proxyFixture(t, server, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			kind, body, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if conn.WriteMessage(kind, body) != nil {
				return
			}
		}
	}))
	proxy := httptest.NewServer(server)
	defer proxy.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, "ws"+strings.TrimPrefix(proxy.URL, "http")+"/socket", http.Header{"Host": []string{"probe.example.test"}})
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for i := 0; i < 3; i++ {
		want := fmt.Sprintf("message-%d", i)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(want)); err != nil {
			t.Fatal(err)
		}
		kind, body, err := conn.ReadMessage()
		if err != nil || kind != websocket.TextMessage || string(body) != want {
			t.Fatalf("WS echo: kind=%d body=%q err=%v", kind, body, err)
		}
	}
}
