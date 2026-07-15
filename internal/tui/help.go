package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateHelp returns to Browse mode on any key press.
func (m model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}
	m.mode = ModeBrowse
	return m, nil
}

func (m model) viewHelp() string {
	var b strings.Builder
	for i, group := range helpGroups {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(group.Title)
		b.WriteString("\n")
		for _, kb := range group.Bindings {
			fmt.Fprintf(&b, "  %-10s %s\n", kb.Keys, kb.Desc)
		}
	}
	b.WriteString("\n")
	b.WriteString(m.renderStatusLine("press any key to go back"))
	return b.String()
}
