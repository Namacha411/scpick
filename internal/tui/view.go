package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	paneStyle         = lipgloss.NewStyle().Padding(0, 1).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	activePaneStyle   = paneStyle.BorderForeground(lipgloss.Color("205"))
	statusStyle       = lipgloss.NewStyle().Faint(true)
	errStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	cursorStyle       = lipgloss.NewStyle().Reverse(true)
	markedStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	cursorMarkedStyle = cursorStyle.Bold(true).Foreground(lipgloss.Color("214"))
)

// paneChromeWidth is how much of a pane's rendered width is border (1 each
// side) and padding (1 each side, from paneStyle's Padding(0, 1)) rather
// than content.
const paneChromeWidth = 4

// paneChromeHeight is how many lines outside the entry list itself a pane
// takes up: its header line, plus its top and bottom border.
const paneChromeHeight = 3

// outsideChromeHeight is how many lines outside the pane box itself the
// browse view takes up: the blank line before the status bar, and the
// status bar itself.
const outsideChromeHeight = 2

const minPaneWidth = 20

// paneWidth returns the content width available to each of the two
// side-by-side panes, filling the terminal's actual width.
func (m model) paneWidth() int {
	w := m.termWidth/2 - paneChromeWidth
	if w < minPaneWidth {
		w = minPaneWidth
	}
	return w
}

// paneContentHeight returns how many entry rows a pane can show, filling
// the terminal's actual height.
func (m model) paneContentHeight() int {
	h := m.termHeight - paneChromeHeight - outsideChromeHeight
	if h < 1 {
		h = 1
	}
	return h
}

func (m model) viewBrowse() string {
	localHeader := fmt.Sprintf("Local: %s", m.local.path)
	remoteHeader := "Remote: (not connected — press C to connect)"
	if m.remoteClient != nil {
		remoteHeader = fmt.Sprintf("Remote: %s@%s:%s", m.pendingHost.User, m.pendingHost.Name, m.remote.path)
	}

	width := m.paneWidth()
	height := m.paneContentHeight()
	left := renderPaneFrame(localHeader, &m.local, m.focus == 0, width, height, 0, m.yank)
	right := renderPaneFrame(remoteHeader, &m.remote, m.focus == 1, width, height, 1, m.yank)
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

func renderPaneFrame(header string, pane *paneState, active bool, width, height, focus int, yank yankBuffer) string {
	style := paneStyle
	if active {
		style = activePaneStyle
	}
	// lipgloss's word-wrap wraps to Width() minus the style's own left/right
	// Padding(0, 1) — i.e. Width(N) gives N-2 cells of actual text room, not
	// N. width here is already the pure text width (paneWidth already
	// subtracted padding out of paneChromeWidth), so add the 2 padding
	// cells back to make lipgloss wrap at exactly `width`, matching what
	// truncateName below was told to fit into.
	style = style.Width(width + 2)

	vis := pane.visibleIndices()
	start, end := visibleWindow(pane.cursor, len(vis), height)

	lines := make([]string, 0, height+1)
	lines = append(lines, truncateName(header, width))
	for pos := start; pos < end; pos++ {
		i := vis[pos]
		e := pane.entries[i]
		name := e.Name
		if e.IsDir && !e.IsParent {
			name += "/"
		}
		// An entry is highlighted either while marked with Space/v (pending
		// a "y") or, after "y", while it's sitting in the yank buffer
		// waiting for "p" — the yank buffer outlives the mark, which is
		// cleared the moment "y" is pressed.
		highlighted := pane.isSelected(i) || (!e.IsParent && yank.has(focus, joinSourcePath(focus, pane.path, e.Name)))
		if highlighted {
			name = "* " + name
		} else {
			name = "  " + name
		}
		lines = append(lines, renderListRow(truncateName(name, width-2), pos == pane.cursor, highlighted))
	}
	// Pad with blank rows so every pane is exactly `height` rows tall
	// regardless of how many entries it holds — otherwise the border
	// would hug short listings instead of filling the terminal.
	for len(lines) <= height {
		lines = append(lines, "")
	}
	return style.Render(strings.Join(lines, "\n"))
}

// visibleWindow returns the [start, end) slice of a total-length list of
// entries to display in a viewport of the given height, keeping cursor
// inside that window (centered when possible).
func visibleWindow(cursor, total, height int) (int, int) {
	if total <= height {
		return 0, total
	}
	start := cursor - height/2
	if start < 0 {
		start = 0
	}
	if start > total-height {
		start = total - height
	}
	return start, start + height
}

// truncateName shortens s to fit within width terminal cells, replacing the
// tail with an ellipsis if it doesn't. It measures width the same way
// lipgloss's own word-wrap does (ansi.StringWidth), accounting for wide
// East-Asian characters and emoji, so a pane's rendered lines never exceed
// the width lipgloss was asked to wrap to — a mismatch there previously let
// lipgloss silently wrap a too-wide line onto an extra terminal row,
// unbalancing the two panes' heights.
func truncateName(s string, width int) string {
	if width < 1 {
		return ""
	}
	return ansi.Truncate(s, width, "…")
}

func (m model) viewHostSelect() string {
	var b strings.Builder
	b.WriteString("Select a host (from ~/.ssh/config):\n\n")
	for i, h := range m.sshHosts {
		line := fmt.Sprintf("%s (%s@%s:%d)", h.Name, h.User, h.Hostname, h.Port)
		b.WriteString(renderListRow(line, i == m.hostCursor, false))
		b.WriteString("\n")
	}
	b.WriteString(renderListRow(manualEntryLabel, m.hostCursor == len(m.sshHosts), false))
	b.WriteString("\n\n")
	b.WriteString(m.renderStatusLine("j/k: move  enter: select  esc: cancel"))
	return b.String()
}

// renderListRow renders one row of a list, applying the reverse-video
// cursor style when isCursor, and bolding+coloring it when isMarked (an
// entry marked with Space/v, so it survives being scrolled past or having
// the cursor move off it).
func renderListRow(label string, isCursor, isMarked bool) string {
	switch {
	case isCursor && isMarked:
		return cursorMarkedStyle.Render("> " + label)
	case isCursor:
		return cursorStyle.Render("> " + label)
	case isMarked:
		return markedStyle.Render("  " + label)
	default:
		return "  " + label
	}
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
