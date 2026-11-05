package internal

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestLiveTunnel checks the actual WSS/yamux connection and public wildcard TLS
// route, including paths that would otherwise collide with the server's API.
func TestLiveTunnel(t *testing.T) {
	server := os.Getenv("BURROW_SMOKE_SERVER")
	if server == "" {
		t.Skip("set BURROW_SMOKE_SERVER to run the deployment smoke test")
	}
	marker := fmt.Sprintf("burrow-smoke-%d", time.Now().UnixNano())
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Burrow-Smoke", marker)
		body, _ := io.ReadAll(r.Body)
		fmt.Fprintf(w, "%s %s %s %s", marker, r.Method, r.URL.RequestURI(), body)
	}))
	defer local.Close()
	_, portString, _ := net.SplitHostPort(local.Listener.Addr().String())
	port, _ := strconv.Atoi(portString)
	client := NewClient(server, rand.Text(), server, true)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	url, err := client.CreateTunnel(ctx, uint16(port))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "https://") || !strings.HasSuffix(url, "."+server) {
		t.Fatalf("unexpected tunnel URL: %s", url)
	}
	id := strings.TrimSuffix(strings.TrimPrefix(url, "https://"), "."+server)
	defer func() {
		client.Close()
		if err := client.DeleteTunnel(id); err != nil {
			t.Errorf("revoke smoke-test tunnel: %v", err)
		}
	}()
	httpClient := &http.Client{Timeout: 20 * time.Second}
	for _, path := range []string{"/", "/health", "/tunnels", "/nested/path?probe=1"} {
		resp, err := liveRequest(ctx, httpClient, http.MethodPost, url+path, "forwarded-body")
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK || resp.Header.Get("X-Burrow-Smoke") != marker ||
			string(body) != marker+" POST "+path+" forwarded-body" {
			t.Fatalf("forwarding failed for %s: status=%d body=%s err=%v", path, resp.StatusCode, body, err)
		}
		// Origin pools may reuse a connection from a tunnel for the apex API.
		// A local app must never handle that subsequent management request.
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+server+"/health", nil)
		control, err := httpClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		controlBody, err := io.ReadAll(control.Body)
		control.Body.Close()
		if err != nil || control.StatusCode != http.StatusOK || string(controlBody) != `{"status":"ok"}` || control.Header.Get("X-Burrow-Smoke") != "" {
			t.Fatalf("apex routing after %s: status=%d expected health JSON=%t err=%v", path, control.StatusCode, string(controlBody) == `{"status":"ok"}`, err)
		}
	}
	// Drop the transport to exercise the same reconnect path used during a
	// deployment. The existing public URL must keep working after reconnection.
	client.mu.RLock()
	connection := client.tunnels[uint16(port)].wsConn
	client.mu.RUnlock()
	connection.Close()
	deadline := time.Now().Add(30 * time.Second)
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url+"/reconnected", nil)
		resp, err := httpClient.Do(req)
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr == nil && resp.StatusCode == http.StatusOK &&
				string(body) == marker+" GET /reconnected " {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("tunnel did not reconnect at %s: %v", url, err)
		}
		time.Sleep(time.Second)
	}
	t.Logf("Verified HTTPS forwarding and reconnection through %s", url)
}
