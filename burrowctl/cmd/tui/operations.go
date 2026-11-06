package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"burrow/burrowctl/internal"
	"burrow/burrowctl/internal/inspection"
	"burrow/protocol"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type operationsView struct {
	id, generation                    string
	local, loading, details, checking bool
	token                             uint64
	cancel                            context.CancelFunc
	all                               []*protocol.TunnelLog
	query                             inspection.Query
	input                             textinput.Model
	editing                           string
	feedback                          string
	fetched                           time.Time
	start                             int
	port                              uint16
	checked                           internal.HealthObservation
	back                              viewState
	window                            string
}
type logRowsMsg struct {
	id         string
	local      bool
	token      uint64
	rows       []*protocol.TunnelLog
	generation string
	warning    string
	err        error
}
type healthCheckedMsg struct {
	id          string
	token       uint64
	observation internal.HealthObservation
}

func (m *model) cancelOperations() {
	if m.ops.cancel != nil {
		m.ops.cancel()
	}
	m.ops.loading = false
	m.ops.checking = false
	m.ops.token++
}

func (m *model) selectOperations(id string) {
	if m.ops.id == id {
		return
	}
	m.cancelOperations()
	m.ops.id = id
	m.ops.all = nil
	m.ops.start = 0
	m.ops.generation = ""
	m.ops.feedback = ""
	m.ops.details = false
	m.ops.checked = internal.HealthObservation{}
	m.logs = nil
	m.logCursor = 0
}

func (m *model) selectHealthPort(port uint16) {
	if m.ops.port != port {
		m.cancelOperations()
		m.ops.checked = internal.HealthObservation{}
	}
	m.ops.port = port
}

func (m *model) fetchLogRows() tea.Cmd {
	if m.ops.loading {
		return nil
	}
	if m.ops.id == "" && len(m.tunnels) > 0 && m.cursor < len(m.tunnels) {
		m.ops.id = m.tunnels[m.cursor].TunnelID
	}
	if m.ops.id == "" {
		return nil
	}
	m.ops.loading = true
	m.ops.token++
	id, local, token := m.ops.id, m.ops.local, m.ops.token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	m.ops.cancel = cancel
	client := m.client
	return func() tea.Msg {
		defer cancel()
		msg := logRowsMsg{id: id, local: local, token: token}
		if local {
			snapshot := client.Inspection(id)
			msg.rows = snapshot.Rows
			msg.generation = snapshot.Window
			msg.warning = fmt.Sprintf("local metadata · %d evicted / %d dropped · resets each generation", snapshot.Evicted, snapshot.Dropped)
			if !snapshot.Enabled {
				msg.warning = "Local inspection unavailable; restart with --inspect (not capture)"
			}
		} else {
			msg.rows, msg.err = client.GetLogsContext(ctx, id)
			msg.warning = "server logs · bounded retained window · polling can miss evictions"
		}
		return msg
	}
}

func (m model) applyLogRows(msg logRowsMsg) model {
	if m.state != stateLogs || m.ops.id != msg.id || m.ops.local != msg.local || m.ops.token != msg.token {
		return m
	}
	m.ops.loading = false
	if msg.err != nil {
		m.logsErr = "Refresh failed; retained rows are stale"
		return m
	}
	if msg.local && m.client.Inspection(msg.id).Window != msg.generation {
		return m
	}
	selected := ""
	offset := m.logCursor - m.ops.start
	if m.logCursor >= 0 && m.logCursor < len(m.logs) {
		selected = m.logs[m.logCursor].ID
	}
	if m.ops.generation != "" && msg.generation != m.ops.generation {
		m.ops.feedback = "Session window reset; previous selection may be unavailable"
	}
	m.ops.generation = msg.generation
	m.ops.window = msg.warning
	m.ops.all = msg.rows
	m.ops.fetched = time.Now()
	m.logsErr = ""
	m.logs = m.ops.query.Apply(msg.rows)
	if m.logAutoFollow {
		m.logCursor = max(0, len(m.logs)-1)
	} else {
		found := false
		for i, row := range m.logs {
			if row.ID == selected && selected != "" {
				m.logCursor = i
				found = true
				break
			}
		}
		if !found {
			m.logCursor = min(m.logCursor, max(0, len(m.logs)-1))
			if selected != "" {
				m.ops.feedback = "Selected row left the retained/filter window; nearest row selected"
			}
		}
	}
	m.ops.start = max(0, m.logCursor-max(0, offset))
	m.ensureLogViewport()
	return m
}

func (m *model) refilter() {
	selected := ""
	if m.logCursor < len(m.logs) {
		selected = m.logs[m.logCursor].ID
	}
	m.logs = m.ops.query.Apply(m.ops.all)
	m.logCursor = min(m.logCursor, max(0, len(m.logs)-1))
	for i, row := range m.logs {
		if row.ID == selected {
			m.logCursor = i
			break
		}
	}
	m.ops.start = min(m.ops.start, m.logCursor)
	m.ensureLogViewport()
}

func (m *model) ensureLogViewport() {
	count := max(1, m.height-10)
	m.ops.start = min(m.ops.start, m.logCursor)
	if m.logCursor >= m.ops.start+count {
		m.ops.start = m.logCursor - count + 1
	}
	m.ops.start = max(0, m.ops.start)
}

func (m model) operationsLogsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		m.cancelOperations()
		return m, tea.Quit
	}
	if m.ops.editing != "" {
		switch key {
		case "esc":
			m.ops.editing = ""
			m.ops.input.Blur()
			return m, nil
		case "enter":
			q := m.ops.query
			value := m.ops.input.Value()
			switch m.ops.editing {
			case "search":
				q.Search = value
			case "method":
				q.Methods = nil
				if value != "" {
					q.Methods = strings.Split(value, ",")
				}
			case "status":
				q.Statuses = nil
				if value != "" {
					q.Statuses = strings.Split(value, ",")
				}
			case "path":
				q.Path = value
			case "route":
				q.Route = value
			case "duration":
				q.Minimum = nil
				if value != "" {
					d, err := time.ParseDuration(value)
					if err != nil {
						m.ops.feedback = "Use a nonnegative Go duration"
						return m, nil
					}
					q.Minimum = &d
				}
			}
			source := "server"
			if m.ops.local {
				source = "local"
			}
			if err := q.Validate(source); err != nil {
				m.ops.feedback = err.Error()
				return m, nil
			}
			m.ops.query = q
			m.ops.editing = ""
			m.ops.input.Blur()
			m.ops.feedback = "Filters applied to the full retained snapshot"
			m.refilter()
			return m, nil
		}
		var cmd tea.Cmd
		m.ops.input, cmd = m.ops.input.Update(msg)
		return m, cmd
	}
	if m.ops.details {
		if key == "esc" || key == "enter" {
			m.ops.details = false
		}
		return m, nil
	}
	switch key {
	case "q":
		m.cancelOperations()
		return m, tea.Quit
	case "esc", "l":
		m.cancelOperations()
		m.state = stateList
	case "r":
		return m, m.fetchLogRows()
	case "v":
		if m.ops.local && m.ops.query.Route != "" {
			m.ops.feedback = "Clear local route filter before switching to server logs"
			return m, nil
		}
		m.cancelOperations()
		m.ops.local = !m.ops.local
		m.ops.all = nil
		m.logs = nil
		m.logCursor = 0
		m.ops.start = 0
		m.ops.feedback = "Source changed; rows have separate opaque identities"
		return m, m.fetchLogRows()
	case "enter":
		if len(m.logs) > 0 {
			m.ops.details = true
		}
	case "up", "k":
		m.logAutoFollow = false
		m.logCursor = max(0, m.logCursor-1)
	case "down", "j":
		m.logAutoFollow = false
		m.logCursor = min(max(0, len(m.logs)-1), m.logCursor+1)
	case "g", "home":
		m.logAutoFollow = false
		m.logCursor = 0
	case "G", "end":
		m.logAutoFollow = false
		m.logCursor = max(0, len(m.logs)-1)
	case "pgup":
		m.logAutoFollow = false
		m.logCursor = max(0, m.logCursor-10)
	case "pgdown":
		m.logAutoFollow = false
		m.logCursor = min(max(0, len(m.logs)-1), m.logCursor+10)
	case "f":
		m.logAutoFollow = !m.logAutoFollow
		if m.logAutoFollow {
			m.ops.query.Sort = ""
			m.ops.query.Descending = false
			m.refilter()
			m.logCursor = max(0, len(m.logs)-1)
		}
	case "o":
		m.logAutoFollow = false
		switch m.ops.query.Sort {
		case "":
			m.ops.query.Sort = "duration"
		case "duration":
			m.ops.query.Sort = "status"
		default:
			m.ops.query.Sort = ""
			m.ops.query.Descending = false
		}
		m.refilter()
	case "D":
		if m.ops.query.Sort != "" {
			m.ops.query.Descending = !m.ops.query.Descending
			m.refilter()
		}
	case "/", "m", "s", "p", "a", "t":
		field, value := "search", m.ops.query.Search
		switch key {
		case "m":
			field, value = "method", strings.Join(m.ops.query.Methods, ",")
		case "s":
			field, value = "status", strings.Join(m.ops.query.Statuses, ",")
		case "p":
			field, value = "path", m.ops.query.Path
		case "a":
			field, value = "route", m.ops.query.Route
		case "t":
			field = "duration"
			value = ""
			if m.ops.query.Minimum != nil {
				value = m.ops.query.Minimum.String()
			}
		}
		m.ops.editing = field
		m.ops.input = textinput.New()
		m.ops.input.CharLimit = 256
		m.ops.input.Width = max(8, m.width-20)
		m.ops.input.SetValue(value)
		return m, m.ops.input.Focus()
	}
	m.ensureLogViewport()
	return m, nil
}

func (m model) viewOperationsLogs() string {
	width := max(1, m.width-4)
	var b strings.Builder
	line := func(text string) { b.WriteString("  " + statusStyle.Render(truncate(text, width)) + "\n") }
	source := "server logs"
	if m.ops.local {
		source = "local live metadata"
	}
	mode := "paused"
	if m.logAutoFollow {
		mode = "follow"
	}
	b.WriteString("  " + titleStyle.Render(truncate("BURROW · "+source+" · "+mode, width)) + "\n")
	line(m.ops.id)
	if m.ops.editing != "" {
		b.WriteString("  " + truncate(m.ops.editing+" › ", width) + m.ops.input.View() + "\n")
	} else {
		filters := []string{}
		if m.ops.query.Search != "" {
			filters = append(filters, "search="+m.ops.query.Search)
		}
		if len(m.ops.query.Methods) > 0 {
			filters = append(filters, "method="+strings.Join(m.ops.query.Methods, ","))
		}
		if len(m.ops.query.Statuses) > 0 {
			filters = append(filters, "status="+strings.Join(m.ops.query.Statuses, ","))
		}
		if m.ops.query.Path != "" {
			filters = append(filters, "path="+m.ops.query.Path)
		}
		if m.ops.query.Route != "" {
			filters = append(filters, "route="+m.ops.query.Route)
		}
		if m.ops.query.Minimum != nil {
			filters = append(filters, "duration≥"+m.ops.query.Minimum.String())
		}
		if len(filters) == 0 {
			filters = append(filters, "none")
		}
		order := m.ops.query.Sort
		if order == "" {
			order = "append"
		}
		line("Filters: " + strings.Join(filters, " · ") + " · sort: " + order)
	}
	fresh := "unknown"
	if !m.ops.fetched.IsZero() {
		fresh = "fresh " + m.ops.fetched.Format("15:04:05")
		if time.Since(m.ops.fetched) > 30*time.Second {
			fresh = "stale"
		}
	}
	if m.logsErr != "" {
		fresh = "stale/error"
	}
	line(fmt.Sprintf("%d matching / %d retained · %s · max 200 / 512 KiB", len(m.logs), len(m.ops.all), fresh))
	feedback := m.ops.feedback
	if feedback == "" {
		feedback = m.ops.window
	}
	if m.logsErr != "" {
		feedback = m.logsErr
	}
	line(feedback)
	line(strings.Repeat("─", width))
	if m.ops.details && m.logCursor < len(m.logs) {
		r := m.logs[m.logCursor]
		for _, text := range []string{"REQUEST DETAILS · " + source, "ID " + r.ID, "Generation " + r.Generation, r.Method + " " + r.Path, "Route " + r.Route, fmt.Sprintf("HTTP %s · origin %s", statusText(r.StatusCode), r.StatusOrigin), "Outcome " + r.Outcome + " · " + r.Reason, "Duration " + r.Duration, "Completed " + observedTime(r.CompletedAt), "No saved capture is implied; this row is not replayable."} {
			line(text)
		}
		if r.PathTruncated {
			line("Path truncated at the retained 2 KiB limit")
		}
		line("esc back · selection and filters preserved")
	} else {
		line(fmt.Sprintf("%-2s %-8s %-6s %-6s %-8s %s", "", "TIME", "METHOD", "STATUS", "DURATION", "PATH"))
		count := max(1, m.height-10)
		start := min(m.ops.start, m.logCursor)
		if m.logCursor >= start+count {
			start = m.logCursor - count + 1
		}
		if len(m.logs) == 0 {
			line("No matching requests in this retained source window")
		}
		for i := start; i < min(len(m.logs), start+count); i++ {
			r := m.logs[i]
			status := "?"
			if r.StatusCode != 0 {
				status = fmt.Sprint(r.StatusCode)
			}
			prefix := " "
			if i == m.logCursor {
				prefix = "▶"
			}
			text := fmt.Sprintf("%-2s %-8s %-6s %-6s %-8s %s", prefix, r.Timestamp.Format("15:04:05"), truncate(r.Method, 6), status, displayDuration(r.Duration), r.Path)
			if i == m.logCursor {
				b.WriteString("  " + selRowStyle.Width(width).Render(truncate(text, width)) + "\n")
			} else {
				line(text)
			}
		}
		if m.width < 80 {
			line("filters /m/s/p/a/t · enter detail · v src · o sort")
			line("↑↓ pause · f follow · r refresh · esc back · q quit")
		} else {
			line("/ m s p a t filters · enter details · v source · o sort")
			line("↑↓ pause · f follow · D reverse · r refresh · esc back · q quit")
		}
	}
	return lipgloss.NewStyle().Background(clrBg).Width(m.width).Height(m.height).Render(b.String())
}

func (m model) handleHealthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.cancelOperations()
		return m, tea.Quit
	case "esc", "h":
		m.cancelOperations()
		m.state = m.ops.back
	case "r":
		if !m.ops.checking {
			m.ops.checking = true
			m.ops.token++
			id, port, token, client := m.ops.id, m.ops.port, m.ops.token, m.client
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			m.ops.cancel = cancel
			return m, func() tea.Msg { defer cancel(); return healthCheckedMsg{id, token, client.RecheckTCP(ctx, id, port)} }
		}
	}
	return m, nil
}

func (m model) viewHealth() string {
	width := max(1, m.width-4)
	var b strings.Builder
	line := func(text string) { b.WriteString("  " + statusStyle.Render(truncate(text, width)) + "\n") }
	b.WriteString("  " + titleStyle.Render(truncate("BURROW · scoped health", width)) + "\n")
	line(m.ops.id)
	line("Registration/admission is separate from app health")
	for _, t := range m.tunnels {
		if t.TunnelID == m.ops.id {
			line("Transport: " + t.Status)
			admission := "unknown (lifecycle capability unavailable)"
			if t.LifecycleConfirmed {
				admission = t.Status
			}
			line("Admission: " + admission)
		}
	}
	server, local := m.client.Health(m.ops.id)
	line(fmt.Sprintf("Apex: %s · %s · %s", server.Outcome, server.Reason, server.Freshness))
	line("Observed " + observedTime(server.ObservedAt))
	if len(local) == 0 {
		line("Local TCP/application: unknown (no observed traffic)")
	}
	shown := 0
	for _, o := range local {
		if o.Port != m.ops.port {
			continue
		}
		if shown >= max(1, (m.height-11)/2) {
			line("Additional scoped observations outside this viewport")
			break
		}
		shown++
		line(fmt.Sprintf("%s :%d · %s · HTTP %s · %s", o.Source, o.Port, o.Outcome, statusText(o.Status), o.Freshness))
		line("Observed " + observedTime(o.ObservedAt))
	}
	if m.ops.checking {
		line("Explicit TCP recheck in progress (not app HTTP)")
	} else if m.ops.checked.Outcome != "" {
		line("Recheck: " + m.ops.checked.Outcome + " · " + m.ops.checked.Reason)
	} else {
		line("No active app probes run automatically")
	}
	line("TCP acceptance does not prove HTTP health.")
	line("r TCP recheck (visible to app) · esc back · q quit")
	return lipgloss.NewStyle().Background(clrBg).Width(m.width).Height(m.height).Render(b.String())
}

func statusText(code int) string {
	if code == 0 {
		return "unknown"
	}
	return fmt.Sprint(code)
}
func observedTime(at time.Time) string {
	if at.IsZero() {
		return "unknown"
	}
	return at.Format(time.RFC3339)
}

func displayDuration(value string) string {
	d, err := time.ParseDuration(value)
	if err != nil {
		return "?"
	}
	if d >= time.Second {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
}
