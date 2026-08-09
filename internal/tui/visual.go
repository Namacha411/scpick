package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// deactivateEntry unmarks entry i in pane's selected set and, unless it's
// the ".." entry, removes its path from yank too (a no-op if it wasn't
// there). Shared by toggleMark and applyVisualRange, the two places that
// turn an active (marked and/or yanked) entry back off.
func deactivateEntry(focus int, pane *paneState, selected map[int]bool, yank yankBuffer, i int) yankBuffer {
	delete(selected, i)
	entry := pane.entries[i]
	if entry.IsParent {
		return yank
	}
	return yank.without(focus, joinSourcePath(focus, pane.path, entry.Name))
}

// toggleMark toggles the mark on the entry under the cursor (available
// directly from Browse mode, independent of ModeVisual). If the entry is
// currently sitting in the yank buffer instead, Space un-yanks it there
// rather than adding a redundant mark, so a single item can always be
// removed from a pending transfer without replacing the whole buffer.
func (m model) toggleMark() (tea.Model, tea.Cmd) {
	pane := m.activePane()
	i, ok := pane.currentIndex()
	if !ok || pane.entries[i].IsParent {
		return m, nil
	}

	path := joinSourcePath(m.focus, pane.path, pane.entries[i].Name)
	if m.yank.has(m.focus, path) {
		m.yank = deactivateEntry(m.focus, pane, pane.selected, m.yank, i)
		m.status = fmt.Sprintf("un-yanked (%d left)", m.yank.count())
		return m, nil
	}

	if pane.selected == nil {
		pane.selected = make(map[int]bool)
	}
	if pane.selected[i] {
		delete(pane.selected, i)
	} else {
		pane.selected[i] = true
	}
	return m, nil
}

// enterVisual anchors a selection range at the current cursor position and
// switches to ModeVisual, remembering the pane's prior marks and yank buffer
// so Esc can restore them. Whether extending the range activates or
// deactivates entries is decided once here, from the anchor entry's current
// state (see applyVisualRange).
func (m model) enterVisual() (tea.Model, tea.Cmd) {
	pane := m.activePane()
	m.visualAnchor = pane.cursor
	m.visualSnapshot = cloneSelection(pane.selected)
	m.visualYankSnapshot = m.yank
	m.mode = ModeVisual
	m.applyVisualRange()
	return m, nil
}

// applyVisualRange applies Space's toggle to every visible entry between the
// visual anchor and the current cursor (inclusive), recomputed from the
// snapshot taken when ModeVisual was entered so shrinking the range restores
// entries outside it to their prior mark/yank state. The direction is fixed
// for the whole visual session: if the anchor entry was inactive (neither
// marked nor yanked) when "v" was pressed, the range activates (marks)
// entries; if the anchor was active, the range deactivates them (unmarks
// and un-yanks), mirroring what a single Space press on the anchor would do.
func (m *model) applyVisualRange() {
	pane := m.activePane()
	vis := pane.visibleIndices()
	lo, hi := m.visualAnchor, pane.cursor
	if lo > hi {
		lo, hi = hi, lo
	}

	anchorIdx := vis[m.visualAnchor]
	anchorPath := joinSourcePath(m.focus, pane.path, pane.entries[anchorIdx].Name)
	activate := !m.visualSnapshot[anchorIdx] && !m.visualYankSnapshot.has(m.focus, anchorPath)

	selected := cloneSelection(m.visualSnapshot)
	yank := m.visualYankSnapshot
	for pos := lo; pos <= hi && pos < len(vis); pos++ {
		i := vis[pos]
		if pane.entries[i].IsParent {
			continue
		}
		if activate {
			selected[i] = true
			continue
		}
		yank = deactivateEntry(m.focus, pane, selected, yank, i)
	}
	pane.selected = selected
	m.yank = yank
}

func cloneSelection(src map[int]bool) map[int]bool {
	out := make(map[int]bool, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// updateVisual handles key input while ModeVisual is extending a selection
// range.
func (m model) updateVisual(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		newModel, _ := m.moveCursor(1)
		m = newModel.(model)
		m.applyVisualRange()
		return m, nil
	case "k", "up":
		newModel, _ := m.moveCursor(-1)
		m = newModel.(model)
		m.applyVisualRange()
		return m, nil
	case "v":
		m.mode = ModeBrowse
		return m, nil
	case "y":
		m.mode = ModeBrowse
		return m.yankSelection()
	case "esc":
		pane := m.activePane()
		pane.selected = m.visualSnapshot
		m.yank = m.visualYankSnapshot
		m.mode = ModeBrowse
		return m, nil
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}
