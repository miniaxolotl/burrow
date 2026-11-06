package tui

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"burrow/burrowctl/internal"
	"burrow/protocol"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestLogRefreshDoesNotPerformNetworkInReducer(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-release; w.Write([]byte("[]")) }))
	defer server.Close()
	defer close(release)
	client := internal.NewClient(strings.TrimPrefix(server.URL, "http://"), "synthetic-owner", "example.test", false)
	defer client.Close()
	m := newModel(client, "example.test")
	m.state = stateLogs
	m.tunnels = []*protocol.TunnelInfo{{TunnelID: "a"}}
	done := make(chan struct{})
	go func() { m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}); close(done) }()
	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Update blocked on network")
	}
}

func TestLogRefreshPreservesIdentityDraftOffsetAndStaleRows(t *testing.T) {
	m := model{state: stateLogs, height: 24, width: 80, logs: []*protocol.TunnelLog{{ID: "a", Path: "/a"}, {ID: "b", Path: "/b"}}, logCursor: 1, ops: operationsView{id: "selected", token: 5, start: 1}}
	m.ops.editing = "search"
	m.ops.input.SetValue("draft")
	m = m.applyLogRows(logRowsMsg{id: "previous", token: 5, rows: []*protocol.TunnelLog{{ID: "wrong"}}})
	if m.logs[1].ID != "b" {
		t.Fatal("old tunnel response replaced rows")
	}
	m = m.applyLogRows(logRowsMsg{id: "selected", token: 5, rows: []*protocol.TunnelLog{{ID: "new"}, {ID: "a", Path: "/a"}, {ID: "b", Path: "/b"}}})
	if m.logCursor != 2 || m.ops.start != 2 || m.ops.input.Value() != "draft" || m.ops.editing != "search" {
		t.Fatal("refresh moved identity/offset/draft")
	}
	m = m.applyLogRows(logRowsMsg{id: "selected", token: 5, err: errors.New("private")})
	if len(m.logs) != 3 || !strings.Contains(m.viewOperationsLogs(), "/b") || !strings.Contains(m.logsErr, "stale") {
		t.Fatal("failed refresh hid mounted rows")
	}
}

func TestInspectionLayoutsFitTerminalWidths(t *testing.T) {
	for _, size := range [][2]int{{60, 20}, {80, 24}, {110, 32}} {
		m := model{state: stateLogs, width: size[0], height: size[1], ops: operationsView{id: "synthetic"}}
		for i := range 40 {
			m.logs = append(m.logs, &protocol.TunnelLog{ID: fmt.Sprint(i), Method: "POST", Path: "/界/\x1b[31mhook", Duration: "250ms", StatusCode: 500})
		}
		for _, details := range []bool{false, true} {
			m.ops.details = details
			frame := ansi.Strip(m.viewOperationsLogs())
			lines := strings.Split(strings.TrimSuffix(frame, "\n"), "\n")
			if len(lines) > size[1] {
				t.Fatalf("%v overflow height: %d", size, len(lines))
			}
			for _, line := range lines {
				if ansi.StringWidth(line) > size[0] {
					t.Fatalf("%v overflow width: %q", size, line)
				}
			}
		}
	}
}

func TestManualNavigationPausesLogFollow(t *testing.T) {
	m := model{state: stateLogs, logs: []*protocol.TunnelLog{{ID: "a"}, {ID: "b"}}, logCursor: 1, logAutoFollow: true}
	next, _ := m.handleLogsKey(tea.KeyMsg{Type: tea.KeyUp})
	if next.(model).logAutoFollow {
		t.Fatal("manual navigation left follow enabled")
	}
}
