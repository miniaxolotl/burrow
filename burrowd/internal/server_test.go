package internal

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/yamux"
)

// Requests with bodies can leave net/http's look-ahead byte in the hijacked
// reader. Reusing the connection must preserve every byte of the next method.
func TestTunnelHTTPKeepAlive(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	serverSession, err := yamux.Server(serverConn, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := yamux.Client(clientConn, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	local := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, "%s %s %s %s", r.Method, r.Host, r.URL.RequestURI(), body)
	})}
	defer local.Close()
	go local.Serve(clientSession)

	server := NewServer(nil, "example.test", "", false)
	server.connections["probe"] = serverSession
	proxy := httptest.NewServer(server)
	defer proxy.Close()
	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	reused := 0
	for i := range 12 {
		path := fmt.Sprintf("/health?request=%d", i)
		body := fmt.Sprintf("request-body-%d", i)
		req, err := http.NewRequest(http.MethodPost, proxy.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "probe.example.test"
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
			GotConn: func(info httptrace.GotConnInfo) {
				if info.Reused {
					reused++
				}
			},
		}))
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		result, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		want := "POST probe.example.test " + path + " " + body
		if resp.StatusCode != http.StatusOK || string(result) != want {
			t.Fatalf("request %d: status=%d body=%q, want %q", i, resp.StatusCode, result, want)
		}
	}
	if reused != 11 {
		t.Fatalf("expected 11 keep-alive requests, got %d", reused)
	}
}
