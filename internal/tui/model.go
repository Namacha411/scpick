// Package tui implements scpick's dual-pane (local + remote) file manager,
// built on charmbracelet/bubbletea and charmbracelet/lipgloss. See SPEC.md
// for the overall design and keybindings.
package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"scpick/internal/remotefs"
	"scpick/internal/sshconf"
	"scpick/internal/transfer"
)

// mode identifies which screen/dialog is currently driving key input.
type mode int

const (
	// ModeBrowse is the normal dual-pane browsing screen.
	ModeBrowse mode = iota
	// ModeHostSelect shows the ~/.ssh/config host list plus a manual-entry
	// option.
	ModeHostSelect
	// ModeManualHost collects hostname/user/port for a manually-entered
	// host, one field at a time.
	ModeManualHost
	// ModeConnecting is shown while a connection attempt is in flight and
	// no prompt is currently needed from the user.
	ModeConnecting
	// ModePasswordPrompt asks for a password or key passphrase, requested
	// by the in-flight connection attempt's auth chain.
	ModePasswordPrompt
	// ModeHostKeyConfirm asks whether to trust an unknown host key,
	// requested by the in-flight connection attempt.
	ModeHostKeyConfirm
	// ModeTransferConfirm asks how to handle a destination file that
	// already exists. It only appears once an in-flight transfer actually
	// hits a conflict; the answer then applies to the rest of the batch.
	ModeTransferConfirm
	// ModeTransferProgress shows progress while a paste's transfer is in
	// flight.
	ModeTransferProgress
	// ModeVisual extends a selection range from an anchor as the cursor
	// moves, on top of whatever was already marked with Space.
	ModeVisual
	// ModeFilter incrementally narrows the focused pane to entries whose
	// name fuzzy-matches the typed query.
	ModeFilter
	// ModeHelp shows the keybinding reference.
	ModeHelp
)

// paneState holds one side of the dual-pane browser: its current directory,
// the listing for that directory, the cursor position within the *visible*
// (possibly filtered) entries, which entries are marked, and any active
// filter query.
type paneState struct {
	path        string
	entries     []paneEntry
	cursor      int
	selected    map[int]bool // keyed by index into entries, not into the visible/filtered subset
	filterQuery string
}

// visibleIndices returns the indices into entries that should currently be
// shown, in display order: every entry, unless a filter query is active, in
// which case only the fuzzy-matching entries, best match first.
func (p *paneState) visibleIndices() []int {
	if p.filterQuery == "" {
		idx := make([]int, len(p.entries))
		for i := range p.entries {
			idx[i] = i
		}
		return idx
	}
	return fuzzyMatchIndices(p.filterQuery, p.entries)
}

// currentIndex returns the index into entries of the entry under the
// cursor (resolved through any active filter), or false if the pane is
// empty or the filter matched nothing.
func (p *paneState) currentIndex() (int, bool) {
	vis := p.visibleIndices()
	if p.cursor < 0 || p.cursor >= len(vis) {
		return -1, false
	}
	return vis[p.cursor], true
}

func (p *paneState) current() (paneEntry, bool) {
	i, ok := p.currentIndex()
	if !ok {
		return paneEntry{}, false
	}
	return p.entries[i], true
}

func (p *paneState) isSelected(entryIndex int) bool {
	return p.selected[entryIndex]
}

// manualHostStep tracks which field of the manual host entry wizard is
// currently being collected.
type manualHostStep int

const (
	stepHostname manualHostStep = iota
	stepUser
	stepPort
)

// manualHostForm accumulates the fields of a manually-entered host across
// the three-step wizard driven by ModeManualHost.
type manualHostForm struct {
	step     manualHostStep
	hostname string
	user     string
}

// yankBuffer holds what was last yanked with "y": the source pane (0 =
// local, 1 = remote) and the full source paths of the plain files and
// directories yanked. A directory present here is always transferred
// recursively.
type yankBuffer struct {
	sourcePane int
	files      []string
	dirs       []string
}

func (y yankBuffer) empty() bool {
	return len(y.files) == 0 && len(y.dirs) == 0
}

func (y yankBuffer) count() int {
	return len(y.files) + len(y.dirs)
}

// Fallback terminal size used only until the first tea.WindowSizeMsg
// arrives (bubbletea sends one immediately on startup, so this is mostly
// defensive).
const (
	defaultTermWidth  = 80
	defaultTermHeight = 24
)

// model is scpick's top-level bubbletea model.
type model struct {
	local  paneState
	remote paneState
	focus  int // 0 = local, 1 = remote

	termWidth  int
	termHeight int

	mode mode

	remoteClient *remotefs.Client

	sshHosts   []sshconf.Host
	hostCursor int
	manualHost manualHostForm

	textInput textinput.Model

	pendingHost sshconf.Host

	// In-flight connection attempt plumbing: connectEvents carries
	// passwordNeededMsg/hostKeyNeededMsg/connectResultMsg from the
	// background connect goroutine back into Update(); passwordAnswer and
	// hostKeyAnswer carry the user's replies back to that goroutine.
	connectEvents  chan tea.Msg
	passwordAnswer chan string
	hostKeyAnswer  chan bool

	passwordPrompt     string
	hostKeyHostname    string
	hostKeyFingerprint string

	yank yankBuffer

	// visualAnchor and visualSnapshot support ModeVisual: the anchor is the
	// cursor position when 'v' was pressed, and the snapshot is the
	// pane's selection at that moment, so Esc can restore it.
	visualAnchor   int
	visualSnapshot map[int]bool

	// In-flight transfer plumbing: transferEvents carries
	// transferProgressMsg/conflictNeededMsg/transferDoneMsg back from the
	// background transfer goroutine; conflictAnswer carries the user's
	// overwrite/skip/rename reply back to it.
	transferEvents chan tea.Msg
	conflictAnswer chan transfer.OverwriteDecision
	transferLabel  string
	transferDone   int64
	transferTotal  int64

	// conflictDestPath/conflictExistingSize/conflictNewSize describe the
	// in-flight conflict ModeTransferConfirm is currently showing.
	conflictDestPath     string
	conflictExistingSize int64
	conflictNewSize      int64

	status string
	errMsg string

	quitting bool
}

// NewModel builds the initial model: the local pane starts at the current
// working directory, listed immediately; the remote pane starts
// disconnected and empty. Since connecting to a remote host is the near-
// universal first action, startup goes straight into ModeHostSelect
// (equivalent to pressing "C" immediately) instead of requiring it — Esc
// backs out to plain Browse mode if that's not wanted.
func NewModel() model {
	cwd, err := workingDir()
	m := model{
		local:      paneState{path: cwd},
		remote:     paneState{path: ""},
		textInput:  textinput.New(),
		termWidth:  defaultTermWidth,
		termHeight: defaultTermHeight,
	}
	if err != nil {
		m.errMsg = err.Error()
		return m
	}
	entries, err := listLocalEntries(cwd)
	if err != nil {
		m.errMsg = err.Error()
		return m
	}
	m.local.entries = entries

	newModel, _ := m.openHostSelect()
	return newModel.(model)
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		return m, nil
	case passwordNeededMsg, hostKeyNeededMsg, connectResultMsg:
		return m.updateConnectEvent(msg)
	case transferProgressMsg, conflictNeededMsg, transferDoneMsg:
		return m.updateTransferEvent(msg)
	}
	return m, nil
}

func (m model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case ModeHostSelect:
		return m.updateHostSelect(msg)
	case ModeManualHost:
		return m.updateManualHost(msg)
	case ModePasswordPrompt:
		return m.updatePasswordPrompt(msg)
	case ModeHostKeyConfirm:
		return m.updateHostKeyConfirm(msg)
	case ModeTransferConfirm:
		return m.updateTransferConfirm(msg)
	case ModeVisual:
		return m.updateVisual(msg)
	case ModeFilter:
		return m.updateFilter(msg)
	case ModeHelp:
		return m.updateHelp(msg)
	case ModeConnecting, ModeTransferProgress:
		return m, nil
	default:
		return m.updateBrowse(msg)
	}
}

func (m model) View() string {
	switch m.mode {
	case ModeHostSelect:
		return m.viewHostSelect()
	case ModeManualHost:
		return m.viewManualHost()
	case ModeConnecting:
		return m.viewConnecting()
	case ModePasswordPrompt:
		return m.viewPasswordPrompt()
	case ModeHostKeyConfirm:
		return m.viewHostKeyConfirm()
	case ModeTransferConfirm:
		return m.viewTransferConfirm()
	case ModeTransferProgress:
		return m.viewTransferProgress()
	case ModeHelp:
		return m.viewHelp()
	default:
		return m.viewBrowse()
	}
}
