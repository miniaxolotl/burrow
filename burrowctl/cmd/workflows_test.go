package cmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
