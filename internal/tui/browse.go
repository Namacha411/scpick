package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"scpick/internal/sshconf"
)

func workingDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("tui: get working directory: %w", err)
	}
	return dir, nil
}

// activePane returns a pointer to whichever pane currently has focus, so
// callers can mutate it in place.
func (m *model) activePane() *paneState {
	if m.focus == 1 {
		return &m.remote
	}
	return &m.local
}

// updateBrowse handles key input on the normal dual-pane browsing screen.
func (m model) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		if m.remoteClient != nil {
			_ = m.remoteClient.Close()
		}
		return m, tea.Quit
	case "C":
		return m.openHostSelect()
	case "tab":
		m.focus = 1 - m.focus
		return m, nil
	case "j", "down":
		return m.moveCursor(1)
	case "k", "up":
		return m.moveCursor(-1)
	case "h", "left":
		return m.ascend()
	case "l", "right", "enter":
		return m.descend()
	case "y":
		return m.yankSelection()
	case "p":
		return m.beginPaste()
	case " ":
		return m.toggleMark()
	case "v":
		return m.enterVisual()
	case "/":
		return m.enterFilter()
	case "?":
		m.mode = ModeHelp
		return m, nil
	}
	return m, nil
}

// yankSelection records the pane's marked entries (or, if nothing is
// marked, just the entry under the cursor) as the yank buffer, replacing
// any previous yank. Marks are cleared afterward, mirroring cut-after-yank
// file manager conventions.
func (m model) yankSelection() (tea.Model, tea.Cmd) {
	pane := m.activePane()

	indices := markedIndices(pane)
	if len(indices) == 0 {
		if i, ok := pane.currentIndex(); ok {
			indices = []int{i}
		}
	}

	yank := yankBuffer{sourcePane: m.focus}
	for _, i := range indices {
		entry := pane.entries[i]
		if entry.IsParent {
			continue
		}
		src := joinSourcePath(m.focus, pane.path, entry.Name)
		if entry.IsDir {
			yank.dirs = append(yank.dirs, src)
		} else {
			yank.files = append(yank.files, src)
		}
	}
	if yank.empty() {
		return m, nil
	}

	pane.selected = nil
	m.yank = yank
	m.status = fmt.Sprintf("yanked %d item(s)", yank.count())
	return m, nil
}

// markedIndices returns the sorted entry indices currently marked in pane.
func markedIndices(pane *paneState) []int {
	indices := make([]int, 0, len(pane.selected))
	for i := range pane.entries {
		if pane.selected[i] {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m model) moveCursor(delta int) (tea.Model, tea.Cmd) {
	pane := m.activePane()
	last := len(pane.visibleIndices()) - 1
	next := pane.cursor + delta
	if next < 0 {
		next = 0
	}
	if next > last {
		next = last
	}
	if next < 0 {
		next = 0
	}
	pane.cursor = next
	return m, nil
}

// ascend navigates the focused pane to its current directory's parent.
func (m model) ascend() (tea.Model, tea.Cmd) {
	if m.focus == 1 && m.remoteClient == nil {
		return m, nil
	}
	pane := m.activePane()
	var parent string
	if m.focus == 0 {
		parent = localParent(pane.path)
	} else {
		parent = remoteParent(pane.path)
	}
	return m.loadPane(parent)
}

// descend navigates into the entry under the cursor: ".." goes to the
// parent directory, a directory is entered, and (until Milestone 3 wires up
// transfers) a file is a no-op.
func (m model) descend() (tea.Model, tea.Cmd) {
	if m.focus == 1 && m.remoteClient == nil {
		return m, nil
	}
	pane := m.activePane()
	entry, ok := pane.current()
	if !ok || !entry.IsDir {
		return m, nil
	}
	if entry.IsParent {
		return m.ascend()
	}

	return m.loadPane(joinSourcePath(m.focus, pane.path, entry.Name))
}

// loadPane lists dir on the focused pane's filesystem and, on success,
// switches that pane to it: cursor reset to the top, marks and any active
// filter cleared, since both are scoped to the directory being left.
func (m model) loadPane(dir string) (tea.Model, tea.Cmd) {
	pane := m.activePane()
	var entries []paneEntry
	var err error
	if m.focus == 0 {
		entries, err = listLocalEntries(dir)
	} else {
		entries, err = listRemoteEntries(m.remoteClient, dir)
	}
	if err != nil {
		m.errMsg = err.Error()
		return m, nil
	}
	pane.path = dir
	pane.entries = entries
	pane.cursor = 0
	pane.selected = nil
	pane.filterQuery = ""
	m.errMsg = ""
	return m, nil
}

// openHostSelect loads the ~/.ssh/config host list and switches to
// ModeHostSelect. Loading errors are shown in the status line rather than
// blocking entry to the mode, since manual entry is always available.
func (m model) openHostSelect() (tea.Model, tea.Cmd) {
	m.errMsg = ""
	cfg, err := sshconf.LoadConfig()
	if err != nil {
		m.sshHosts = nil
		m.errMsg = fmt.Sprintf("load ~/.ssh/config: %v", err)
	} else {
		m.sshHosts = cfg.Hosts()
	}
	m.hostCursor = 0
	m.mode = ModeHostSelect
	return m, nil
}
