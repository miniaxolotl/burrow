package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func (m model) viewRoutes() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.banner(m.server))
	b.WriteString("  Explicit startup routes · one shared public origin\n\n")
	routes := m.client.Routes()
	count := max(1, m.height-9)
	start := min(m.routeCursor, max(0, len(routes)-count))
	for i := start; i < min(len(routes), start+count); i++ {
		r := routes[i]
		host := r.HostHeader
		if host == "" {
			host = "preserve public Host"
		}
		line := fmt.Sprintf("%s → loopback:%d · strip=%t · %s", r.Path, r.Port, r.StripPrefix, host)
		b.WriteString("  " + truncate(line, max(1, m.width-4)) + "\n")
	}
	if len(routes) == 0 {
		b.WriteString("  Single-port raw forwarding; no route table\n")
	}
	b.WriteString("\n  " + renderHelp([][2]string{{"↑/↓", "scroll"}, {"esc", "back"}, {"q", "quit"}}) + "\n")
	return lipgloss.NewStyle().Background(clrBg).Width(m.width).Height(m.height).Render(b.String())
}
