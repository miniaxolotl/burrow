package internal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestCombinedWorkflowsRealServices(t *testing.T) {
	address := os.Getenv("BURROW_TEST_REDIS")
	if address == "" || os.Getenv("BURROW_TEST_SECURE_BACKEND") != "1" {
		t.Skip("requires disposable Redis and an explicitly isolated unlocked credential backend")
	}
	redis, err := NewRedisClient(address)
	if err != nil {
		t.Fatal(err)
	}
	defer redis.Close()
	registry := NewTunnelRegistry(redis, "example.test", false)
	var current atomic.Pointer[Server]
	first := NewServer(registry, "example.test", false)
	current.Store(first)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { current.Load().ServeHTTP(w, r) }))
	defer proxy.Close()
	defer func() { current.Load().Shutdown(context.Background()) }()
	var apiCalls, rootCalls atomic.Int32
	body := []byte("synthetic-combined-body")
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls.Add(1)
		data, _ := io.ReadAll(r.Body)
		if r.URL.Path != "/hook" || !bytes.Equal(data, body) {
			t.Error("API request mapping/body changed")
		}
		w.Write(data)
	}))
	defer api.Close()
	root := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { rootCalls.Add(1); fmt.Fprint(w, "root marker") }))
	defer root.Close()
	port := func(server *httptest.Server) int {
		_, p, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
		n, _ := strconv.Atoi(p)
		return n
	}
	home := t.TempDir()
	routeFile := filepath.Join(home, "routes.json")
	if err := os.WriteFile(routeFile, []byte(fmt.Sprintf(`{"version":1,"routes":[{"path":"/api","port":%d,"strip_prefix":true},{"path":"/","port":%d}]}`, port(api), port(root))), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 75*time.Second)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "burrowctl")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../burrowctl")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	command := func(args ...string) *exec.Cmd {
		args = append(args, "--server", strings.TrimPrefix(proxy.URL, "http://"))
		cmd := exec.CommandContext(ctx, binary, args...)
		for _, entry := range os.Environ() {
			key, _, _ := strings.Cut(entry, "=")
			if key == "HOME" || strings.HasPrefix(key, "BURROW_") || strings.HasPrefix(key, "XDG_") {
				continue
			}
			cmd.Env = append(cmd.Env, entry)
		}
		cmd.Env = append(cmd.Env, "HOME="+home, "DO_NOT_TRACK=1")
		return cmd
	}
	type envelope struct {
		Type string `json:"type"`
		Data struct {
			CaptureID string `json:"capture_id"`
			Tunnels   []struct {
				TunnelID   string `json:"tunnel_id"`
				URL        string `json:"url"`
				Generation string `json:"generation"`
			} `json:"tunnels"`
		} `json:"data"`
	}
	start := func(recording bool) (*exec.Cmd, *bufio.Scanner, *bytes.Buffer, envelope) {
		args := []string{"tunnel", "--name", "combined", "--config", routeFile, "--output", "jsonl"}
		if recording {
			args = append(args, "--capture")
		}
		cmd := command(args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		stderr := new(bytes.Buffer)
		cmd.Stderr = stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { cmd.Process.Kill() })
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			var e envelope
			if json.Unmarshal(scanner.Bytes(), &e) != nil {
				t.Fatal("invalid JSONL")
			}
			if e.Type == "error" {
				t.Fatalf("startup: %s", scanner.Text())
			}
			if e.Type == "ready" {
				if len(e.Data.Tunnels) != 1 {
					t.Fatal("invalid barrier")
				}
				return cmd, scanner, stderr, e
			}
		}
		t.Fatalf("no ready: %v %s", scanner.Err(), stderr)
		return nil, nil, nil, envelope{}
	}
	stop := func(cmd *exec.Cmd, scanner *bufio.Scanner, stderr *bytes.Buffer) {
		cmd.Process.Signal(syscall.SIGTERM)
		for scanner.Scan() {
		}
		if err := cmd.Wait(); err != nil || stderr.Len() != 0 {
			t.Fatalf("stop: %v %s", err, stderr)
		}
	}
	public := func(host, method, path string, data []byte) []byte {
		req, _ := http.NewRequestWithContext(ctx, method, proxy.URL+path, bytes.NewReader(data))
		req.Host = host
		req.Header.Set("Authorization", "Bearer synthetic-private-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		out, err := io.ReadAll(resp.Body)
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("public loopback request: %d %v", resp.StatusCode, err)
		}
		return out
	}
	cmd, scanner, stderr, ready := start(true)
	one := ready.Data.Tunnels[0]
	origin, _ := url.Parse(one.URL)
	if out := public(origin.Host, "POST", "/api/hook?token=synthetic-query-secret", body); !bytes.Equal(out, body) {
		t.Fatal("routing changed response")
	}
	requestID := ""
	for scanner.Scan() {
		var e envelope
		if json.Unmarshal(scanner.Bytes(), &e) != nil {
			t.Fatal("invalid capture output")
		}
		if e.Type == "capture_saved" {
			requestID = e.Data.CaptureID
			break
		}
		if e.Type == "capture_unavailable" || e.Type == "error" {
			t.Fatalf("capture: %s", scanner.Text())
		}
	}
	if requestID == "" {
		t.Fatal("capture not committed")
	}
	stop(cmd, scanner, stderr)
	// Exercise the real default, not the shortened unit-test grace.
	if first.gracePeriod != tunnelGracePeriod {
		t.Fatal("fixture changed the default grace")
	}
	expiresBy := time.Now().Add(tunnelGracePeriod + 3*time.Second)
	for {
		td, err := registry.Get(ctx, one.TunnelID)
		if err != nil {
			t.Fatal(err)
		}
		if td == nil {
			break
		}
		if time.Now().After(expiresBy) {
			t.Fatal("metadata survived the default disconnect grace")
		}
		time.Sleep(25 * time.Millisecond)
	}
	cmd, scanner, stderr, ready = start(false)
	two := ready.Data.Tunnels[0]
	if one.TunnelID != two.TunnelID || one.URL != two.URL || one.Generation == two.Generation {
		t.Fatal("saved route did not resume conditionally")
	}
	if out := public(origin.Host, "GET", "/", nil); string(out) != "root marker" {
		t.Fatal("root route failed")
	}
	stop(cmd, scanner, stderr)
	first.Shutdown(ctx)
	current.Store(NewServer(registry, "example.test", false))
	cmd, scanner, stderr, ready = start(true)
	three := ready.Data.Tunnels[0]
	if three.TunnelID != one.TunnelID || three.URL != one.URL {
		t.Fatal("server restart changed saved URL")
	}
	stop(cmd, scanner, stderr)
	current.Load().Shutdown(ctx)
	proxy.Close() // Local inspection/replay must not need an online server.
	run := func(args ...string) []byte {
		cmd := command(args...)
		var out, errout bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errout
		if err := cmd.Run(); err != nil || errout.Len() != 0 {
			t.Fatalf("%v: %v %s %s", args, err, &errout, &out)
		}
		return out.Bytes()
	}
	listed := run("requests", "list", "--name", "combined", "--output", "json")
	var listing struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(listed, &listing) != nil || len(listing.Data) != 1 || listing.Data[0].ID != requestID {
		t.Fatal("capture opt-in/inspection changed across restart")
	}
	out := run("replay", requestID, "--name", "combined", "--yes", "--output", "jsonl")
	if !bytes.Contains(out, []byte(`"completed"`)) || bytes.Contains(out, body) || bytes.Contains(out, []byte("synthetic-private-token")) {
		t.Fatal("unsafe or incomplete replay result")
	}
	if apiCalls.Load() != 2 || rootCalls.Load() != 1 {
		t.Fatalf("unexpected side effects API=%d root=%d", apiCalls.Load(), rootCalls.Load())
	}
}
