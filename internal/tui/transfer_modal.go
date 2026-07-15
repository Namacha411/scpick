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

// beginPaste starts pasting the yank buffer into the focused pane's
// directory: it asks, once for the whole batch, how to handle any
// destination file that already exists (see SPEC.md's Open Questions on
// this being a batch-wide rather than per-file choice).
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
	m.mode = ModeTransferConfirm
	return m, nil
}

// updateTransferConfirm handles the batch-wide overwrite choice for a
// paste, then starts the transfer.
func (m model) updateTransferConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "o":
		return m.startTransfer(fixedOverwrite(transfer.OverwriteAll))
	case "s":
		return m.startTransfer(fixedOverwrite(transfer.OverwriteSkip))
	case "esc":
		m.mode = ModeBrowse
		return m, nil
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func fixedOverwrite(decision transfer.OverwriteDecision) transfer.ConfirmOverwrite {
	return func(string, int64, int64) transfer.OverwriteDecision { return decision }
}

// startTransfer launches the paste's transfer in the background and starts
// listening for its progress/completion events.
func (m model) startTransfer(confirm transfer.ConfirmOverwrite) (tea.Model, tea.Cmd) {
	destDir := m.activePane().path
	pull := m.yank.sourcePane == 1
	files := m.yank.files
	dirs := m.yank.dirs
	client := m.remoteClient

	m.transferEvents = make(chan tea.Msg)
	m.transferLabel = ""
	m.transferDone = 0
	m.transferTotal = 0
	m.mode = ModeTransferProgress

	events := m.transferEvents
	progress := func(label string, done, total int64) {
		events <- transferProgressMsg{label: label, done: done, total: total}
	}

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
	case transferDoneMsg:
		m.transferEvents = nil
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
		"Transfer %d item(s) to %s?\n\n%s",
		m.yank.count(), m.activePane().path,
		m.renderStatusLine("o: overwrite  s: skip existing  esc: cancel"),
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
