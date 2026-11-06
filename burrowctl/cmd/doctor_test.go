package cmd_test

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDoctorChecksTCPWithoutAppRequestOrOwner(t *testing.T) {
	var appRequests, health atomic.Int32
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { appRequests.Add(1) }))
	defer app.Close()
	_, port, _ := net.SplitHostPort(app.Listener.Addr().String())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" || r.Header.Get("X-Tunnel-Token") != "" {
			t.Error("doctor sent management/application request")
		}
		health.Add(1)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	command, home := workflowCommand(t, "doctor", port, "--server", strings.TrimPrefix(server.URL, "http://"), "--output", "json")
	out, err := command.CombinedOutput()
	if err != nil || !bytes.Contains(out, []byte(`"tcp_connected"`)) {
		t.Fatalf("doctor: %v %s", err, out)
	}
	if appRequests.Load() != 0 || health.Load() != 1 {
		t.Fatal("doctor did more than one apex read and a TCP check")
	}
	entries, _ := os.ReadDir(home)
	if len(entries) != 0 {
		t.Fatalf("doctor created local state: %v", entries)
	}
}

func TestDoctorValidatesBeforeIO(t *testing.T) {
	for _, args := range [][]string{{"doctor", "70000"}, {"doctor"}, {"doctor", "3", "--config", "missing"}} {
		command, home := workflowCommand(t, args...)
		out, err := command.CombinedOutput()
		failure, ok := err.(*exec.ExitError)
		if !ok || failure.ExitCode() != 2 {
			t.Fatalf("args=%v code=%v out=%s", args, err, out)
		}
		entries, _ := os.ReadDir(home)
		if len(entries) != 0 {
			t.Fatalf("invalid doctor created local state: %v", entries)
		}
	}
}

func TestDoctorClosedTargetIsFailureNotHealthy(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer server.Close()
	command, _ := workflowCommand(t, "doctor", strconv.Itoa(port), "--server", strings.TrimPrefix(server.URL, "http://"), "--output", "json")
	out, err := command.CombinedOutput()
	if err == nil || !bytes.Contains(out, []byte(`"tcp_unavailable"`)) || !bytes.Contains(out, []byte(`"server_http_status"`)) {
		t.Fatalf("false healthy: %v %s", err, out)
	}
}
