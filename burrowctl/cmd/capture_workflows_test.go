package cmd_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestWorkflowCaptureFailsClosedWithoutSecureBackend(t *testing.T) {
	var connects atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connects.Add(1)
		http.Error(w, "no exposure expected", 500)
	}))
	defer server.Close()
	command, _ := workflowCommand(t, "tunnel", "3000", "--capture", "--server", strings.TrimPrefix(server.URL, "http://"), "--output", "jsonl")
	command.Env = append(command.Env, "DBUS_SESSION_BUS_ADDRESS=unix:path=/nonexistent-burrow-test-bus")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err == nil {
		t.Fatal("capture used plaintext fallback")
	}
	if connects.Load() != 0 {
		t.Fatal("capture exposed before secure storage ready")
	}
	if stderr.Len() != 0 {
		t.Fatalf("unstructured error: %s", &stderr)
	}
	if e := oneWorkflowEnvelope(t, stdout.Bytes()); e.Type != "error" {
		t.Fatal(e)
	}
}

func TestWorkflowEncryptedCaptureAndDurableReplayWithSecureBackend(t *testing.T) {
	if os.Getenv("BURROW_TEST_SECURE_BACKEND") != "1" {
		t.Skip("requires an explicitly isolated, unlocked OS credential backend")
	}
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		call := calls.Add(1)
		if string(body) != "synthetic-replay-body" || r.URL.Path != "/hook" {
			t.Error("request reconstruction changed")
		}
		if call == 1 && r.Header.Get("Authorization") != "Bearer synthetic-secret" {
			t.Error("capture changed live authorization")
		}
		if call == 2 && r.Header.Get("Authorization") != "" {
			t.Error("replay restored a stripped credential")
		}
		fmt.Fprint(w, "accepted")
	}))
	defer upstream.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(upstream.URL, "http://"))
	if _, err := strconv.Atoi(port); err != nil {
		t.Fatal(err)
	}
	server, opened := workflowTunnelPeer(t, "")
	command, home := workflowCommand(t, "tunnel", port, "--capture", "--server", server, "--output", "jsonl")
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { command.Process.Kill() })
	scanner := bufio.NewScanner(stdout)
	tunnel := ""
	for scanner.Scan() {
		var e workflowEnvelope
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatal(err)
		}
		if e.Type == "error" {
			t.Fatalf("capture startup: %s", scanner.Text())
		}
		if e.Type == "connected" {
			tunnel = *e.TunnelID
			break
		}
	}
	if tunnel == "" {
		t.Fatal("no capture connection")
	}
	session := <-opened
	stream, err := session.Open()
	if err != nil {
		t.Fatal(err)
	}
	stream.SetDeadline(time.Now().Add(3 * time.Second))
	fmt.Fprint(stream, "POST /hook?token=synthetic-query-secret HTTP/1.1\r\nHost: public.test\r\nAuthorization: Bearer synthetic-secret\r\nContent-Length: 21\r\nConnection: close\r\n\r\nsynthetic-replay-body")
	resp, err := http.ReadResponse(bufio.NewReader(stream), nil)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	stream.Close()
	captured := ""
	for scanner.Scan() {
		var e workflowEnvelope
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatal(err)
		}
		if e.Type == "capture_saved" {
			var d struct {
				ID string `json:"capture_id"`
			}
			json.Unmarshal(e.Data, &d)
			captured = d.ID
			break
		}
		if e.Type == "capture_unavailable" {
			t.Fatalf("capture unavailable: %s", e.Data)
		}
	}
	if captured == "" {
		t.Fatal("capture not durably accepted")
	}
	command.Process.Signal(syscall.SIGTERM)
	for scanner.Scan() {
	}
	if err := command.Wait(); err != nil || stderr.Len() != 0 {
		t.Fatalf("capture stop %v %s", err, &stderr)
	}
	run := func(args ...string) []byte {
		t.Helper()
		args = append(args, "--server", server)
		cmd, _ := workflowCommand(t, args...)
		cmd.Env = append(cmd.Env, "HOME="+home)
		var out, errout bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errout
		if err := cmd.Run(); err != nil || errout.Len() != 0 {
			t.Fatalf("%v: %v %s %s", args, err, &errout, &out)
		}
		return out.Bytes()
	}
	listed := run("requests", "list", "--tunnel", tunnel, "--output", "json")
	if !bytes.Contains(listed, []byte(captured)) || bytes.Contains(listed, []byte("synthetic-secret")) || bytes.Contains(listed, []byte("synthetic-replay-body")) {
		t.Fatal("unsafe/missing summary")
	}
	replayed := run("replay", captured, "--tunnel", tunnel, "--yes", "--output", "jsonl")
	if !bytes.Contains(replayed, []byte(`"completed"`)) || calls.Load() != 2 {
		t.Fatalf("durable replay did not complete exactly once: %s calls=%d", replayed, calls.Load())
	}
	run("requests", "delete", captured, "--tunnel", tunnel, "--yes", "--output", "json")
	listed = run("requests", "list", "--tunnel", tunnel, "--output", "json")
	if bytes.Contains(listed, []byte(captured)) {
		t.Fatal("deleted record returned")
	}
	jobs := run("replay", "jobs", "--tunnel", tunnel, "--output", "json")
	if !bytes.Contains(jobs, []byte(`"completed"`)) || bytes.Contains(jobs, []byte("synthetic-replay-body")) {
		t.Fatal("content-free job recovery unavailable after erasure")
	}
}

func TestWorkflowReplayRejectsFiniteJSONBeforeCredentialCreation(t *testing.T) {
	command, _ := workflowCommand(t, "replay", "request", "--tunnel", "example", "--yes", "--output", "json")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err == nil || stderr.Len() != 0 {
		t.Fatalf("streaming validation: %v %s", err, &stderr)
	}
	e := oneWorkflowEnvelope(t, stdout.Bytes())
	if e.Type != "error" || !bytes.Contains(e.Data, []byte("invalid_argument")) {
		t.Fatal(e)
	}
}
