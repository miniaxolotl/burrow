package tui

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"burrow/burrowctl/internal"
	"burrow/protocol"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Cmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive tunnel manager",
	RunE:  run,
}

func init() {
	Cmd.Flags().IntSlice("port", []int{}, "Ports to open on start (repeatable)")
}

// ── styles ────────────────────────────────────────────────────────────────────

var (
	clrBg       = lipgloss.Color("#1a1f1a")
	clrPurple   = lipgloss.Color("#b8c9a8")
	clrGreen    = lipgloss.Color("#a8d5a2")
	clrDimGray  = lipgloss.Color("#c8c8c0")
	clrFaint    = lipgloss.Color("#8a9a80")
	clrRed      = lipgloss.Color("#e8a598")
	clrSelected = lipgloss.Color("#4a6b4a")
	clrWhite    = lipgloss.Color("#e8e8e0")

	bg = lipgloss.NewStyle().Background(clrBg)

	titleStyle  = bg.Bold(true).Foreground(clrPurple)
	serverStyle = bg.Foreground(clrDimGray)
	divStyle    = bg.Foreground(clrFaint)

	colHeaderStyle = bg.Bold(true).Foreground(clrDimGray)
	rowIDStyle     = bg.Foreground(clrWhite)
	rowPortStyle   = bg.Foreground(clrDimGray)
	activeStyle    = bg.Foreground(clrGreen)

	selRowStyle  = lipgloss.NewStyle().Foreground(clrWhite).Background(clrSelected).Bold(true)
	selPortStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#d4e4c8")).Background(clrSelected)
	selCursor    = lipgloss.NewStyle().Foreground(clrGreen).Background(clrSelected).Bold(true)

	urlStyle    = bg.Foreground(clrGreen).Underline(true)
	statusStyle = bg.Foreground(clrDimGray)
	errStyle    = bg.Foreground(clrRed)
	okStyle     = bg.Foreground(clrGreen)

	helpKeyStyle  = bg.Foreground(clrDimGray)
	helpDescStyle = bg.Foreground(clrFaint)
)

// ── state ─────────────────────────────────────────────────────────────────────

type viewState int

const (
	stateList     viewState = iota
	stateInput              // entering a port number
	stateCreating           // tunnel creation in flight
	stateLogs               // viewing request logs
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
		m.tunnels = m.client.ListTunnels()
		if m.cursor >= len(m.tunnels) {
			m.cursor = max(0, len(m.tunnels)-1)
		}
		if m.state == stateLogs && len(m.tunnels) > 0 && m.cursor < len(m.tunnels) {
			t := m.tunnels[m.cursor]
			logs, err := m.client.GetLogs(t.TunnelID)
			if err != nil {
				m.logsErr = err.Error()
			} else {
				m.logs = logs
				m.logsErr = ""
				if m.logCursor >= len(m.logs) {
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
	default:
		return m.handleListKey(msg)
	}
}

func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""

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
			t := m.tunnels[m.cursor]
			m.client.CloseTunnel(t.Port)
			m.status = fmt.Sprintf("closed :%d", t.Port)
			m.statusIsErr = false
			m.tunnels = m.client.ListTunnels()
			if m.cursor >= len(m.tunnels) {
				m.cursor = max(0, len(m.tunnels)-1)
			}
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
	b.WriteString(ind + colHeaderStyle.Render(
		fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %s",
			colIDWidth, "TUNNEL ID",
			colPortWidth, "PORT",
			colLatWidth, "LATENCY",
			colReconWidth, "RECONNECTS",
			"STATUS"),
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
			// Render entire row as one block to avoid selection-highlight gaps.
			line := fmt.Sprintf("▶ %-*s  %-*s  %-*s  %-*s  ● active",
				colIDWidth, id, colPortWidth, port, colLatWidth, lat, colReconWidth, recons)
			b.WriteString(selRowStyle.Render(line) + "\n")
		} else {
			b.WriteString(ind +
				rowIDStyle.Render(fmt.Sprintf("%-*s", colIDWidth, id)) + sep +
				rowPortStyle.Render(fmt.Sprintf("%-*s", colPortWidth, port)) + sep +
				rowPortStyle.Render(fmt.Sprintf("%-*s", colLatWidth, lat)) + sep +
				rowPortStyle.Render(fmt.Sprintf("%-*s", colReconWidth, recons)) + sep +
				activeStyle.Render("● active") + "\n")
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

	// status / input / spinner — fixed height of 2 lines to prevent layout shifts
	switch m.state {
	case stateInput:
		b.WriteString(ind + helpKeyStyle.Render("port ›") + bg.Render(" ") + m.input.View() + "\n")
		b.WriteString(ind + renderHelp([][2]string{{"enter", "open"}, {"esc", "cancel"}}) + "\n")

	case stateCreating:
		b.WriteString(ind + m.spinner.View() + bg.Render(" ") +
			statusStyle.Render(fmt.Sprintf("opening :%d…", m.creatingPort)) + "\n")
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
			{"n", "new"}, {"c", "copy"}, {"o", "open"}, {"d", "close"}, {"l", "logs"}, {"↑↓ jk", "select"}, {"q", "quit"},
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
	b.WriteString(ind +
		titleStyle.Render("BURROW") + sep +
		statusStyle.Render(" logs — "+tunnelID) + "\n\n")

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
	b.WriteString(ind + renderHelp([][2]string{{"esc", "back"}, {"↑↓ jk", "scroll"}, {"q", "quit"}}) + "\n\n")

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
		parts[i] = helpKeyStyle.Render(p[0]) + " " + helpDescStyle.Render(p[1])
	}
	return strings.Join(parts, helpDescStyle.Render("  ·  "))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// ── command ───────────────────────────────────────────────────────────────────

func autoTLS(server string) bool {
	if viper.GetBool("tls") {
		return true
	}
	host := server
	if h, _, err := net.SplitHostPort(server); err == nil {
		host = h
	}
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return false
	}
	return true
}

func resolveToken() string {
	if t := viper.GetString("token"); t != "" {
		return t
	}
	if home, err := os.UserHomeDir(); err == nil {
		if data, err := os.ReadFile(home + "/.burrow/token"); err == nil {
			if t := strings.TrimSpace(string(data)); t != "" {
				return t
			}
		}
	}
	if secret := viper.GetString("secret"); secret != "" {
		return protocol.GenerateToken(secret)
	}
	return ""
}

// RunWithClient starts the TUI using an already-initialized client.
// Called by tunnel create after tunnels are established.
func RunWithClient(client *internal.Client, server string) error {
	m := newModel(client, server)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func run(cmd *cobra.Command, _ []string) error {
	ports, _ := cmd.Flags().GetIntSlice("port")
	server := viper.GetString("server")
	domain := viper.GetString("domain")
	token  := resolveToken()

	client := internal.NewClient(server, token, domain, autoTLS(server))
	defer client.Close()

	ctx := context.Background()
	for _, p := range ports {
		if _, err := client.CreateTunnel(ctx, uint16(p)); err != nil {
			return fmt.Errorf("port %d: %w", p, err)
		}
	}

	m := newModel(client, server)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
