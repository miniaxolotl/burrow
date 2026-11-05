package cmd_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestWorkflowNamedModeDoesNotDowngradeToLegacyServer(t *testing.T) {
	var connects atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tunnels/connect" {
			connects.Add(1)
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	command, home := workflowCommand(t, "tunnel", "3000", "--name", "webhook", "--server", strings.TrimPrefix(server.URL, "http://"), "--output", "jsonl")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err == nil {
		t.Fatal("named mode downgraded")
	}
	if connects.Load() != 0 {
		t.Fatal("old server received persistent connect")
	}
	if stderr.Len() != 0 {
		t.Fatalf("unstructured error: %s", &stderr)
	}
	if e := oneWorkflowEnvelope(t, stdout.Bytes()); e.Type != "error" {
		t.Fatal(e)
	}
	files, _ := filepath.Glob(filepath.Join(home, ".config", "burrow", "saved", "*.json"))
	if len(files) != 0 {
		t.Fatal("preflight created alias")
	}
}

func TestWorkflowSavedDoesNotCreateCredential(t *testing.T) {
	command, home := workflowCommand(t, "saved", "--output", "json")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err == nil {
		t.Fatal("missing owner silently created")
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "burrow", "owners")); !os.IsNotExist(err) {
		t.Fatal("local read created credential")
	}
	if e := oneWorkflowEnvelope(t, stdout.Bytes()); e.Type != "error" {
		t.Fatal(e)
	}
}
