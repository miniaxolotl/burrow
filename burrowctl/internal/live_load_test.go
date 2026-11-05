package internal

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"burrow/protocol"
	"github.com/gorilla/websocket"
)

// TestLiveTunnelLoad is deliberately separate from the deployment smoke switch.
// Its fixed ceiling is three tunnels, 100 RPS, 32 in flight and 2100 load requests.
// Run only with operator authorization; this is not a saturation benchmark.
func TestLiveTunnelLoad(t *testing.T) {
	server := os.Getenv("BURROW_LOAD_SERVER")
	if server == "" {
		t.Skip("set BURROW_LOAD_SERVER for authorized bounded live load testing")
	}
	if strings.ContainsAny(server, "/:?# ") {
		t.Fatal("BURROW_LOAD_SERVER must be a TLS hostname, not a URL")
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", os.Getenv("HOME"))
	t.Setenv("BURROW_TELEMETRY_ENABLED", "false")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	owner, marker := rand.Text(), rand.Text()
	client := NewClient(server, owner, server, true)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = 32
	transport.MaxIdleConnsPerHost = 32
	transport.ResponseHeaderTimeout = 8 * time.Second
	public := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	defer transport.CloseIdleConnections()
	var ids, urls []string
	t.Cleanup(func() {
		client.Close()
		for _, id := range ids {
			if err := client.DeleteTunnel(id); err != nil {
				t.Errorf("revoke synthetic tunnel: %v", err)
			}
		}
		list, err := client.RemoteTunnels()
		if err != nil || list.Count != 0 {
			t.Errorf("synthetic owner cleanup not verified: list=%+v err=%v", list, err)
		} else {
			t.Log("cleanup: zero active test-owner tunnels; revoked ownership tombstones remain by design")
		}
	})
	probe := func(ctx context.Context, url string, size int) (time.Duration, error) {
		want := bytes.Repeat([]byte("x"), size)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(want))
		if err != nil {
			return 0, err
		}
		started := time.Now()
		resp, err := public.Do(req)
		if err != nil {
			return time.Since(started), err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, int64(size)+1))
		if err == nil && (resp.StatusCode != 200 || resp.Header.Get("X-Burrow-Fixture") != marker || resp.Header.Get("X-Burrow-Fixture-Path") != req.URL.RequestURI() || !bytes.Equal(body, want)) {
			err = fmt.Errorf("unexpected fixture response: status=%d marker=%t path=%t body=%t", resp.StatusCode, resp.Header.Get("X-Burrow-Fixture") == marker, resp.Header.Get("X-Burrow-Fixture-Path") == req.URL.RequestURI(), bytes.Equal(body, want))
		}
		return time.Since(started), err
	}
	for i := 0; i < 3; i++ {
		local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(io.LimitReader(r.Body, 4*1024*1024+1))
			if err != nil || len(body) > 4*1024*1024 {
				http.Error(w, "fixture input bound", 413)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Burrow-Fixture", marker)
			w.Header().Set("X-Burrow-Fixture-Path", r.URL.RequestURI())
			w.Write(body)
		}))
		defer local.Close()
		_, text, _ := net.SplitHostPort(local.Listener.Addr().String())
		port, _ := strconv.Atoi(text)
		url, err := client.CreateTunnel(ctx, uint16(port))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, strings.TrimSuffix(strings.TrimPrefix(url, "https://"), "."+server))
		urls = append(urls, url)
		if _, err := probe(ctx, url+"/ready", 32); err != nil {
			t.Fatal(err)
		}
	}
	list, err := client.RemoteTunnels()
	if err != nil || list.Count != 3 {
		t.Fatalf("three isolated tunnels: list=%+v err=%v", list, err)
	}
	t.Log("three simultaneous tunnels verified with a fresh owner and unique fixture marker; global installation/config untouched")
	for _, stage := range []struct{ rps, concurrency int }{{5, 2}, {15, 5}, {30, 10}, {60, 20}, {100, 32}} {
		stageCtx, stop := context.WithCancel(ctx)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var samples []time.Duration
		var failures []error
		sem := make(chan struct{}, stage.concurrency)
		ticker := time.NewTicker(time.Second / time.Duration(stage.rps))
		started := time.Now()
		peak := 0
	stageLoop:
		for i := 0; i < stage.rps*10; i++ {
			select {
			case <-stageCtx.Done():
				break stageLoop
			case <-ticker.C:
			}
			select {
			case <-stageCtx.Done():
				break stageLoop
			case sem <- struct{}{}:
			}
			if len(sem) > peak {
				peak = len(sem)
			}
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				defer func() { <-sem }()
				d, err := probe(stageCtx, urls[i%len(urls)]+fmt.Sprintf("/load/%d/%d?probe=1", stage.rps, i), 1024)
				if err == nil && d > 2*time.Second {
					err = fmt.Errorf("request safety deadline exceeded: %s", d)
				}
				mu.Lock()
				samples = append(samples, d)
				if err != nil {
					failures = append(failures, err)
					stop()
				}
				mu.Unlock()
			}(i)
		}
		ticker.Stop()
		wg.Wait()
		stop()
		elapsed := time.Since(started)
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		if len(samples) == 0 {
			t.Fatal("no load samples")
		}
		percentile := func(p float64) time.Duration {
			return samples[int(math.Ceil(p*float64(len(samples))))-1].Round(time.Millisecond)
		}
		t.Logf("target=%d rps completed=%d errors=%d elapsed=%s achieved=%.2f rps peak_inflight=%d p50=%s p95=%s p99=%s max=%s", stage.rps, len(samples), len(failures), elapsed.Round(time.Millisecond), float64(len(samples))/elapsed.Seconds(), peak, percentile(.5), percentile(.95), percentile(.99), samples[len(samples)-1].Round(time.Millisecond))
		if len(failures) > 0 {
			t.Fatalf("load stopped on first error: %v", failures[0])
		}
		if len(samples) != stage.rps*10 {
			t.Fatal("bounded load schedule interrupted")
		}
		// Check management routing too, without exposing any fixture credential to logs.
		list, err = client.RemoteTunnels()
		if err != nil || list.Count != 3 {
			t.Fatalf("management routing under load: list=%+v err=%v", list, err)
		}
	}
	for _, size := range []int{64 * 1024, 1024 * 1024, 4 * 1024 * 1024} {
		d, err := probe(ctx, urls[0]+fmt.Sprintf("/payload/%d", size), size)
		if err != nil {
			t.Fatalf("payload %d bytes: %v", size, err)
		}
		t.Logf("upload+echo integrity: %d bytes in %s", size, d.Round(time.Millisecond))
	}
	var logs []*protocol.TunnelLog
	for attempt := 0; attempt < 20; attempt++ {
		logs, err = client.GetLogs(ids[0])
		if err != nil {
			t.Fatal(err)
		}
		if len(logs) == 200 && logs[len(logs)-1].Path == "/payload/4194304" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(logs) != 200 || logs[len(logs)-1].Path != "/payload/4194304" {
		t.Fatalf("history cap/latest completion: got %d entries", len(logs))
	}
	var retained int64
	for _, entry := range logs {
		retained += entry.Size
	}
	list, err = client.RemoteTunnels()
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, entry := range list.Tunnels {
		if entry.TunnelID == ids[0] {
			total = entry.TotalSize
		}
	}
	if total <= retained {
		t.Fatalf("evicted history lost usage: total=%d retained=%d", total, retained)
	}
	t.Logf("history retained=%d entries; response-stream bytes total=%d retained=%d (not upload accounting or a bandwidth quota)", len(logs), total, retained)
	// Release normal tunnels before a bounded oversized-message negative probe.
	client.Close()
	for _, id := range ids {
		if err := client.DeleteTunnel(id); err != nil {
			t.Fatal(err)
		}
	}
	ids = nil
	id := protocol.RandomTunnelID()
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, "wss://"+server+"/tunnels/connect?tunnel_id="+id+"&port=1", http.Header{"X-Tunnel-Token": []string{owner}})
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		t.Fatal(err)
	}
	ids = append(ids, id)
	defer conn.Close()
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, protocol.MaxMessageSize+1)); err != nil {
		t.Fatal(err)
	}
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err = conn.ReadMessage()
	if !websocket.IsCloseError(err, websocket.CloseMessageTooBig) {
		t.Fatalf("oversized transport message expected close 1009, got %v", err)
	}
	t.Log("1 MiB + 1 byte transport message rejected with close 1009; HTTP payloads up to 4 MiB passed as framed streams")
}
