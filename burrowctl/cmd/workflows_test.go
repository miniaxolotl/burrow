package cmd_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	"burrow/protocol"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

var workflowBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "burrowctl-workflow-tests-")
	if err != nil {
		panic(err)
	}
	workflowBinary = filepath.Join(dir, "burrowctl")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", workflowBinary, "..")
	if output, err := build.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		panic(string(output) + err.Error())
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func workflowCommand(t *testing.T, args ...string) (*exec.Cmd, string) {
	t.Helper()
	home := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	command := exec.CommandContext(ctx, workflowBinary, args...)
	command.WaitDelay = time.Second
	// Do not inherit an operator's owner, telemetry or server configuration.
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if key == "HOME" || strings.HasPrefix(key, "XDG_") || strings.HasPrefix(key, "BURROW_") || key == "DO_NOT_TRACK" {
			continue
		}
		command.Env = append(command.Env, entry)
	}
	command.Env = append(command.Env, "HOME="+home, "XDG_CONFIG_HOME="+home, "DO_NOT_TRACK=1", "BURROW_TELEMETRY=0")
	return command, home
}

func TestWorkflowHelpDoesNotCreateOwner(t *testing.T) {
	command, home := workflowCommand(t, "tunnel", "--help")
	output, err := command.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte("tunnel")) {
		t.Fatalf("help: %v: %s", err, output)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "burrow", "owners")); !os.IsNotExist(err) {
		t.Fatalf("help created an owner: %v", err)
	}
}

func TestWorkflowInvalidPortDoesNotCreateOwner(t *testing.T) {
	command, home := workflowCommand(t, "tunnel", "70000")
	if err := command.Run(); err == nil {
		t.Fatal("out-of-range port succeeded")
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "burrow", "owners")); !os.IsNotExist(err) {
		t.Fatalf("invalid port created an owner: %v", err)
	}
}

func TestWorkflowListUsesPrivateOwner(t *testing.T) {
	owner := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/tunnels" || r.URL.RawQuery != "" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		owner <- r.Header.Get("X-Tunnel-Token")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"tunnels": []any{}, "count": 0})
	}))
	defer server.Close()
	command, home := workflowCommand(t, "list", "--server", strings.TrimPrefix(server.URL, "http://"))
	output, err := command.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte("0 active tunnel(s)")) {
		t.Fatalf("list: %v: %s", err, output)
	}
	credential := <-owner
	if credential == "" || bytes.Contains(output, []byte(credential)) {
		t.Fatal("owner missing or leaked into output")
	}
	files, err := filepath.Glob(filepath.Join(home, ".config", "burrow", "owners", "*.token"))
	if err != nil || len(files) != 1 {
		t.Fatalf("owner files: %v: %v", files, err)
	}
	info, err := os.Stat(files[0])
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("private owner mode: %v: %v", info, err)
	}
}

type workflowEnvelope struct {
	SchemaVersion int             `json:"schema_version"`
	Type          string          `json:"type"`
	Timestamp     time.Time       `json:"timestamp"`
	Sequence      uint64          `json:"sequence"`
	TunnelID      *string         `json:"tunnel_id"`
	Data          json.RawMessage `json:"data"`
}

func oneWorkflowEnvelope(t *testing.T, data []byte) workflowEnvelope {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	var envelope workflowEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		t.Fatalf("invalid structured output: %v: %s", err, data)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		t.Fatalf("extra output after envelope: %v: %s", err, data)
	}
	if envelope.SchemaVersion != 1 || envelope.Sequence != 1 || envelope.Timestamp.IsZero() || envelope.Timestamp.Location() != time.UTC {
		t.Fatalf("invalid envelope metadata: %+v", envelope)
	}
	return envelope
}

func TestWorkflowListJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"tunnels":[],"count":0}`)
	}))
	defer server.Close()
	command, _ := workflowCommand(t, "list", "--server", strings.TrimPrefix(server.URL, "http://"), "--output", "json")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil || stderr.Len() != 0 {
		t.Fatalf("list json: %v, stderr=%s", err, &stderr)
	}
	envelope := oneWorkflowEnvelope(t, stdout.Bytes())
	if envelope.Type != "result" || envelope.TunnelID != nil || string(envelope.Data) != `{"tunnels":[],"count":0}` {
		t.Fatalf("unexpected list result: %+v: %s", envelope, envelope.Data)
	}
}

func TestWorkflowCloseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/tunnels/example" {
			t.Errorf("unexpected closure: %s %s", r.Method, r.URL)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	command, _ := workflowCommand(t, "close", "example", "--server", strings.TrimPrefix(server.URL, "http://"), "--output", "json")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil || stderr.Len() != 0 {
		t.Fatalf("close json: %v, stderr=%s", err, &stderr)
	}
	envelope := oneWorkflowEnvelope(t, stdout.Bytes())
	if envelope.Type != "result" || envelope.TunnelID == nil || *envelope.TunnelID != "example" {
		t.Fatalf("unexpected close result: %+v", envelope)
	}
}

func TestWorkflowJSONArgumentErrorsBeforeOwnerCreation(t *testing.T) {
	for _, args := range [][]string{
		{"tunnel", "3000", "--output", "json"},
		{"tunnel", "70000", "--output", "jsonl"},
		{"list", "--output", "json", "--not-a-flag"},
		{"list", "extra", "--output", "json"},
		{"logs", "example", "--tail", "-1", "--output", "json"},
		{"logs", "example", "--follow", "--output", "json"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			command, home := workflowCommand(t, args...)
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			err := command.Run()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 2 || stderr.Len() != 0 {
				t.Fatalf("argument error: %v, stderr=%s, stdout=%s", err, &stderr, &stdout)
			}
			envelope := oneWorkflowEnvelope(t, stdout.Bytes())
			if envelope.Type != "error" {
				t.Fatalf("argument error envelope: %+v", envelope)
			}
			if _, err := os.Stat(filepath.Join(home, ".config", "burrow", "owners")); !os.IsNotExist(err) {
				t.Fatalf("argument error created owner: %v", err)
			}
		})
	}
}

func TestWorkflowJSONErrorOmitsProviderContent(t *testing.T) {
	const secret = "synthetic-sensitive-provider-detail"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, secret, http.StatusUnauthorized)
	}))
	defer server.Close()
	command, _ := workflowCommand(t, "list", "--server", strings.TrimPrefix(server.URL, "http://"), "--output", "json")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || stderr.Len() != 0 {
		t.Fatalf("runtime error: %v, stderr=%s", err, &stderr)
	}
	envelope := oneWorkflowEnvelope(t, stdout.Bytes())
	if envelope.Type != "error" || bytes.Contains(stdout.Bytes(), []byte(secret)) {
		t.Fatalf("unsafe error output: %s", &stdout)
	}
}

// This is a real WebSocket/yamux peer, not a fake CLI lifecycle. It exposes only
// an httptest loopback address and never registers a public tunnel or uses Redis.
func workflowTunnelPeer(t *testing.T, rejectPort string) (string, <-chan *yamux.Session) {
	return workflowTunnelPeerWithPolicy(t, func(r *http.Request) int {
		if r.URL.Query().Get("port") == rejectPort {
			return http.StatusForbidden
		}
		return 0
	})
}

func workflowTunnelPeerWithPolicy(t *testing.T, reject func(*http.Request) int) (string, <-chan *yamux.Session) {
	t.Helper()
	opened := make(chan *yamux.Session, 4)
	var mu sync.Mutex
	var sessions []*yamux.Session
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			io.WriteString(w, `{"status":"ok"}`)
			return
		}
		if r.URL.Path != "/tunnels/connect" || r.Header.Get("X-Tunnel-Token") == "" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		if status := reject(r); status != 0 {
			http.Error(w, "synthetic-private-rejection-body", status)
			return
		}
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, http.Header{"X-Tunnel-URL": {"https://" + r.URL.Query().Get("tunnel_id") + ".example.test"}})
		if err != nil {
			return
		}
		defer conn.Close()
		cfg := yamux.DefaultConfig()
		cfg.LogOutput = io.Discard
		session, err := yamux.Server(&protocol.WsConn{Conn: conn}, cfg)
		if err != nil {
			t.Error(err)
			return
		}
		defer session.Close()
		mu.Lock()
		sessions = append(sessions, session)
		mu.Unlock()
		opened <- session
		<-session.CloseChan()
	}))
	t.Cleanup(func() {
		server.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, session := range sessions {
			session.Close()
		}
	})
	return strings.TrimPrefix(server.URL, "http://"), opened
}

func TestWorkflowHeadlessJSONLForwardsAndStopsOnSignal(t *testing.T) {
	server, opened := workflowTunnelPeer(t, "")
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fmt.Fprintf(w, "%s %s %s", r.Method, r.URL.RequestURI(), body)
	}))
	defer local.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(local.URL, "http://"))
	command, _ := workflowCommand(t, "tunnel", port, "--server", server, "--headless", "--output", "jsonl")
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
	var events []workflowEnvelope
	for scanner.Scan() {
		var event workflowEnvelope
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("non-JSON lifecycle: %q", scanner.Text())
		}
		events = append(events, event)
		if event.Type == "connected" {
			var data struct {
				ObservedAt time.Time `json:"observed_at"`
			}
			if err := json.Unmarshal(event.Data, &data); err != nil || data.ObservedAt.IsZero() || data.ObservedAt.After(event.Timestamp) {
				t.Fatalf("lifecycle has no trustworthy observation time: %v: %s", err, event.Data)
			}
			break
		}
	}
	if len(events) < 2 || events[len(events)-1].Type != "connected" {
		t.Fatalf("no connection lifecycle; stderr=%s", &stderr)
	}
	session := <-opened
	stream, err := session.Open()
	if err != nil {
		t.Fatal(err)
	}
	stream.SetDeadline(time.Now().Add(3 * time.Second))
	fmt.Fprint(stream, "POST /hook?test=1 HTTP/1.1\r\nHost: local.example\r\nContent-Length: 3\r\nConnection: close\r\n\r\nabc")
	response, err := http.ReadResponse(bufio.NewReader(stream), nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	stream.Close()
	if err != nil || string(body) != "POST /hook?test=1 abc" {
		t.Fatalf("forwarded response: %q: %v", body, err)
	}
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	for scanner.Scan() {
		var event workflowEnvelope
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	if err := command.Wait(); err != nil || stderr.Len() != 0 {
		t.Fatalf("signal stop: %v; stderr=%s", err, &stderr)
	}
	if scanner.Err() != nil || events[len(events)-1].Type != "stopped" {
		t.Fatalf("missing clean stop: %v: %+v", scanner.Err(), events)
	}
	for i, event := range events {
		if event.Sequence != uint64(i+1) || event.Type == "ready" {
			t.Fatalf("invalid sequence or invented readiness: %+v", events)
		}
	}
	select {
	case <-session.CloseChan():
	case <-time.After(time.Second):
		t.Fatal("signal left tunnel transport open")
	}
}

func TestWorkflowHeadlessPartialStartupRollsBack(t *testing.T) {
	server, opened := workflowTunnelPeer(t, "3001")
	command, _ := workflowCommand(t, "tunnel", "3000", "3001", "--server", server, "--headless", "--output", "jsonl")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || stderr.Len() != 0 {
		t.Fatalf("startup failure: %v, stderr=%s, stdout=%s", err, &stderr, &stdout)
	}
	decoder := json.NewDecoder(&stdout)
	var last workflowEnvelope
	for decoder.More() {
		if err := decoder.Decode(&last); err != nil {
			t.Fatal(err)
		}
	}
	if last.Type != "error" {
		t.Fatalf("missing startup error: %+v", last)
	}
	session := <-opened
	select {
	case <-session.CloseChan():
	case <-time.After(time.Second):
		t.Fatal("partial startup left first tunnel connected")
	}
}

func TestWorkflowFollowJSONLDoesNotRepeatEntries(t *testing.T) {
	var calls atomic.Int32
	polledAgain := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 2 {
			close(polledAgain)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `[{"id":"one","method":"POST","path":"/first"},{"id":"two","method":"POST","path":"/second"}]`)
	}))
	defer server.Close()
	command, _ := workflowCommand(t, "logs", "example", "--follow", "--tail", "1", "--server", strings.TrimPrefix(server.URL, "http://"), "--output", "jsonl")
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
	if !scanner.Scan() {
		t.Fatalf("no log event: %v", scanner.Err())
	}
	var event workflowEnvelope
	if err := json.Unmarshal(scanner.Bytes(), &event); err != nil || event.Type != "snapshot" || !bytes.Contains(event.Data, []byte(`"retained":2`)) {
		t.Fatalf("missing bounded snapshot: %s", scanner.Text())
	}
	if !scanner.Scan() {
		t.Fatal("missing initial filtered tail log")
	}
	if err := json.Unmarshal(scanner.Bytes(), &event); err != nil || event.Type != "log" || !bytes.Contains(event.Data, []byte("/second")) {
		t.Fatalf("wrong log event: %v: %s", err, scanner.Text())
	}
	select {
	case <-polledAgain:
	case <-time.After(3 * time.Second):
		t.Fatal("follow never refreshed its snapshot")
	}
	command.Process.Signal(syscall.SIGTERM)
	for scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil || event.Type == "log" {
			t.Fatalf("duplicate or invalid follow: %v: %s", err, scanner.Text())
		}
	}
	if err := command.Wait(); err != nil || stderr.Len() != 0 {
		t.Fatalf("follow stop: %v, stderr=%s", err, &stderr)
	}
}

func TestWorkflowSignalCancelsManagementRequest(t *testing.T) {
	started, cancelled := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer server.Close()
	command, _ := workflowCommand(t, "list", "--server", strings.TrimPrefix(server.URL, "http://"), "--output", "json")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { command.Process.Kill() })
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("management request did not start")
	}
	command.Process.Signal(syscall.SIGTERM)
	if err := command.Wait(); err != nil || stderr.Len() != 0 {
		t.Fatalf("management stop: %v, stderr=%s, stdout=%s", err, &stderr, &stdout)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("signal did not cancel the HTTP request")
	}
	if event := oneWorkflowEnvelope(t, stdout.Bytes()); event.Type != "stopped" {
		t.Fatalf("wrong cancelled result: %+v", event)
	}
}

func TestWorkflowTextSignalCancelsManagementRequest(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	command, _ := workflowCommand(t, "list", "--server", strings.TrimPrefix(server.URL, "http://"))
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { command.Process.Kill() })
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("management request did not start")
	}
	command.Process.Signal(syscall.SIGTERM)
	if err := command.Wait(); err != nil || stderr.Len() != 0 || stdout.Len() != 0 {
		t.Fatalf("text management stop: %v, stderr=%s, stdout=%s", err, &stderr, &stdout)
	}
}

func TestWorkflowPipedTunnelDefaultsToPlainHeadless(t *testing.T) {
	server, _ := workflowTunnelPeer(t, "")
	command, _ := workflowCommand(t, "tunnel", "3000", "--server", server)
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
	connected := false
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "\x1b") {
			t.Fatal("piped output entered the TUI")
		}
		if strings.HasPrefix(scanner.Text(), "connected ") {
			connected = true
			break
		}
	}
	if !connected {
		t.Fatal("piped tunnel did not start headlessly")
	}
	command.Process.Signal(syscall.SIGTERM)
	io.Copy(io.Discard, stdout)
	if err := command.Wait(); err != nil || stderr.Len() != 0 {
		t.Fatalf("piped stop: %v; stderr=%s", err, &stderr)
	}
}

func TestWorkflowTerminalReconnectFailureStopsInvocation(t *testing.T) {
	var attempts atomic.Int32
	server, opened := workflowTunnelPeerWithPolicy(t, func(r *http.Request) int {
		if attempts.Add(1) > 1 {
			return http.StatusGone
		}
		return 0
	})
	command, _ := workflowCommand(t, "tunnel", "3000", "--server", server, "--output", "jsonl")
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
	var types []string
	var id string
	for scanner.Scan() {
		var event workflowEnvelope
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(scanner.Bytes(), []byte("synthetic-private-rejection-body")) {
			t.Fatal("provider content entered lifecycle output")
		}
		types = append(types, event.Type)
		if event.TunnelID != nil {
			if id != "" && id != *event.TunnelID {
				t.Fatal("reconnect changed tunnel ID")
			}
			id = *event.TunnelID
		}
		if event.Type == "connected" {
			(<-opened).Close()
		}
	}
	err = command.Wait()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || stderr.Len() != 0 {
		t.Fatalf("terminal failure: %v, stderr=%s", err, &stderr)
	}
	want := "connecting,connected,disconnected,reconnecting,failed,error"
	if strings.Join(types, ",") != want || attempts.Load() != 2 {
		t.Fatalf("wrong terminal lifecycle: %v, attempts=%d", types, attempts.Load())
	}
}

func TestWorkflowJSONUsesNumericUnitsAndUnknownHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/tunnels" {
			io.WriteString(w, `{"tunnels":[{"tunnel_id":"example","port":3000,"latency":"2ms","total_size":42}],"count":1}`)
			return
		}
		io.WriteString(w, `[{"id":"request","duration":"1.5ms","size":1024,"status_code":0}]`)
	}))
	defer server.Close()
	for _, args := range [][]string{{"list"}, {"logs", "example"}} {
		t.Run(args[0], func(t *testing.T) {
			args = append(args, "--server", strings.TrimPrefix(server.URL, "http://"), "--output", "json")
			command, _ := workflowCommand(t, args...)
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			if err := command.Run(); err != nil || stderr.Len() != 0 {
				t.Fatalf("numeric result: %v; stderr=%s", err, &stderr)
			}
			event := oneWorkflowEnvelope(t, stdout.Bytes())
			if args[0] == "list" {
				var data struct {
					Tunnels []map[string]any `json:"tunnels"`
				}
				json.Unmarshal(event.Data, &data)
				if len(data.Tunnels) != 1 || data.Tunnels[0]["latency_ms"] != float64(2) || data.Tunnels[0]["total_size_bytes"] != float64(42) {
					t.Fatalf("missing numeric tunnel units: %s", event.Data)
				}
			} else {
				var data []map[string]any
				json.Unmarshal(event.Data, &data)
				if len(data) != 1 || data[0]["duration_ms"] != float64(1) || data[0]["size_bytes"] != float64(1024) || data[0]["status_code"] != nil {
					t.Fatalf("missing numeric log units or fabricated status: %s", event.Data)
				}
			}
		})
	}
}

func TestWorkflowBrokenOutputDrainsBeforeRuntimeExit(t *testing.T) {
	server, opened := workflowTunnelPeer(t, "")
	command, _ := workflowCommand(t, "tunnel", "3000", "--server", server, "--output", "jsonl")
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
	for scanner.Scan() {
		var event workflowEnvelope
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if event.Type == "connected" {
			break
		}
	}
	stdout.Close()
	session := <-opened
	session.Close() // Cause the next lifecycle write to observe the broken pipe.
	err = command.Wait()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || !strings.Contains(stderr.String(), "command output unavailable") {
		t.Fatalf("broken output bypassed clean failure: %v; stderr=%s", err, &stderr)
	}
}

func TestWorkflowTextOutputFailureIsNotSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
		} else if r.URL.Path == "/tunnels" {
			io.WriteString(w, `{"tunnels":[],"count":0}`)
		} else {
			io.WriteString(w, `[{"id":"one","path":"/request"}]`)
		}
	}))
	defer server.Close()
	for _, args := range [][]string{{"list"}, {"close", "example"}, {"logs", "example"}} {
		t.Run(args[0], func(t *testing.T) {
			args = append(args, "--server", strings.TrimPrefix(server.URL, "http://"))
			command, _ := workflowCommand(t, args...)
			read, write, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			read.Close()
			defer write.Close()
			command.Stdout = write
			err = command.Run()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 {
				t.Fatalf("text write failure reported success: %v", err)
			}
		})
	}
}
