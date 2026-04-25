package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"burrow/burrowctl/internal"
	"burrow/protocol"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	clrBg       = lipgloss.Color("#1a1f1a")
	clrPurple   = lipgloss.Color("#b8c9a8")
	clrGreen    = lipgloss.Color("#a8d5a2")
	clrDimGray  = lipgloss.Color("#c8c8c0")
	clrFaint    = lipgloss.Color("#8a9a80")
	clrRed      = lipgloss.Color("#e8a598")
	clrOrange   = lipgloss.Color("#e8c98a")
	clrSelected = lipgloss.Color("#4a6b4a")
	clrWhite    = lipgloss.Color("#e8e8e0")

	clrStatus2xx = lipgloss.Color("#a8d5a2")
	clrStatus3xx = lipgloss.Color("#c8c8a0")
	clrStatus4xx = lipgloss.Color("#e8c98a")
	clrStatus5xx = lipgloss.Color("#e8a598")

	bg = lipgloss.NewStyle().Background(clrBg)

	titleStyle  = bg.Bold(true).Foreground(clrPurple)
	serverStyle = bg.Foreground(clrDimGray)
	divStyle    = bg.Foreground(clrFaint)

	colHeaderStyle = bg.Bold(true).Foreground(clrDimGray)
	rowIDStyle     = bg.Foreground(clrWhite)
	rowPortStyle   = bg.Foreground(clrDimGray)
	activeStyle    = bg.Foreground(clrGreen)
	reconStyle     = bg.Foreground(clrOrange)

	selRowStyle = lipgloss.NewStyle().Foreground(clrWhite).Background(clrSelected).Bold(true)

	urlStyle    = bg.Foreground(clrGreen).Underline(true)
	statusStyle = bg.Foreground(clrDimGray)
	errStyle    = bg.Foreground(clrRed)
	okStyle     = bg.Foreground(clrGreen)

	helpKeyStyle  = bg.Foreground(clrDimGray)
	helpDescStyle = bg.Foreground(clrFaint)

	overlayBg = lipgloss.NewStyle().Background(clrBg).Foreground(clrWhite).Padding(1, 2)
)

// ── state ─────────────────────────────────────────────────────────────────────

type viewState int

const (
	stateList     viewState = iota
	stateInput              // entering a port number
	stateCreating           // tunnel creation in flight
	stateLogs               // viewing request logs
	stateConfirm            // confirmation prompt (close tunnel)
	stateHelp               // help overlay
)

type sortMode int

const (
	sortPort sortMode = iota
	sortLatency
	sortReconnects
)

// ── messages ──────────────────────────────────────────────────────────────────

type tickMsg struct{}
type tunnelCreatedMsg struct {
	port uint16
	url  string
}
type errMsg struct{ err error }

// ── model ─────────────────────────────────────────────────────────────────────

type model struct {
	client       *internal.Client
	tunnels      []*protocol.TunnelInfo
	cursor       int
	state        viewState
	input        textinput.Model
	spinner      spinner.Model
	creatingPort uint16
	status       string
	statusIsErr  bool
	server       string
	width        int
	height       int
	logs         []*protocol.TunnelLog
	logCursor    int
	logsErr      string
	logAutoFollow bool
	sortMode     sortMode
	confirmTarget *protocol.TunnelInfo
}

func newModel(client *internal.Client, server string) model {
	ti := textinput.New()
	ti.Placeholder = "8080"
	ti.CharLimit = 5
	ti.Width = 8
	ti.TextStyle = bg.Foreground(clrWhite)
	ti.PlaceholderStyle = bg.Foreground(clrDimGray)
	ti.PromptStyle = bg.Foreground(clrPurple)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = bg.Foreground(clrPurple)

	return model{
		client:  client,
		server:  server,
		input:   ti,
		spinner: sp,
	}
}

// ── lifecycle ─────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	return doTick()
}

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

// ── update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tickMsg:
		selectedID := ""
		if len(m.tunnels) > 0 && m.cursor < len(m.tunnels) {
			selectedID = m.tunnels[m.cursor].TunnelID
		}
		m.tunnels = m.client.ListTunnels()
		m.sortTunnels()
		if len(m.tunnels) == 0 {
			m.cursor = 0
		} else if selectedID != "" {
			found := false
			for i, t := range m.tunnels {
				if t.TunnelID == selectedID {
					m.cursor = i
					found = true
					break
				}
			}
			if !found {
				m.cursor = max(0, len(m.tunnels)-1)
			}
		} else {
			m.cursor = max(0, min(m.cursor, len(m.tunnels)-1))
		}
		if m.state == stateLogs && len(m.tunnels) > 0 && m.cursor < len(m.tunnels) {
			t := m.tunnels[m.cursor]
			wasAtBottom := m.logCursor >= len(m.logs)-1
			logs, err := m.client.GetLogs(t.TunnelID)
			if err != nil {
				m.logsErr = err.Error()
			} else {
				m.logs = logs
				m.logsErr = ""
				if m.logAutoFollow || wasAtBottom {
					m.logCursor = max(0, len(m.logs)-1)
				} else if m.logCursor >= len(m.logs) {
					m.logCursor = max(0, len(m.logs)-1)
				}
			}
		}
		return m, doTick()

	case spinner.TickMsg:
		if m.state == stateCreating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tunnelCreatedMsg:
		m.state = stateList
		m.status = "created → " + msg.url
		m.statusIsErr = false
		m.tunnels = m.client.ListTunnels()

	case errMsg:
		m.state = stateList
		m.status = msg.err.Error()
		m.statusIsErr = true

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateInput:
		return m.handleInputKey(msg)
	case stateCreating:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	case stateLogs:
		return m.handleLogsKey(msg)
	case stateConfirm:
		return m.handleConfirmKey(msg)
	case stateHelp:
		if msg.String() == "?" || msg.String() == "esc" || msg.String() == "q" {
			m.state = stateList
		}
		return m, nil
	default:
		return m.handleListKey(msg)
	}
}

func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.tunnels)-1 {
			m.cursor++
		}

	case "n":
		m.state = stateInput
		m.input.SetValue("")
		return m, m.input.Focus()

	case "c":
		if len(m.tunnels) > 0 {
			t := m.tunnels[m.cursor]
			if err := clipboard.WriteAll(t.URL); err != nil {
				m.status = "clipboard failed: " + err.Error()
				m.statusIsErr = true
			} else {
				m.status = "copied " + t.URL
				m.statusIsErr = false
			}
		}

	case "d":
		if len(m.tunnels) > 0 {
			m.confirmTarget = m.tunnels[m.cursor]
			m.state = stateConfirm
		}

	case "l":
		if len(m.tunnels) > 0 {
			m.state = stateLogs
			m.logCursor = 0
			m.logs = nil
			m.logsErr = ""
		}

	case "o":
		if len(m.tunnels) > 0 {
			t := m.tunnels[m.cursor]
			if err := openURL(t.URL); err != nil {
				m.status = "open failed: " + err.Error()
				m.statusIsErr = true
			} else {
				m.status = "opened " + t.URL
				m.statusIsErr = false
			}
		}

	case "s":
		m.sortMode = (m.sortMode + 1) % 3
		m.sortTunnels()
		m.status = "sorted by " + [...]string{"port", "latency", "reconnects"}[m.sortMode]
		m.statusIsErr = false

	case "?":
		m.state = stateHelp
	}

	return m, nil
}

func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if m.confirmTarget != nil {
			m.client.CloseTunnel(m.confirmTarget.Port)
			m.status = fmt.Sprintf("closed :%d", m.confirmTarget.Port)
			m.statusIsErr = false
			m.tunnels = m.client.ListTunnels()
			m.sortTunnels()
			if m.cursor >= len(m.tunnels) {
				m.cursor = max(0, len(m.tunnels)-1)
			}
		}
		m.confirmTarget = nil
		m.state = stateList

	case "n", "esc":
		m.confirmTarget = nil
		m.state = stateList
	}
	return m, nil
}

func (m model) handleLogsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "l":
		m.state = stateList
		m.logs = nil
	case "up", "k":
		if m.logCursor > 0 {
			m.logCursor--
		}
	case "down", "j":
		if m.logCursor < len(m.logs)-1 {
			m.logCursor++
		}
	case "f":
		m.logAutoFollow = !m.logAutoFollow
		if m.logAutoFollow {
			m.logCursor = max(0, len(m.logs)-1)
		}
	case "r":
		if len(m.tunnels) > 0 && m.cursor < len(m.tunnels) {
			t := m.tunnels[m.cursor]
			logs, err := m.client.GetLogs(t.TunnelID)
			if err != nil {
				m.logsErr = err.Error()
			} else {
				m.logs = logs
				m.logsErr = ""
				if m.logAutoFollow {
					m.logCursor = max(0, len(m.logs)-1)
				}
			}
		}
	}
	return m, nil
}

func (m model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		raw := strings.TrimSpace(m.input.Value())
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			m.state = stateList
			m.status = fmt.Sprintf("invalid port: %q", raw)
			m.statusIsErr = true
			m.input.Blur()
			return m, nil
		}
		m.state = stateCreating
		m.creatingPort = uint16(port)
		m.input.Blur()
		return m, tea.Batch(m.spinner.Tick, m.openTunnel(uint16(port)))

	case "esc":
		m.state = stateList
		m.input.Blur()
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) openTunnel(port uint16) tea.Cmd {
	return func() tea.Msg {
		url, err := m.client.CreateTunnel(context.Background(), port)
		if err != nil {
			return errMsg{err}
		}
		return tunnelCreatedMsg{port: port, url: url}
	}
}

func (m *model) sortTunnels() {
	sort.SliceStable(m.tunnels, func(i, j int) bool {
		a, b := m.tunnels[i], m.tunnels[j]
		switch m.sortMode {
		case sortLatency:
			ai, _ := time.ParseDuration(a.Latency)
			bi, _ := time.ParseDuration(b.Latency)
			return ai < bi
		case sortReconnects:
			return a.Reconnects < b.Reconnects
		default:
			return a.Port < b.Port
		}
	})
}

// ── view ──────────────────────────────────────────────────────────────────────

const (
	colIDWidth     = 34
	colPortWidth   = 6
	colLatWidth    = 10
	colReconWidth  = 12
)

func (m model) View() string {
	if m.width == 0 {
		return ""
	}

	if m.state == stateHelp {
		return m.viewHelp()
	}

	if m.state == stateLogs {
		return m.viewLogs()
	}

	var b strings.Builder

	ind := bg.Render("  ")  // 2-space indent with background
	sep := bg.Render("  ")  // 2-space column separator with background

	// header
	titleW  := lipgloss.Width(titleStyle.Render("BURROW"))
	serverW := lipgloss.Width(serverStyle.Render(m.server))
	dashW   := max(0, m.width-titleW-serverW-6)
	b.WriteString(bg.Render("\n"))
	b.WriteString(ind +
		titleStyle.Render("BURROW") + sep +
		divStyle.Render(strings.Repeat("─", dashW)) + sep +
		serverStyle.Render(m.server) + "\n\n")

	// table header
	tableW := min(m.width-4, colIDWidth+colPortWidth+colLatWidth+colReconWidth+20)
	sortLabel := [...]string{"port", "latency", "reconnects"}[m.sortMode]
	b.WriteString(ind + colHeaderStyle.Render(
		fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %s  [%s]",
			colIDWidth, "TUNNEL ID",
			colPortWidth, "PORT",
			colLatWidth, "LATENCY",
			colReconWidth, "RECONNECTS",
			"STATUS", sortLabel),
	) + "\n")
	b.WriteString(ind + divStyle.Render(strings.Repeat("─", tableW)) + "\n")

	// rows
	if len(m.tunnels) == 0 {
		b.WriteString(ind + statusStyle.Render("no active tunnels") + "\n")
	}
	for i, t := range m.tunnels {
		id     := truncate(t.TunnelID, colIDWidth)
		port   := strconv.Itoa(int(t.Port))
		lat    := truncate(t.Latency, colLatWidth)
		recons := strconv.Itoa(t.Reconnects)

		if i == m.cursor {
			statusText := "● active"
			if t.Reconnects > 0 {
				statusText = fmt.Sprintf("● active (%d recon)", t.Reconnects)
			}
			line := fmt.Sprintf("▶ %-*s  %-*s  %-*s  %-*s  %s",
				colIDWidth, id, colPortWidth, port, colLatWidth, lat, colReconWidth, recons, statusText)
			b.WriteString(selRowStyle.Render(line) + "\n")
		} else {
			statusText := "● active"
			statusSty := activeStyle
			if t.Reconnects > 0 {
				statusText = fmt.Sprintf("● active (%d)", t.Reconnects)
				statusSty = reconStyle
			}
			b.WriteString(ind +
				rowIDStyle.Render(fmt.Sprintf("%-*s", colIDWidth, id)) + sep +
				rowPortStyle.Render(fmt.Sprintf("%-*s", colPortWidth, port)) + sep +
				rowPortStyle.Render(fmt.Sprintf("%-*s", colLatWidth, lat)) + sep +
				rowPortStyle.Render(fmt.Sprintf("%-*s", colReconWidth, recons)) + sep +
				statusSty.Render(statusText) + "\n")
		}
	}
	b.WriteString(ind + divStyle.Render(strings.Repeat("─", tableW)) + "\n\n")

	// url detail for selected tunnel
	if len(m.tunnels) > 0 && m.cursor < len(m.tunnels) {
		b.WriteString(ind + urlStyle.Render(m.tunnels[m.cursor].URL) + "\n")
	} else {
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// status / input / spinner / confirmation — fixed height of 2 lines to prevent layout shifts
	switch m.state {
	case stateInput:
		b.WriteString(ind + helpKeyStyle.Render("port ›") + bg.Render(" ") + m.input.View() + "\n")
		b.WriteString(ind + renderHelp([][2]string{{"enter", "open"}, {"esc", "cancel"}}) + "\n")

	case stateCreating:
		b.WriteString(ind + m.spinner.View() + bg.Render(" ") +
			statusStyle.Render(fmt.Sprintf("opening :%d…", m.creatingPort)) + "\n")
		b.WriteString("\n")

	case stateConfirm:
		if m.confirmTarget != nil {
			b.WriteString(ind + errStyle.Render(fmt.Sprintf("close :%d? [y/n]", m.confirmTarget.Port)) + "\n")
		}
		b.WriteString("\n")

	default:
		if m.status != "" {
			if m.statusIsErr {
				b.WriteString(ind + errStyle.Render("✗ "+m.status) + "\n")
			} else {
				b.WriteString(ind + okStyle.Render("✓ "+m.status) + "\n")
			}
		} else {
			b.WriteString("\n")
		}
		b.WriteString(ind + renderHelp([][2]string{
			{"n", "new"}, {"c", "copy"}, {"o", "open"}, {"d", "close"}, {"l", "logs"}, {"s", "sort"}, {"?", "help"}, {"↑↓ jk", "select"}, {"q", "quit"},
		}) + "\n")
	}

	b.WriteString("\n")

	return lipgloss.NewStyle().
		Background(clrBg).
		Width(m.width).
		Height(m.height).
		Render(b.String())
}

func (m model) viewLogs() string {
	var b strings.Builder
	ind := bg.Render("  ")
	sep := bg.Render("  ")

	tunnelID := ""
	if len(m.tunnels) > 0 && m.cursor < len(m.tunnels) {
		tunnelID = m.tunnels[m.cursor].TunnelID
	}

	// header
	b.WriteString(bg.Render("\n"))
	logsTitle := " logs — " + tunnelID
	if m.logAutoFollow {
		logsTitle += " [follow]"
	}
	b.WriteString(ind +
		titleStyle.Render("BURROW") + sep +
		statusStyle.Render(logsTitle) + "\n\n")

	if m.logsErr != "" {
		b.WriteString(ind + errStyle.Render(m.logsErr) + "\n\n")
		b.WriteString(ind + renderHelp([][2]string{{"esc", "back"}, {"q", "quit"}}) + "\n\n")
		return lipgloss.NewStyle().Background(clrBg).Width(m.width).Height(m.height).Render(b.String())
	}

	if len(m.logs) == 0 {
		b.WriteString(ind + statusStyle.Render("no requests yet…") + "\n\n")
		b.WriteString(ind + renderHelp([][2]string{{"esc", "back"}, {"q", "quit"}}) + "\n\n")
		return lipgloss.NewStyle().Background(clrBg).Width(m.width).Height(m.height).Render(b.String())
	}

	// table header
	const colTimeW = 10
	const colMethodW = 6
	const colPathW = 30
	const colSizeW = 8
	const colDurW = 10
	logTableW := min(m.width-4, colTimeW+colMethodW+colPathW+colSizeW+colDurW+20)
	b.WriteString(ind + colHeaderStyle.Render(
		fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %s",
			colTimeW, "TIME",
			colMethodW, "METHOD",
			colPathW, "PATH",
			colSizeW, "SIZE",
			"DURATION"),
	) + "\n")
	b.WriteString(ind + divStyle.Render(strings.Repeat("─", logTableW)) + "\n")

	// show last N entries that fit, with cursor
	maxRows := max(1, m.height-10)
	start := max(0, len(m.logs)-maxRows)
	if m.logCursor < start {
		start = m.logCursor
	}
	end := min(len(m.logs), start+maxRows)

	for i := start; i < end; i++ {
		l := m.logs[i]
		ts := l.Timestamp.Format("15:04:05")
		method := truncate(l.Method, colMethodW)
		path := truncate(l.Path, colPathW)
		size := formatSize(l.Size)
		dur := truncate(l.Duration, colDurW)

		if i == m.logCursor {
			line := fmt.Sprintf("▶ %-*s  %-*s  %-*s  %-*s  %s",
				colTimeW, ts, colMethodW, method, colPathW, path, colSizeW, size, dur)
			b.WriteString(selRowStyle.Render(line) + "\n")
		} else {
			b.WriteString(ind +
				rowPortStyle.Render(fmt.Sprintf("%-*s", colTimeW, ts)) + sep +
				rowPortStyle.Render(fmt.Sprintf("%-*s", colMethodW, method)) + sep +
				rowIDStyle.Render(fmt.Sprintf("%-*s", colPathW, path)) + sep +
				rowPortStyle.Render(fmt.Sprintf("%-*s", colSizeW, size)) + sep +
				rowPortStyle.Render(dur) + "\n")
		}
	}

	b.WriteString("\n")
	autoFollowPairs := [][2]string{{"esc", "back"}, {"↑↓ jk", "scroll"}, {"r", "refresh"}, {"q", "quit"}}
	if m.logAutoFollow {
		autoFollowPairs = append([][2]string{{"f", "follow on"}}, autoFollowPairs...)
	} else {
		autoFollowPairs = append([][2]string{{"f", "follow off"}}, autoFollowPairs...)
	}
	b.WriteString(ind + renderHelp(autoFollowPairs) + "\n\n")

	return lipgloss.NewStyle().
		Background(clrBg).
		Width(m.width).
		Height(m.height).
		Render(b.String())
}

func (m model) viewHelp() string {
	var b strings.Builder
	ind := bg.Render("  ")

	b.WriteString(bg.Render("\n"))
	b.WriteString(ind + titleStyle.Render("KEYBOARD SHORTCUTS") + "\n\n")

	b.WriteString(ind + helpKeyStyle.Render("TUNNEL LIST") + "\n")
	b.WriteString(renderHelp([][2]string{
		{"n", "new tunnel"}, {"c", "copy URL"}, {"o", "open URL"},
		{"d", "close tunnel"}, {"l", "view logs"}, {"s", "cycle sort"},
		{"↑↓ jk", "navigate"}, {"?", "this help"}, {"q", "quit"},
	}) + "\n\n")

	b.WriteString(ind + helpKeyStyle.Render("LOG VIEW") + "\n")
	b.WriteString(renderHelp([][2]string{
		{"↑↓ jk", "scroll"}, {"f", "toggle auto-follow"},
		{"r", "refresh"}, {"esc/l", "back"}, {"q", "quit"},
	}) + "\n\n")

	b.WriteString(ind + helpKeyStyle.Render("GENERAL") + "\n")
	b.WriteString(renderHelp([][2]string{
		{"ctrl+c", "quit"}, {"esc", "cancel/back"},
	}) + "\n\n")

	b.WriteString(ind + helpDescStyle.Render("press any key to dismiss") + "\n\n")

	return lipgloss.NewStyle().
		Background(clrBg).
		Width(m.width).
		Height(m.height).
		Render(b.String())
}

func formatSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(n)/(1024*1024))
}

func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func renderHelp(pairs [][2]string) string {
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = helpKeyStyle.Render(p[0]) + bg.Render(" ") + helpDescStyle.Render(p[1])
	}
	return strings.Join(parts, bg.Render("  ·  "))
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n-1]) + "…"
}

// RunWithClient starts the TUI using an already-initialized client.
// Called by tunnel create after tunnels are established.
func RunWithClient(client *internal.Client, server string) error {
	m := newModel(client, server)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
