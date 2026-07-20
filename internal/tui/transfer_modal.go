package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"scpick/internal/remotefs"
	"scpick/internal/transfer"
)

// transferProgressMsg is emitted by the in-flight transfer goroutine as
// each file copies.
type transferProgressMsg struct {
	label       string
	done, total int64
}

// transferDoneMsg is emitted once, terminally, when the transfer goroutine
// finishes.
type transferDoneMsg struct {
	result transfer.Result
}

// conflictNeededMsg is emitted by the in-flight transfer goroutine when it
// hits a destination file that already exists and needs the user's
// overwrite/skip/rename decision.
type conflictNeededMsg struct {
	destPath              string
	existingSize, newSize int64
}

// beginPaste starts pasting the yank buffer into the focused pane's
// directory immediately. If a destination file already exists, the
// transfer goroutine pauses on that conflict and ModeTransferConfirm asks
// how to handle it; the answer then applies to the rest of the batch (see
// SPEC.md's Open Questions on this being a batch-wide rather than per-file
// choice). A paste with no conflicts at all never shows a modal.
func (m model) beginPaste() (tea.Model, tea.Cmd) {
	if m.yank.empty() {
		return m, nil
	}
	if m.focus == m.yank.sourcePane {
		m.errMsg = "switch to the other pane before pasting"
		return m, nil
	}
	if m.remoteClient == nil {
		m.errMsg = "not connected"
		return m, nil
	}
	m.errMsg = ""
	return m.startTransfer()
}

// updateTransferConfirm answers the conflict that paused the in-flight
// transfer goroutine.
func (m model) updateTransferConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "o":
		return m.answerConflict(transfer.OverwriteYes)
	case "s":
		return m.answerConflict(transfer.OverwriteSkip)
	case "enter":
		return m.answerConflict(transfer.OverwriteRename)
	case "esc":
		// Aborting the in-flight transfer outright isn't implemented yet
		// (see CLAUDE.md's "Known gap" on cancellation); skipping every
		// remaining conflict is the closest safe approximation.
		return m.answerConflict(transfer.OverwriteSkip)
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

// answerConflict delivers the user's reply to the conflict that paused the
// in-flight transfer goroutine. The event listener is already running
// (updateTransferEvent re-armed it when the conflict was reported), so
// this only needs to send the answer.
func (m model) answerConflict(decision transfer.OverwriteDecision) (tea.Model, tea.Cmd) {
	answers := m.conflictAnswer
	m.mode = ModeTransferProgress
	return m, sendConflictAnswer(answers, decision)
}

// sendConflictAnswer delivers the user's decision back to the blocked
// transfer goroutine without blocking the UI thread.
func sendConflictAnswer(answers chan transfer.OverwriteDecision, decision transfer.OverwriteDecision) tea.Cmd {
	return func() tea.Msg {
		answers <- decision
		return nil
	}
}

func fixedOverwrite(decision transfer.OverwriteDecision) transfer.ConfirmOverwrite {
	return func(string, int64, int64) transfer.OverwriteDecision { return decision }
}

// interactiveOverwrite blocks the transfer goroutine on the first conflict
// it hits, asking the TUI how to resolve it; the same answer is then
// reused for the rest of the batch without asking again.
func interactiveOverwrite(events chan tea.Msg, answers chan transfer.OverwriteDecision) transfer.ConfirmOverwrite {
	var decided bool
	var decision transfer.OverwriteDecision
	return func(destPath string, existingSize, newSize int64) transfer.OverwriteDecision {
		if decided {
			return decision
		}
		events <- conflictNeededMsg{destPath: destPath, existingSize: existingSize, newSize: newSize}
		decision = <-answers
		decided = true
		return decision
	}
}

// startTransfer launches the paste's transfer in the background and starts
// listening for its progress/conflict/completion events.
func (m model) startTransfer() (tea.Model, tea.Cmd) {
	destDir := m.activePane().path
	pull := m.yank.sourcePane == 1
	files := m.yank.files
	dirs := m.yank.dirs
	client := m.remoteClient

	m.transferEvents = make(chan tea.Msg)
	m.conflictAnswer = make(chan transfer.OverwriteDecision)
	m.transferLabel = ""
	m.transferDone = 0
	m.transferTotal = 0
	m.mode = ModeTransferProgress

	events := m.transferEvents
	answers := m.conflictAnswer
	progress := func(label string, done, total int64) {
		events <- transferProgressMsg{label: label, done: done, total: total}
	}
	confirm := interactiveOverwrite(events, answers)

	return m, tea.Batch(
		startTransferWorker(client, pull, files, dirs, destDir, confirm, progress, events),
		waitForTransferEvent(events),
	)
}

// startTransferWorker runs the actual transfer in its own goroutine and
// returns immediately, so it never blocks bubbletea's event loop.
func startTransferWorker(client *remotefs.Client, pull bool, files, dirs []string, destDir string, confirm transfer.ConfirmOverwrite, progress transfer.ProgressPrinter, events chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		go func() {
			var result transfer.Result
			switch {
			case pull && len(dirs) > 0:
				result = transfer.RecursivePull(client, files, dirs, destDir, confirm, progress)
			case pull:
				result = transfer.Pull(client, files, destDir, confirm, progress)
			case len(dirs) > 0:
				result = transfer.RecursivePush(client, files, dirs, destDir, confirm, progress)
			default:
				result = transfer.Push(client, files, destDir, confirm, progress)
			}
			events <- transferDoneMsg{result: result}
		}()
		return nil
	}
}

// waitForTransferEvent blocks until the next message arrives on events. It
// must be re-issued after every non-terminal event, mirroring
// waitForConnectEvent in connect.go.
func waitForTransferEvent(events chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-events
	}
}

// updateTransferEvent handles messages emitted by the in-flight transfer
// goroutine.
func (m model) updateTransferEvent(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case transferProgressMsg:
		m.transferLabel = msg.label
		m.transferDone = msg.done
		m.transferTotal = msg.total
		return m, waitForTransferEvent(m.transferEvents)
	case conflictNeededMsg:
		m.conflictDestPath = msg.destPath
		m.conflictExistingSize = msg.existingSize
		m.conflictNewSize = msg.newSize
		m.mode = ModeTransferConfirm
		return m, waitForTransferEvent(m.transferEvents)
	case transferDoneMsg:
		m.transferEvents = nil
		m.conflictAnswer = nil
		m.status = summarizeResult(msg.result)
		m.errMsg = firstFailure(msg.result.Failed)
		m.mode = ModeBrowse
		return m.loadPane(m.activePane().path)
	}
	return m, nil
}

func summarizeResult(r transfer.Result) string {
	return fmt.Sprintf("%d succeeded, %d skipped, %d failed", len(r.Succeeded), len(r.Skipped), len(r.Failed))
}

func firstFailure(failed map[string]error) string {
	for path, err := range failed {
		return fmt.Sprintf("%s: %v", path, err)
	}
	return ""
}

func (m model) viewTransferConfirm() string {
	return fmt.Sprintf(
		"%s already exists (%s → %s)\n\n%s",
		m.conflictDestPath, formatBytesShort(m.conflictExistingSize), formatBytesShort(m.conflictNewSize),
		m.renderStatusLine("o: overwrite  s: skip  enter: keep both (rename)  esc: skip rest"),
	)
}

func (m model) viewTransferProgress() string {
	bar := renderProgressBar(m.transferDone, m.transferTotal, 20)
	return fmt.Sprintf(
		"Transferring %s\n%s %s/%s\n\n%s",
		m.transferLabel, bar, formatBytesShort(m.transferDone), formatBytesShort(m.transferTotal),
		m.renderStatusLine(""),
	)
}

func renderProgressBar(done, total int64, width int) string {
	filled := 0
	if total > 0 {
		filled = int(float64(width) * float64(done) / float64(total))
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func formatBytesShort(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
