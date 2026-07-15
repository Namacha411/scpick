package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	paneStyle       = lipgloss.NewStyle().Padding(0, 1).Width(40).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	activePaneStyle = paneStyle.BorderForeground(lipgloss.Color("205"))
	statusStyle     = lipgloss.NewStyle().Faint(true)
	errStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	cursorStyle     = lipgloss.NewStyle().Reverse(true)
)

func (m model) viewBrowse() string {
	localHeader := fmt.Sprintf("Local: %s", m.local.path)
	remoteHeader := "Remote: (not connected — press C to connect)"
	if m.remoteClient != nil {
		remoteHeader = fmt.Sprintf("Remote: %s@%s:%s", m.pendingHost.User, m.pendingHost.Name, m.remote.path)
	}

	left := renderPaneFrame(localHeader, &m.local, m.focus == 0)
	right := renderPaneFrame(remoteHeader, &m.remote, m.focus == 1)
	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	hint := "j/k: move  h/l: dir  Tab: switch  y/p: yank/paste  Space/v: select  /: filter  C: connect  q: quit"
	switch m.mode {
	case ModeVisual:
		hint = "VISUAL  j/k: extend  y: yank  esc: cancel  v: done"
	case ModeFilter:
		hint = fmt.Sprintf("filter: %s   enter: keep  esc: clear", m.textInput.View())
	}
	return panes + "\n" + m.renderStatusLine(hint)
}

func renderPaneFrame(header string, pane *paneState, active bool) string {
	style := paneStyle
	if active {
		style = activePaneStyle
	}

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	for pos, i := range pane.visibleIndices() {
		e := pane.entries[i]
		name := e.Name
		if e.IsDir && !e.IsParent {
			name += "/"
		}
		if pane.isSelected(i) {
			name = "* " + name
		} else {
			name = "  " + name
		}
		b.WriteString(renderListRow(name, pos == pane.cursor))
		b.WriteString("\n")
	}
	return style.Render(strings.TrimRight(b.String(), "\n"))
}

func (m model) viewHostSelect() string {
	var b strings.Builder
	b.WriteString("Select a host (from ~/.ssh/config):\n\n")
	for i, h := range m.sshHosts {
		line := fmt.Sprintf("%s (%s@%s:%d)", h.Name, h.User, h.Hostname, h.Port)
		b.WriteString(renderListRow(line, i == m.hostCursor))
		b.WriteString("\n")
	}
	b.WriteString(renderListRow(manualEntryLabel, m.hostCursor == len(m.sshHosts)))
	b.WriteString("\n\n")
	b.WriteString(m.renderStatusLine("j/k: move  enter: select  esc: cancel"))
	return b.String()
}

func renderListRow(label string, selected bool) string {
	if selected {
		return cursorStyle.Render("> " + label)
	}
	return "  " + label
}

func (m model) viewManualHost() string {
	var prompt string
	switch m.manualHost.step {
	case stepHostname:
		prompt = "Hostname:"
	case stepUser:
		prompt = "User:"
	default:
		prompt = "Port [22]:"
	}
	return fmt.Sprintf("%s\n%s\n\n%s", prompt, m.textInput.View(), m.renderStatusLine("enter: next  esc: back"))
}

func (m model) viewConnecting() string {
	return fmt.Sprintf("Connecting to %s...\n\n%s", m.pendingHost.Name, m.renderStatusLine(""))
}

func (m model) viewPasswordPrompt() string {
	return fmt.Sprintf("%s\n%s\n\n%s", m.passwordPrompt, m.textInput.View(), m.renderStatusLine("enter: submit  esc: cancel"))
}

func (m model) viewHostKeyConfirm() string {
	return fmt.Sprintf(
		"The authenticity of host %q can't be established.\nKey fingerprint: %s\nTrust this host? [y/N]\n\n%s",
		m.hostKeyHostname, m.hostKeyFingerprint, m.renderStatusLine(""),
	)
}

func (m model) renderStatusLine(hint string) string {
	line := hint
	if m.errMsg != "" {
		if line != "" {
			line += "  "
		}
		line += errStyle.Render("error: " + m.errMsg)
	} else if m.status != "" {
		if line != "" {
			line += "  "
		}
		line += statusStyle.Render(m.status)
	}
	return statusStyle.Render(line)
}
