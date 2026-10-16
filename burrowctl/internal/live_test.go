package internal

import (
	"context"
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
	client := NewClient(server, marker, server, true)
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
	httpClient := &http.Client{Timeout: 20 * time.Second}
	for _, path := range []string{"/", "/health", "/tunnels", "/nested/path?probe=1"} {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url+path, strings.NewReader("forwarded-body"))
		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK || resp.Header.Get("X-Burrow-Smoke") != marker ||
			string(body) != marker+" POST "+path+" forwarded-body" {
			t.Fatalf("forwarding failed for %s: status=%d body=%s err=%v", path, resp.StatusCode, body, err)
		}
	}
	t.Logf("Verified HTTPS tunnel forwarding through %s", url)
}
