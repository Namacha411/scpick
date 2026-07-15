package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"scpick/internal/sshconf"
)

// TestViewDoesNotPanicInAnyMode is a smoke test: View() must render some
// string without panicking, in every mode, regardless of what partial state
// that mode happens to hold (SPEC.md sets no correctness bar on rendering
// itself, only that it doesn't crash).
func TestViewDoesNotPanicInAnyMode(t *testing.T) {
	base := newModelWithThreeFiles(t)
	base.sshHosts = []sshconf.Host{{Name: "example", Hostname: "example.com", User: "alice", Port: 22}}
	base.hostCursor = 0
	base.manualHost = manualHostForm{step: stepHostname, hostname: "h", user: "u"}
	base.textInput = newTextInput("x", false)
	base.passwordPrompt = "alice's password: "
	base.hostKeyHostname = "example.com"
	base.hostKeyFingerprint = "SHA256:abc"
	base.yank = yankBuffer{sourcePane: 0, files: []string{"/a"}}
	base.transferLabel = "file.txt"
	base.transferDone = 5
	base.transferTotal = 10
	base.status = "some status"

	for _, mode := range []mode{
		ModeBrowse,
		ModeHostSelect,
		ModeManualHost,
		ModeConnecting,
		ModePasswordPrompt,
		ModeHostKeyConfirm,
		ModeTransferConfirm,
		ModeTransferProgress,
		ModeVisual,
		ModeFilter,
		ModeHelp,
	} {
		m := base
		m.mode = mode
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("View() panicked in mode %v: %v", mode, r)
				}
			}()
			if out := m.View(); out == "" {
				t.Fatalf("View() in mode %v returned an empty string", mode)
			}
		}()
	}
}

func TestUpdateHelpReturnsToBrowseOnAnyKey(t *testing.T) {
	m := NewModel()
	m.mode = ModeHelp

	newModel, _ := m.updateHelp(keyMsg("x"))
	got := newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse", got.mode)
	}
}

func TestUpdateHelpCtrlCQuits(t *testing.T) {
	m := NewModel()
	m.mode = ModeHelp

	newModel, cmd := m.updateHelp(tea.KeyMsg{Type: tea.KeyCtrlC})
	got := newModel.(model)
	if !got.quitting {
		t.Fatal("expected quitting = true")
	}
	if cmd == nil {
		t.Fatal("expected a tea.Quit command")
	}
}

func TestBrowseQuestionMarkEntersHelp(t *testing.T) {
	m := NewModel()
	newModel, _ := m.updateBrowse(keyMsg("?"))
	got := newModel.(model)
	if got.mode != ModeHelp {
		t.Fatalf("mode = %v, want ModeHelp", got.mode)
	}
}
