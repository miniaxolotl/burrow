package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"burrow/burrowctl/internal/capture"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type captureView struct {
	tunnel         string
	rows           []capture.Summary
	cursor, offset int
	loading, busy  bool
	observed       time.Time
	refreshErr     bool
	feedback       string
	preview        capture.Preview
	job            capture.Job
	retryParent    string
}
type captureRowsMsg struct {
	tunnel   string
	rows     []capture.Summary
	job      capture.Job
	err      bool
	observed time.Time
}
type capturePreviewMsg struct {
	preview capture.Preview
	err     bool
}
type captureQueuedMsg struct {
	job capture.Job
	err bool
}
type captureResultMsg struct {
	job capture.Job
	err bool
}
type captureErasedMsg struct{ err bool }

func (m model) loadCaptures() tea.Cmd {
	store, tunnel, jobID := m.client.CaptureStore(), m.capture.tunnel, m.capture.job.ID
	return func() tea.Msg {
		if store == nil {
			return captureRowsMsg{tunnel: tunnel, err: true}
		}
		rows, err := store.List(tunnel)
		var job capture.Job
		if err == nil && jobID != "" {
			job, err = store.Status(jobID)
		}
		return captureRowsMsg{tunnel, rows, job, err != nil, time.Now().UTC()}
	}
}
func (m model) applyCaptureRows(msg captureRowsMsg) model {
	if msg.tunnel != m.capture.tunnel {
		return m
	}
	m.capture.loading = false
	if msg.err {
		m.capture.refreshErr = true
		return m
	}
	selected := ""
	if m.capture.cursor < len(m.capture.rows) {
		selected = m.capture.rows[m.capture.cursor].ID
	}
	oldCursor := m.capture.cursor
	m.capture.rows = msg.rows
	m.capture.observed = msg.observed
	m.capture.refreshErr = false
	if msg.job.ID != "" && msg.job.ID == m.capture.job.ID {
		m.capture.job = msg.job
	}
	if selected != "" {
		for i, row := range msg.rows {
			if row.ID == selected {
				m.capture.cursor = i
				break
			}
		}
	}
	m.capture.cursor = max(0, min(m.capture.cursor, len(msg.rows)-1))
	m.capture.offset = max(0, m.capture.offset+m.capture.cursor-oldCursor)
	return m
}
func (m model) handleCaptureKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "q" || key == "ctrl+c" {
		return m, tea.Quit
	}
	if key == "esc" {
		if m.state == stateRequests {
			m.state = stateList
		} else {
			m.state = stateRequests
		}
		return m, nil
	}
	if m.state == stateReplayConfirm {
		if key == "n" {
			m.state = stateRequests
			return m, nil
		}
		if key == "y" && !m.capture.busy {
			m.capture.busy = true
			m.capture.feedback = "Committing approval (not yet accepted)"
			preview := m.capture.preview
			parent := m.capture.retryParent
			store := m.client.CaptureStore()
			return m, func() tea.Msg {
				job, err := store.Queue(preview.ID, preview.TunnelID, preview.ConfigDigest, parent, true)
				return captureQueuedMsg{job, err != nil}
			}
		}
		return m, nil
	}
	if m.state == stateEraseConfirm {
		if key == "n" {
			m.state = stateRequests
			return m, nil
		}
		if key == "y" && !m.capture.busy {
			m.capture.busy = true
			id, tunnel, store := m.capture.preview.ID, m.capture.tunnel, m.client.CaptureStore()
			return m, func() tea.Msg { return captureErasedMsg{store.Delete(id, tunnel) != nil} }
		}
		return m, nil
	}
	switch key {
	case "up", "k":
		m.capture.cursor = max(0, m.capture.cursor-1)
	case "down", "j":
		m.capture.cursor = min(max(0, len(m.capture.rows)-1), m.capture.cursor+1)
	case "r", "d":
		if m.capture.busy || len(m.capture.rows) == 0 {
			return m, nil
		}
		id := m.capture.rows[m.capture.cursor].ID
		if key == "d" {
			m.capture.preview = capture.Preview{ID: id}
			m.state = stateEraseConfirm
			return m, nil
		}
		m.capture.busy = true
		m.capture.feedback = "Reading scoped replay preview"
		tunnel, store := m.capture.tunnel, m.client.CaptureStore()
		return m, func() tea.Msg {
			preview, err := store.Preview(id, tunnel)
			return capturePreviewMsg{preview, err != nil}
		}
	}
	count := max(1, m.height-12)
	if m.capture.cursor < m.capture.offset {
		m.capture.offset = m.capture.cursor
	}
	if m.capture.cursor >= m.capture.offset+count {
		m.capture.offset = m.capture.cursor - count + 1
	}
	return m, nil
}
func jobFeedback(job capture.Job, failed bool) string {
	status := "unknown"
	if job.HTTPStatus != nil {
		status = strconv.Itoa(*job.HTTPStatus)
	}
	text := fmt.Sprintf("%s · %s · HTTP %s · %s", job.ID, job.State, status, job.Code)
	if failed && job.State == "" {
		return "Replay worker unavailable; inspect the queued job before retrying"
	}
	return text
}
func (m model) viewCaptures() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.banner(m.server))
	width := max(1, m.width-4)
	line := func(text string) { b.WriteString("  " + lipgloss.NewStyle().MaxWidth(width).Render(text) + "\n") }
	if m.state == stateReplayConfirm {
		p := m.capture.preview
		line("Approve one local replay? Every method, including GET, may duplicate effects.")
		line("")
		line(p.Method + " " + p.Path)
		line(fmt.Sprintf("loopback:%d · age %s · redacted=%t", p.Port, time.Since(p.CreatedAt).Round(time.Second), p.Redacted))
		line("Configuration: " + p.ConfigDigest)
		line("Verify the application currently using this port. Secrets/signatures are not restored.")
		if m.capture.retryParent != "" {
			line("Linked resend of uncertain/recovering job " + m.capture.retryParent)
		}
		line("")
		line(renderHelp([][2]string{{"y", "approve once"}, {"n/esc", "cancel"}}))
	} else if m.state == stateEraseConfirm {
		line("Erase this local capture?")
		line(m.capture.preview.ID)
		line("Unsent work is fenced. Already-dispatched effects cannot be undone.")
		line("")
		line(renderHelp([][2]string{{"y", "erase"}, {"n/esc", "cancel"}}))
	} else {
		line("Encrypted local captures · " + m.capture.tunnel)
		fresh := "not observed yet"
		if !m.capture.observed.IsZero() {
			fresh = "observed " + m.capture.observed.Format(time.RFC3339)
		}
		if m.capture.refreshErr {
			fresh = "refresh failed · " + fresh
		}
		line(fresh)
		line("")
		count := max(1, m.height-12)
		start := min(m.capture.offset, max(0, len(m.capture.rows)-1))
		end := min(len(m.capture.rows), start+count)
		for i := start; i < end; i++ {
			row := m.capture.rows[i]
			mark := "  "
			if i == m.capture.cursor {
				mark = "› "
			}
			line(fmt.Sprintf("%s%s  %s %s  %d B  redacted=%t", mark, row.ID, row.Method, row.Route, row.SizeBytes, row.Redacted))
		}
		for i := end - start; i < count; i++ {
			if len(m.capture.rows) == 0 && i == 0 {
				line("No complete replayable captures in this scope")
			} else {
				line("")
			}
		}
		line(fmt.Sprintf("%d records · newest first · %d–%d", len(m.capture.rows), min(start+1, len(m.capture.rows)), end))
		if m.capture.job.ID != "" {
			line(jobFeedback(m.capture.job, false))
		} else {
			line("")
		}
		line(m.capture.feedback)
		line(renderHelp([][2]string{{"↑/↓", "select"}, {"r", "replay"}, {"d", "erase"}, {"esc", "back"}, {"q", "quit"}}))
	}
	return lipgloss.NewStyle().Background(clrBg).Width(m.width).Height(m.height).Render(b.String())
}
