package internal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestNamedCLIWithRetainedRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	testNamedCLI(t, "redis://"+mr.Addr())
}
func TestNamedCLIWithRealRedis(t *testing.T) {
	address := os.Getenv("BURROW_TEST_REDIS")
	if address == "" {
		t.Skip("set BURROW_TEST_REDIS to a disposable real Redis")
	}
	testNamedCLI(t, address)
}

func testNamedCLI(t *testing.T, address string) {
	t.Helper()
	redis, err := NewRedisClient(address)
	if err != nil {
		t.Fatal(err)
	}
	defer redis.Close()
	// This test requires an isolated Redis, never the operator's server database.
	registry := NewTunnelRegistry(redis, "example.test", false)
	var current atomic.Pointer[Server]
	first := NewServer(registry, "example.test", false)
	first.gracePeriod = 20 * time.Millisecond
	current.Store(first)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { current.Load().ServeHTTP(w, r) }))
	defer proxy.Close()
	defer func() { current.Load().Shutdown(context.Background()) }()
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "burrowctl")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../burrowctl")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v %s", err, output)
	}
	home := t.TempDir()
	start := func() (*exec.Cmd, *bufio.Scanner, *bytes.Buffer) {
		command := exec.CommandContext(ctx, binary, "tunnel", "3000", "--name", "acceptance", "--server", strings.TrimPrefix(proxy.URL, "http://"), "--output", "jsonl")
		for _, entry := range os.Environ() {
			key, _, _ := strings.Cut(entry, "=")
			if key == "HOME" || strings.HasPrefix(key, "BURROW_") || strings.HasPrefix(key, "XDG_") {
				continue
			}
			command.Env = append(command.Env, entry)
		}
		command.Env = append(command.Env, "HOME="+home, "DO_NOT_TRACK=1")
		out, err := command.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		stderr := new(bytes.Buffer)
		command.Stderr = stderr
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { command.Process.Kill() })
		return command, bufio.NewScanner(out), stderr
	}
	type acknowledgement struct {
		TunnelID   string `json:"tunnel_id"`
		Generation string `json:"generation"`
		URL        string `json:"url"`
	}
	ready := func(scanner *bufio.Scanner) acknowledgement {
		for scanner.Scan() {
			var e struct {
				Type string `json:"type"`
				Data struct {
					Tunnels []acknowledgement `json:"tunnels"`
				} `json:"data"`
			}
			if json.Unmarshal(scanner.Bytes(), &e) != nil {
				t.Fatalf("invalid CLI output %q", scanner.Text())
			}
			if e.Type == "error" {
				t.Fatalf("CLI error: %s", scanner.Text())
			}
			if e.Type == "ready" {
				if len(e.Data.Tunnels) != 1 {
					t.Fatal("partial barrier")
				}
				return e.Data.Tunnels[0]
			}
		}
		t.Fatalf("CLI ended before ready: %v", scanner.Err())
		return acknowledgement{}
	}
	stop := func(command *exec.Cmd, scanner *bufio.Scanner, stderr *bytes.Buffer) {
		command.Process.Signal(syscall.SIGTERM)
		for scanner.Scan() {
		}
		if err := command.Wait(); err != nil || stderr.Len() != 0 {
			t.Fatalf("stop: %v %s", err, stderr)
		}
	}
	a, scan, stderr := start()
	one := ready(scan)
	stop(a, scan, stderr)
	time.Sleep(40 * time.Millisecond) // Past the explicitly shortened disconnect grace.
	if td, err := registry.Get(ctx, one.TunnelID); err != nil || td != nil {
		t.Fatalf("fixture metadata not expired: %+v %v", td, err)
	}
	b, scan, stderr := start()
	two := ready(scan)
	if one.TunnelID != two.TunnelID || one.URL != two.URL || one.Generation == two.Generation {
		t.Fatal("restart did not reuse conditional alias")
	}
	first.Shutdown(ctx)
	second := NewServer(registry, "example.test", false)
	current.Store(second)
	three := ready(scan)
	if three.TunnelID != one.TunnelID || three.Generation == two.Generation {
		t.Fatal("server restart lost durable fencing")
	}
	stop(b, scan, stderr)
}
