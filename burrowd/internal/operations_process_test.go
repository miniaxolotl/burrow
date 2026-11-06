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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// Real binaries, real SIGTERM, default 15s drain, retained disposable Redis and
// an isolated owner HOME. Nothing is sent to a public tunnel/provider.
func TestOperationsProcessesWithRealRedis(t *testing.T) {
	redisURL := os.Getenv("BURROW_TEST_REDIS")
	if redisURL == "" || os.Getenv("BURROW_TEST_OPERATIONS") != "1" {
		t.Skip("requires explicitly disposable real Redis and operations acceptance")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	base := t.TempDir()
	home := filepath.Join(base, "home")
	os.Mkdir(home, 0700)
	environment := func(home string) []string {
		env := []string{}
		for _, entry := range os.Environ() {
			key, _, _ := strings.Cut(entry, "=")
			if key == "HOME" || strings.HasPrefix(key, "BURROW_") || strings.HasPrefix(key, "XDG_") {
				continue
			}
			env = append(env, entry)
		}
		return append(env, "HOME="+home, "DO_NOT_TRACK=1", "BURROW_TELEMETRY=0")
	}
	binaries := map[string]string{}
	for _, name := range []string{"burrowd", "burrowctl"} {
		binary := filepath.Join(base, name)
		out, err := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../"+name).CombinedOutput()
		if err != nil {
			t.Fatalf("build %s: %v %s", name, err, out)
		}
		binaries[name] = binary
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serverPort := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	origin := fmt.Sprintf("http://127.0.0.1:%d", serverPort)
	httpClient := &http.Client{Timeout: 25 * time.Second}
	defer httpClient.CloseIdleConnections()
	startServer := func() (*exec.Cmd, <-chan error) {
		t.Helper()
		cmd := exec.CommandContext(ctx, binaries["burrowd"], "serve", "--port", fmt.Sprint(serverPort), "--domain", "example.test", "--redis-url", redisURL)
		cmd.Env = environment(base)
		var output bytes.Buffer
		cmd.Stdout = &output
		cmd.Stderr = &output
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { cmd.Process.Kill() })
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		deadline := time.Now().Add(8 * time.Second)
		for {
			request, stop := context.WithTimeout(ctx, time.Second)
			req, _ := http.NewRequestWithContext(request, "GET", origin+"/ready", nil)
			resp, err := httpClient.Do(req)
			stop()
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					return cmd, done
				}
			}
			select {
			case err := <-done:
				t.Fatalf("server startup: %v %s", err, &output)
			default:
			}
			if time.Now().After(deadline) {
				t.Fatal("server readiness deadline exceeded")
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	server, serverDone := startServer()
	var effects, calls atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path == "/hold" {
			w.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(w, "data: admitted\n\n")
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		if r.Method == "POST" {
			effects.Add(1)
			once.Do(func() { close(entered) })
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		io.WriteString(w, "synthetic-effect")
	}))
	defer app.Close()
	_, port, _ := net.SplitHostPort(app.Listener.Addr().String())
	config := filepath.Join(base, "routes.json")
	os.WriteFile(config, []byte(fmt.Sprintf(`{"version":1,"routes":[{"path":"/api","port":%s,"strip_prefix":true}]}`, port)), 0600)
	cli := exec.CommandContext(ctx, binaries["burrowctl"], "tunnel", "--config", config, "--inspect", "--name", "operations", "--server", strings.TrimPrefix(origin, "http://"), "--output", "jsonl")
	cli.Env = environment(home)
	stdout, err := cli.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var cliErrors bytes.Buffer
	cli.Stderr = &cliErrors
	if err := cli.Start(); err != nil {
		t.Fatal(err)
	}
	defer cli.Process.Kill()
	type frame struct {
		Type string `json:"type"`
		Data struct {
			Reason, Code, Generation string
			RemainingMS              int64 `json:"remaining_ms"`
			Tunnels                  []struct {
				TunnelID        string `json:"tunnel_id"`
				Generation, URL string
			} `json:"tunnels"`
		} `json:"data"`
	}
	events := make(chan frame, 64)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			var e frame
			if json.Unmarshal(scanner.Bytes(), &e) == nil {
				events <- e
			}
		}
		close(events)
	}()
	next := func(kind string) frame {
		t.Helper()
		for {
			select {
			case e, ok := <-events:
				if !ok {
					t.Fatalf("CLI ended before %s", kind)
				}
				if e.Type == kind {
					return e
				}
			case <-ctx.Done():
				t.Fatalf("waiting for %s: %v", kind, ctx.Err())
			}
		}
	}
	ready := next("ready")
	if len(ready.Data.Tunnels) != 1 {
		t.Fatal("partial readiness")
	}
	identity := ready.Data.Tunnels[0]
	if calls.Load() != 0 {
		t.Fatal("startup sent application probes")
	}
	request := func(method, path string) (*http.Response, error) {
		var body io.Reader
		if method == "POST" {
			body = strings.NewReader("synthetic-body")
		}
		req, _ := http.NewRequestWithContext(ctx, method, origin+"/api"+path, body)
		req.Host = identity.TunnelID + ".example.test"
		return httpClient.Do(req)
	}
	stream, err := request("GET", "/hold")
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Body.Close()
	posted := make(chan error, 1)
	go func() {
		resp, err := request("POST", "/effect")
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			err = readErr
			if resp.StatusCode != 200 || string(body) != "synthetic-effect" {
				err = fmt.Errorf("admitted POST interrupted: %d %q", resp.StatusCode, body)
			}
		}
		posted <- err
	}()
	<-entered
	started := time.Now()
	server.Process.Signal(syscall.SIGTERM)
	invalidated := next("not_ready")
	if invalidated.Data.Reason != "server_draining" {
		t.Fatalf("wrong invalidation: %+v", invalidated)
	}
	notice := next("draining")
	if notice.Data.RemainingMS <= 0 || notice.Data.RemainingMS > 15000 {
		t.Fatalf("unbounded notice: %+v", notice)
	}
	late, err := request("POST", "/late")
	if err != nil {
		t.Fatal(err)
	}
	late.Body.Close()
	if late.StatusCode != 503 || late.Header.Get("Retry-After") == "" {
		t.Fatal("late admission was not rejected")
	}
	close(release)
	if err := <-posted; err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err == nil {
		t.Fatal("forced deadline was reported as clean shutdown")
	}
	elapsed := time.Since(started)
	if elapsed < 14500*time.Millisecond || elapsed > 22*time.Second {
		t.Fatalf("default drain budget violated: %s", elapsed)
	}
	if _, err := io.ReadAll(stream.Body); err == nil {
		t.Fatal("forced SSE shutdown appeared complete")
	}
	replacement, replacementDone := startServer()
	after := next("ready")
	if len(after.Data.Tunnels) != 1 || after.Data.Tunnels[0].TunnelID != identity.TunnelID || after.Data.Tunnels[0].URL != identity.URL || after.Data.Tunnels[0].Generation == identity.Generation {
		t.Fatal("reconnect lost same-ID generation fencing")
	}
	resp, err := request("GET", "/after")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if effects.Load() != 1 || calls.Load() != 3 {
		t.Fatalf("automatic resend/probe: effects=%d calls=%d", effects.Load(), calls.Load())
	}
	cli.Process.Signal(syscall.SIGTERM)
	for range events {
	}
	if err := cli.Wait(); err != nil {
		t.Fatalf("CLI cancellation: %v %s", err, &cliErrors)
	}
	replacement.Process.Signal(syscall.SIGTERM)
	if err := <-replacementDone; err != nil {
		t.Fatal(err)
	}
	t.Logf("Default SIGTERM drain %s: admitted POST completed once, SSE interrupted, late POST not forwarded; same URL re-registered with new generation", elapsed)
}
