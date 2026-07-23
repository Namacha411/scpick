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
	m := newModelAt(t, dir) // entries: [file.txt, ..]
	m.local.cursor = 0

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
	m := newModelAt(t, dir) // entries: [sub, ..]
	m.local.cursor = 0

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

func TestUpdateTransferConfirmSendsAnswer(t *testing.T) {
	tests := []struct {
		key  string
		want transfer.OverwriteDecision
	}{
		{"o", transfer.OverwriteYes},
		{"s", transfer.OverwriteSkip},
		{"enter", transfer.OverwriteRename},
		{"esc", transfer.OverwriteSkip},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			m := NewModel()
			m.mode = ModeTransferConfirm
			m.conflictAnswer = make(chan transfer.OverwriteDecision, 1)

			newModel, cmd := m.updateTransferConfirm(keyMsg(tt.key))
			got := newModel.(model)
			if got.mode != ModeTransferProgress {
				t.Fatalf("mode = %v, want ModeTransferProgress", got.mode)
			}
			if cmd == nil {
				t.Fatal("expected a command to send the answer")
			}
			cmd()

			select {
			case decision := <-m.conflictAnswer:
				if decision != tt.want {
					t.Fatalf("decision = %v, want %v", decision, tt.want)
				}
			default:
				t.Fatal("expected a decision to be sent on conflictAnswer")
			}
		})
	}
}

func TestInteractiveOverwriteAsksOnceThenReusesAnswer(t *testing.T) {
	events := make(chan tea.Msg, 1)
	answers := make(chan transfer.OverwriteDecision, 1)
	confirm := interactiveOverwrite(events, answers)

	answers <- transfer.OverwriteRename
	got := confirm("/dest/a.txt", 3, 5)
	if got != transfer.OverwriteRename {
		t.Fatalf("first decision = %v, want OverwriteRename", got)
	}
	select {
	case msg := <-events:
		conflict, ok := msg.(conflictNeededMsg)
		if !ok {
			t.Fatalf("event = %T, want conflictNeededMsg", msg)
		}
		if conflict.destPath != "/dest/a.txt" || conflict.existingSize != 3 || conflict.newSize != 5 {
			t.Fatalf("conflict = %+v, want destPath=/dest/a.txt existingSize=3 newSize=5", conflict)
		}
	default:
		t.Fatal("expected a conflictNeededMsg on the first call")
	}

	// Second call for a different file must reuse the first answer without
	// touching events/answers again.
	got = confirm("/dest/b.txt", 1, 1)
	if got != transfer.OverwriteRename {
		t.Fatalf("second decision = %v, want OverwriteRename (reused)", got)
	}
	select {
	case msg := <-events:
		t.Fatalf("unexpected second event: %+v", msg)
	default:
	}
}

func TestUpdateTransferEventConflictSwitchesToConfirm(t *testing.T) {
	m := NewModel()
	m.mode = ModeTransferProgress
	m.transferEvents = make(chan tea.Msg, 1)

	newModel, cmd := m.updateTransferEvent(conflictNeededMsg{destPath: "/dest/a.txt", existingSize: 3, newSize: 5})
	got := newModel.(model)
	if got.mode != ModeTransferConfirm {
		t.Fatalf("mode = %v, want ModeTransferConfirm", got.mode)
	}
	if got.conflictDestPath != "/dest/a.txt" || got.conflictExistingSize != 3 || got.conflictNewSize != 5 {
		t.Fatalf("conflict fields = %q %d/%d, want /dest/a.txt 3/5", got.conflictDestPath, got.conflictExistingSize, got.conflictNewSize)
	}
	if cmd == nil {
		t.Fatal("expected updateTransferEvent to re-arm the event listener")
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
