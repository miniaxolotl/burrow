package internal

import (
	"bytes"
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
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNativeCaptureTUIUsesDurableReplayAndStableSelection(t *testing.T) {
	if os.Getenv("BURROW_TEST_SECURE_BACKEND") != "1" {
		t.Skip("requires explicitly isolated credential backend")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux is required for a real terminal fixture")
	}
	_, proxy, _ := privacyServer(t)
	var calls atomic.Int32
	var outcome atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != "synthetic-tui-body" {
			t.Error("replay changed body")
		}
		switch outcome.Load() {
		case 1:
			w.WriteHeader(500)
		case 2:
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
			return
		}
		fmt.Fprint(w, "accepted")
	}))
	defer upstream.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(upstream.URL, "http://"))
	if _, err := strconv.Atoi(port); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Second)
	defer cancel()
	base := t.TempDir()
	binary := filepath.Join(base, "burrowctl")
	if out, err := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../burrowctl").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	socket := filepath.Join(base, "tmux.sock")
	home := filepath.Join(base, "home")
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal(err)
	}
	tmux := func(args ...string) string {
		t.Helper()
		cmd := exec.CommandContext(ctx, "tmux", append([]string{"-S", socket}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("tmux %v: %v %s", args, err, out)
		}
		return string(out)
	}
	defer func() { exec.Command("tmux", "-S", socket, "kill-server").Run() }()
	tmux("new-session", "-d", "-s", "verify", "-x", "110", "-y", "32", "-e", "HOME="+home, "-e", "DO_NOT_TRACK=1", binary, "tunnel", port, "--name", "tui-fixture", "--capture", "--server", strings.TrimPrefix(proxy.URL, "http://"))
	wait := func(text string) string {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for {
			frame := tmux("capture-pane", "-p", "-t", "verify:0.0")
			if strings.Contains(frame, text) {
				return frame
			}
			if time.Now().After(deadline) {
				t.Fatalf("terminal never showed %q:\n%s", text, frame)
			}
			time.Sleep(30 * time.Millisecond)
		}
	}
	frame := wait(".example.test")
	urlPattern := regexp.MustCompile(`http://([a-z0-9-]+\.example\.test)`)
	matched := urlPattern.FindStringSubmatch(frame)
	if len(matched) != 2 {
		t.Fatalf("no public fixture URL:\n%s", frame)
	}
	post := func() {
		req, _ := http.NewRequestWithContext(ctx, "POST", proxy.URL+"/hook", strings.NewReader("synthetic-tui-body"))
		req.Host = matched[1]
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("forwarding failed %d", resp.StatusCode)
		}
	}
	post()
	tmux("send-keys", "-t", "verify:0.0", "x")
	frame = wait("1 records")
	selectedPattern := regexp.MustCompile(`› ([A-Z2-7]{26})`)
	selection := selectedPattern.FindStringSubmatch(frame)
	if len(selection) != 2 {
		t.Fatalf("missing capture selection:\n%s", frame)
	}
	row := func(frame, id string) int {
		for index, line := range strings.Split(frame, "\n") {
			if strings.Contains(line, "› "+id) {
				return index
			}
		}
		return -1
	}
	position := row(frame, selection[1])
	post()
	frame = wait("2 records")
	if !strings.Contains(frame, "erase") || !strings.Contains(frame, "back") || !strings.Contains(frame, "quit") {
		t.Fatalf("terminal clipped the action footer:\n%s", frame)
	}
	if row(frame, selection[1]) != position {
		t.Fatalf("refresh moved selected row:\n%s", frame)
	}
	save := func(name string) {
		dir := os.Getenv("BURROW_TEST_TUI_CAPTURE_DIR")
		if dir == "" {
			return
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		ansi := tmux("capture-pane", "-p", "-e", "-t", "verify:0.0")
		if err := os.WriteFile(filepath.Join(dir, name+".ansi"), []byte(ansi), 0600); err != nil {
			t.Fatal(err)
		}
	}
	save("capture-list")
	tmux("send-keys", "-t", "verify:0.0", "r")
	wait("Approve one local replay?")
	save("capture-confirmation")
	tmux("send-keys", "-t", "verify:0.0", "n")
	frame = wait("2 records")
	if row(frame, selection[1]) != position {
		t.Fatal("cancelled confirmation lost selection/scroll")
	}
	tmux("send-keys", "-t", "verify:0.0", "r")
	wait("Approve one local replay?")
	tmux("send-keys", "-t", "verify:0.0", "y")
	frame = wait("completed · HTTP 200")
	save("capture-replay-completed")
	if calls.Load() != 3 {
		t.Fatalf("replay count %d, expected exactly one local resend", calls.Load())
	}
	outcome.Store(1)
	tmux("send-keys", "-t", "verify:0.0", "r")
	wait("Approve one local replay?")
	tmux("send-keys", "-t", "verify:0.0", "y")
	wait("completed · HTTP 500")
	wait("Replay returned an error")
	save("capture-replay-http-failure")
	outcome.Store(2)
	tmux("send-keys", "-t", "verify:0.0", "r")
	wait("Approve one local replay?")
	tmux("send-keys", "-t", "verify:0.0", "y")
	wait("uncertain · HTTP unknown")
	save("capture-replay-uncertain")
	if calls.Load() != 5 {
		t.Fatal("failed/uncertain replay automatically retried")
	}
	tmux("send-keys", "-t", "verify:0.0", "r")
	wait("Linked resend of uncertain/recovering job")
	tmux("send-keys", "-t", "verify:0.0", "n")
	wait("2 records")
	tmux("send-keys", "-t", "verify:0.0", "d")
	wait("Erase this local capture?")
	tmux("send-keys", "-t", "verify:0.0", "y")
	wait("1 records")
	frame = tmux("capture-pane", "-p", "-t", "verify:0.0")
	if bytes.Contains([]byte(frame), []byte("synthetic-tui-body")) {
		t.Fatal("body leaked into terminal")
	}
	tmux("send-keys", "-t", "verify:0.0", "q")
}
