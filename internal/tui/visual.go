package tui

import tea "github.com/charmbracelet/bubbletea"

// toggleMark toggles the mark on the entry under the cursor (available
// directly from Browse mode, independent of ModeVisual).
func (m model) toggleMark() (tea.Model, tea.Cmd) {
	pane := m.activePane()
	i, ok := pane.currentIndex()
	if !ok || pane.entries[i].IsParent {
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
// switches to ModeVisual, remembering the pane's prior marks so Esc can
// restore them.
func (m model) enterVisual() (tea.Model, tea.Cmd) {
	pane := m.activePane()
	m.visualAnchor = pane.cursor
	m.visualSnapshot = cloneSelection(pane.selected)
	m.mode = ModeVisual
	m.applyVisualRange()
	return m, nil
}

// applyVisualRange marks every visible entry between the visual anchor and
// the current cursor (inclusive), on top of the snapshot taken when
// ModeVisual was entered.
func (m *model) applyVisualRange() {
	pane := m.activePane()
	vis := pane.visibleIndices()
	lo, hi := m.visualAnchor, pane.cursor
	if lo > hi {
		lo, hi = hi, lo
	}

	selected := cloneSelection(m.visualSnapshot)
	for pos := lo; pos <= hi && pos < len(vis); pos++ {
		i := vis[pos]
		if !pane.entries[i].IsParent {
			selected[i] = true
		}
	}
	pane.selected = selected
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
		m.mode = ModeBrowse
		return m, nil
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}
