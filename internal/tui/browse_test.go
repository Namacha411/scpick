package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"scpick/internal/sshconf"
)

func TestUpdateBrowseQuitClosesRemoteClientAndQuits(t *testing.T) {
	m := NewModel()
	newModel, cmd := m.updateBrowse(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := newModel.(model)
	if !got.quitting {
		t.Fatal("quitting = false, want true after 'q'")
	}
	if cmd == nil {
		t.Fatal("expected a tea.Quit command, got nil")
	}
}

func TestUpdateBrowseCKeyOpensHostSelect(t *testing.T) {
	m := NewModel()
	newModel, _ := m.updateBrowse(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	got := newModel.(model)
	if got.mode != ModeHostSelect {
		t.Fatalf("mode = %v, want ModeHostSelect", got.mode)
	}
}

func TestOpenHostSelectResetsCursorAndError(t *testing.T) {
	m := NewModel()
	m.hostCursor = 3
	m.errMsg = "stale error"
	newModel, _ := m.openHostSelect()
	got := newModel.(model)
	if got.mode != ModeHostSelect {
		t.Fatalf("mode = %v, want ModeHostSelect", got.mode)
	}
	if got.hostCursor != 0 {
		t.Fatalf("hostCursor = %d, want 0", got.hostCursor)
	}
}

func newModelAt(t *testing.T, dir string) model {
	t.Helper()
	entries, err := listLocalEntries(dir)
	if err != nil {
		t.Fatalf("listLocalEntries: %v", err)
	}
	m := NewModel()
	m.local = paneState{path: dir, entries: entries}
	return m
}

func TestMoveCursorIsBoundedByEntryCount(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := newModelAt(t, dir) // entries: [.., sub] = 2 entries

	newModel, _ := m.updateBrowse(keyMsg("up")) // already at 0, must clamp
	got := newModel.(model)
	if got.local.cursor != 0 {
		t.Fatalf("cursor = %d, want 0", got.local.cursor)
	}

	newModel, _ = got.updateBrowse(keyMsg("down"))
	got = newModel.(model)
	if got.local.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", got.local.cursor)
	}

	newModel, _ = got.updateBrowse(keyMsg("down")) // past the end, must clamp
	got = newModel.(model)
	if got.local.cursor != 1 {
		t.Fatalf("cursor = %d, want 1 (clamped)", got.local.cursor)
	}
}

func TestTabTogglesFocus(t *testing.T) {
	m := NewModel()
	if m.focus != 0 {
		t.Fatalf("initial focus = %d, want 0", m.focus)
	}
	newModel, _ := m.updateBrowse(tea.KeyMsg{Type: tea.KeyTab})
	got := newModel.(model)
	if got.focus != 1 {
		t.Fatalf("focus = %d, want 1", got.focus)
	}
	newModel, _ = got.updateBrowse(tea.KeyMsg{Type: tea.KeyTab})
	got = newModel.(model)
	if got.focus != 0 {
		t.Fatalf("focus = %d, want 0", got.focus)
	}
}

func TestDescendIntoDirectoryThenAscendBack(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	m := newModelAt(t, parent) // entries: [.., child]
	m.local.cursor = 1         // "child"

	newModel, _ := m.updateBrowse(keyMsg("l"))
	got := newModel.(model)
	if got.local.path != child {
		t.Fatalf("local.path = %q, want %q", got.local.path, child)
	}
	if got.local.cursor != 0 {
		t.Fatalf("cursor after descend = %d, want 0", got.local.cursor)
	}

	newModel, _ = got.updateBrowse(keyMsg("h"))
	got = newModel.(model)
	if got.local.path != parent {
		t.Fatalf("local.path after ascend = %q, want %q", got.local.path, parent)
	}
}

func TestDashAndBackspaceAlsoAscend(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, key := range []tea.KeyMsg{keyMsg("-"), {Type: tea.KeyBackspace}} {
		m := newModelAt(t, child)
		newModel, _ := m.updateBrowse(key)
		got := newModel.(model)
		if got.local.path != parent {
			t.Fatalf("key %v: local.path = %q, want %q", key, got.local.path, parent)
		}
	}
}

func TestDescendOnParentEntryAscends(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	m := newModelAt(t, child)
	m.local.cursor = 0 // ".."

	newModel, _ := m.updateBrowse(keyMsg("enter"))
	got := newModel.(model)
	if got.local.path != parent {
		t.Fatalf("local.path = %q, want %q", got.local.path, parent)
	}
}

func TestDescendOnFileIsNoop(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModelAt(t, dir) // entries: [.., file.txt]
	m.local.cursor = 1

	newModel, _ := m.updateBrowse(keyMsg("l"))
	got := newModel.(model)
	if got.local.path != dir {
		t.Fatalf("local.path = %q, want unchanged %q", got.local.path, dir)
	}
}

func TestAscendAndDescendAreNoopsOnDisconnectedRemotePane(t *testing.T) {
	m := NewModel()
	m.focus = 1 // remote, but remoteClient is nil

	newModel, _ := m.updateBrowse(keyMsg("h"))
	got := newModel.(model)
	if got.remote.path != "" {
		t.Fatalf("remote.path = %q, want unchanged empty", got.remote.path)
	}

	newModel, _ = m.updateBrowse(keyMsg("l"))
	got = newModel.(model)
	if got.remote.path != "" {
		t.Fatalf("remote.path = %q, want unchanged empty", got.remote.path)
	}
}

func TestUpdateBrowseUnknownKeyIsNoop(t *testing.T) {
	m := NewModel()
	m.mode = ModeBrowse
	m.sshHosts = []sshconf.Host{{Name: "example"}}
	newModel, cmd := m.updateBrowse(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	got := newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse", got.mode)
	}
	if cmd != nil {
		t.Fatal("expected no command for an unbound key")
	}
}
