package internal

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNativeOperationsTerminal(t *testing.T) {
	if os.Getenv("BURROW_TEST_OPERATIONS") != "1" {
		t.Skip("explicit isolated native terminal acceptance")
	}
	s, _, _ := privacyServer(t)
	if err := s.SetDrainTimeout(3 * time.Second); err != nil {
		t.Fatal(err)
	}
	var current atomic.Pointer[Server]
	current.Store(s)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { current.Load().ServeHTTP(w, r) }))
	defer proxy.Close()
	defer func() { current.Load().Shutdown(context.Background()) }()
	var status, calls atomic.Int32
	status.Store(500)
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		io.Copy(io.Discard, r.Body)
		if r.URL.Path == "/hold" {
			w.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(w, "data: open\n\n")
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		w.WriteHeader(int(status.Load()))
		fmt.Fprint(w, "synthetic")
	}))
	defer app.Close()
	_, port, _ := net.SplitHostPort(app.Listener.Addr().String())
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	base := t.TempDir()
	binary := filepath.Join(base, "burrowctl")
	home := filepath.Join(base, "home")
	os.Mkdir(home, 0700)
	if out, err := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../burrowctl").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	socket := filepath.Join(base, "tmux.sock")
	tmux := func(args ...string) string {
		t.Helper()
		cmd := exec.CommandContext(ctx, "tmux", append([]string{"-f", "/dev/null", "-S", socket}, args...)...)
		for _, entry := range os.Environ() {
			key, _, _ := strings.Cut(entry, "=")
			if key == "HOME" || strings.HasPrefix(key, "BURROW_") || strings.HasPrefix(key, "XDG_") {
				continue
			}
			cmd.Env = append(cmd.Env, entry)
		}
		cmd.Env = append(cmd.Env, "HOME="+home, "DO_NOT_TRACK=1", "BURROW_TELEMETRY=0")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("tmux %v: %v %s", args, err, out)
		}
		return string(out)
	}
	defer exec.Command("tmux", "-S", socket, "kill-server").Run()
	tmux("new-session", "-d", "-s", "operations", "-x", "80", "-y", "24", "-e", "HOME="+home, "-e", "DO_NOT_TRACK=1", "-e", "BURROW_TELEMETRY=0", binary, "tunnel", port, "--inspect", "--server", strings.TrimPrefix(proxy.URL, "http://"))
	wait := func(text string) string {
		t.Helper()
		deadline := time.Now().Add(6 * time.Second)
		for {
			frame := tmux("capture-pane", "-p", "-t", "operations:0.0")
			if strings.Contains(frame, text) {
				return frame
			}
			if time.Now().After(deadline) {
				t.Fatalf("missing %q:\n%s", text, frame)
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	save := func(name string) {
		t.Helper()
		dir := os.Getenv("BURROW_TEST_OPERATIONS_CAPTURE_DIR")
		if dir == "" {
			return
		}
		os.MkdirAll(dir, 0700)
		data := tmux("capture-pane", "-p", "-e", "-t", "operations:0.0")
		if err := os.WriteFile(filepath.Join(dir, name+".ansi"), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		size := strings.TrimSpace(tmux("display-message", "-p", "-t", "operations:0.0", "#{pane_width}x#{pane_height}"))
		if err := os.WriteFile(filepath.Join(dir, name+".size"), []byte(size), 0600); err != nil {
			t.Fatal(err)
		}
	}
	frame := wait(".example.test")
	match := regexp.MustCompile(`http://([a-z0-9-]+\.example\.test)`).FindStringSubmatch(frame)
	if len(match) != 2 {
		t.Fatalf("missing URL: %s", frame)
	}
	post := func(path string) {
		t.Helper()
		req, _ := http.NewRequestWithContext(ctx, "POST", proxy.URL+path, strings.NewReader("synthetic"))
		req.Host = match[1]
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	post("/first")
	tmux("send-keys", "-t", "operations:0.0", "l")
	wait("server logs")
	tmux("send-keys", "-t", "operations:0.0", "v")
	wait("local live metadata")
	wait("/first")
	selected := func(frame string) string {
		for _, line := range strings.Split(frame, "\n") {
			if strings.Contains(line, "▶") {
				return line
			}
		}
		return ""
	}
	before := selected(wait("/first"))
	post("/later")
	frame = wait("2 matching / 2 retained")
	if before == "" || selected(frame) != before {
		t.Fatalf("paused selection moved:\n%s", frame)
	}
	save("inspection-paused-80x24")
	tmux("send-keys", "-t", "operations:0.0", "Enter")
	wait("REQUEST DETAILS")
	save("inspection-details-80x24")
	tmux("send-keys", "-t", "operations:0.0", "Escape")
	wait("DURATION")
	tmux("send-keys", "-t", "operations:0.0", "/")
	tmux("send-keys", "-t", "operations:0.0", "-l", "first")
	wait("search ›")
	post("/another")
	wait("3 matching / 3 retained")
	frame = wait("search ›")
	if !strings.Contains(frame, "first") {
		t.Fatal("slow refresh overwrote filter draft")
	}
	tmux("send-keys", "-t", "operations:0.0", "Enter")
	wait("1 matching / 3 retained")
	save("inspection-filtered-80x24")
	for _, size := range []struct{ w, h, name string }{{"110", "32", "inspection-filtered-110x32"}, {"60", "20", "inspection-filtered-60x20"}} {
		tmux("resize-window", "-t", "operations:0", "-x", size.w, "-y", size.h)
		wait("1 matching / 3 retained")
		save(size.name)
	}
	tmux("resize-window", "-t", "operations:0", "-x", "80", "-y", "24")
	tmux("send-keys", "-t", "operations:0.0", "Escape")
	wait("TUNNEL ID")
	tmux("send-keys", "-t", "operations:0.0", "h")
	wait("scoped health")
	wait("HTTP 500")
	save("health-application-error")
	status.Store(200)
	post("/recovered")
	wait("HTTP 200")
	save("health-recovered")
	tmux("resize-window", "-t", "operations:0", "-x", "110", "-y", "32")
	tmux("send-keys", "-t", "operations:0.0", "Escape")
	wait("TUNNEL ID")
	req, _ := http.NewRequestWithContext(ctx, "GET", proxy.URL+"/hold", nil)
	req.Host = match[1]
	stream, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	shutdown := make(chan error, 1)
	go func() { shutdown <- s.Shutdown(ctx) }()
	wait("draining")
	save("draining-110x32")
	stream.Body.Close()
	if err := <-shutdown; err != nil {
		t.Fatal(err)
	}
	current.Store(NewServer(s.registry, "example.test", false))
	frame = wait("active (1 recon)")
	if !strings.Contains(frame, match[1]) {
		t.Fatal("reconnect changed the URL")
	}
	save("reconnected-110x32")
	post("/after-reconnect")
	tmux("resize-window", "-t", "operations:0", "-x", "80", "-y", "24")
	tmux("send-keys", "-t", "operations:0.0", "h")
	wait("HTTP 200")
	app.Close()
	tmux("send-keys", "-t", "operations:0.0", "r")
	wait("tcp_unavailable")
	save("health-tcp-failed")
	if calls.Load() != 6 {
		t.Fatalf("health/inspection issued synthetic app requests: %d", calls.Load())
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "burrow", "captures")); !os.IsNotExist(err) {
		t.Fatal("inspection opened capture storage")
	}
	tmux("send-keys", "-t", "operations:0.0", "q")
}
