package tui

import tea "github.com/charmbracelet/bubbletea"

// enterFilter switches the focused pane into incremental fuzzy-filter
// input, seeded with any filter already active on it.
func (m model) enterFilter() (tea.Model, tea.Cmd) {
	pane := m.activePane()
	m.textInput = newTextInput("filter", false)
	m.textInput.SetValue(pane.filterQuery)
	m.textInput.CursorEnd()
	m.mode = ModeFilter
	return m, nil
}

// updateFilter handles key input while typing an incremental fuzzy filter.
// Enter keeps the filter active and returns to Browse; Esc clears it
// entirely and returns to Browse.
func (m model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.mode = ModeBrowse
		return m, nil
	case "esc":
		pane := m.activePane()
		pane.filterQuery = ""
		pane.cursor = 0
		m.mode = ModeBrowse
		return m, nil
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	pane := m.activePane()
	pane.filterQuery = m.textInput.Value()
	pane.cursor = 0
	return m, cmd
}
