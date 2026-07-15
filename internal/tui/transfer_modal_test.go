package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"scpick/internal/transfer"
)

func TestYankSelectionRecordsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModelAt(t, dir) // entries: [.., file.txt]
	m.local.cursor = 1

	newModel, _ := m.yankSelection()
	got := newModel.(model)
	if got.yank.sourcePane != 0 {
		t.Fatalf("sourcePane = %d, want 0", got.yank.sourcePane)
	}
	want := filepath.Join(dir, "file.txt")
	if len(got.yank.files) != 1 || got.yank.files[0] != want {
		t.Fatalf("yank.files = %v, want [%q]", got.yank.files, want)
	}
	if len(got.yank.dirs) != 0 {
		t.Fatalf("yank.dirs = %v, want empty", got.yank.dirs)
	}
}

func TestYankSelectionRecordsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := newModelAt(t, dir) // entries: [.., sub]
	m.local.cursor = 1

	newModel, _ := m.yankSelection()
	got := newModel.(model)
	want := filepath.Join(dir, "sub")
	if len(got.yank.dirs) != 1 || got.yank.dirs[0] != want {
		t.Fatalf("yank.dirs = %v, want [%q]", got.yank.dirs, want)
	}
}

func TestYankSelectionOnParentEntryIsNoop(t *testing.T) {
	dir := t.TempDir()
	m := newModelAt(t, dir) // cursor starts on ".."

	newModel, _ := m.yankSelection()
	got := newModel.(model)
	if !got.yank.empty() {
		t.Fatalf("yank = %+v, want empty", got.yank)
	}
}

func TestBeginPasteRequiresNonEmptyYank(t *testing.T) {
	m := NewModel()
	m.mode = ModeBrowse
	newModel, _ := m.beginPaste()
	got := newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse (nothing yanked)", got.mode)
	}
}

func TestBeginPasteRejectsSameSourceAndDestPane(t *testing.T) {
	m := NewModel()
	m.mode = ModeBrowse
	m.yank = yankBuffer{sourcePane: 0, files: []string{"/some/file"}}
	m.focus = 0

	newModel, _ := m.beginPaste()
	got := newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse", got.mode)
	}
	if got.errMsg == "" {
		t.Fatal("expected an error when pasting into the yank's own source pane")
	}
}

func TestBeginPasteRequiresConnection(t *testing.T) {
	m := NewModel()
	m.mode = ModeBrowse
	m.yank = yankBuffer{sourcePane: 0, files: []string{"/some/file"}}
	m.focus = 1 // remote, but remoteClient is nil

	newModel, _ := m.beginPaste()
	got := newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse", got.mode)
	}
	if got.errMsg == "" {
		t.Fatal("expected an error when the remote pane isn't connected")
	}
}

func TestUpdateTransferConfirmEscCancels(t *testing.T) {
	m := NewModel()
	m.mode = ModeTransferConfirm
	m.yank = yankBuffer{sourcePane: 1, files: []string{"/some/file"}}

	newModel, cmd := m.updateTransferConfirm(keyMsg("esc"))
	got := newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse", got.mode)
	}
	if cmd != nil {
		t.Fatal("expected no command on cancel")
	}
}

func TestUpdateTransferEventProgressUpdatesFieldsAndRearms(t *testing.T) {
	m := NewModel()
	m.mode = ModeTransferProgress
	m.transferEvents = make(chan tea.Msg)

	newModel, cmd := m.updateTransferEvent(transferProgressMsg{label: "file.txt", done: 5, total: 10})
	got := newModel.(model)
	if got.transferLabel != "file.txt" || got.transferDone != 5 || got.transferTotal != 10 {
		t.Fatalf("progress fields = %q %d/%d, want file.txt 5/10", got.transferLabel, got.transferDone, got.transferTotal)
	}
	if cmd == nil {
		t.Fatal("expected updateTransferEvent to re-arm the event listener")
	}
}

func TestUpdateTransferEventDoneReloadsDestPaneAndSummarizes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModelAt(t, dir)
	m.focus = 0 // destination pane is local, so reloading needs no network

	result := transfer.Result{
		Succeeded: []string{"a.txt"},
		Skipped:   []string{"b.txt"},
		Failed:    map[string]error{},
	}
	newModel, _ := m.updateTransferEvent(transferDoneMsg{result: result})
	got := newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse", got.mode)
	}
	if got.status != "1 succeeded, 1 skipped, 0 failed" {
		t.Fatalf("status = %q", got.status)
	}
	if got.errMsg != "" {
		t.Fatalf("errMsg = %q, want empty (no failures)", got.errMsg)
	}
}
