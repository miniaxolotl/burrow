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
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestWorkflowRoutesOneLegacyWireTunnelToTwoLocalServices(t *testing.T) {
	marker := func(name string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "%s %s %s", name, r.Host, r.URL.RequestURI())
		}))
	}
	root, api := marker("root"), marker("api")
	defer root.Close()
	defer api.Close()
	port := func(s *httptest.Server) int {
		_, p, _ := net.SplitHostPort(strings.TrimPrefix(s.URL, "http://"))
		n, _ := strconv.Atoi(p)
		return n
	}
	config := filepath.Join(t.TempDir(), "routes.json")
	if err := os.WriteFile(config, []byte(fmt.Sprintf(`{"version":1,"routes":[{"path":"/api","port":%d,"strip_prefix":true},{"path":"/","port":%d}]}`, port(api), port(root))), 0600); err != nil {
		t.Fatal(err)
	}
	server, opened := workflowTunnelPeer(t, "")
	command, _ := workflowCommand(t, "tunnel", "--config", config, "--server", server, "--output", "jsonl")
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
		var e workflowEnvelope
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatal(err)
		}
		if e.Type == "connected" {
			if !bytes.Contains(e.Data, []byte(`"routes"`)) {
				t.Fatal("missing route summary")
			}
			connected = true
			break
		}
	}
	if !connected {
		t.Fatal("routed command failed before connection")
	}
	session := <-opened
	for _, s := range []struct {
		path, want string
		status     int
	}{{"/", "root public.test /", 200}, {"/api/a%20b?q=1", "api public.test /a%20b?q=1", 200}, {"/apiculture", "root public.test /apiculture", 200}, {"/api/%2e%2e/admin", "", 400}} {
		stream, err := session.Open()
		if err != nil {
			t.Fatal(err)
		}
		stream.SetDeadline(time.Now().Add(time.Second))
		fmt.Fprintf(stream, "GET %s HTTP/1.1\r\nHost: public.test\r\nConnection: close\r\n\r\n", s.path)
		resp, err := http.ReadResponse(bufio.NewReader(stream), nil)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		stream.Close()
		if err != nil || resp.StatusCode != s.status || (s.want != "" && string(body) != s.want) {
			t.Fatalf("%s => %q %d %v", s.path, body, resp.StatusCode, err)
		}
	}
	command.Process.Signal(syscall.SIGTERM)
	for scanner.Scan() {
	}
	if err := command.Wait(); err != nil || stderr.Len() != 0 {
		t.Fatalf("routed stop: %v %s", err, &stderr)
	}
}
