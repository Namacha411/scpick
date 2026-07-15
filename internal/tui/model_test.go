package tui

import "testing"

func TestNewModelStartsInHostSelectWithLocalPanePreloaded(t *testing.T) {
	m := NewModel()
	if m.mode != ModeHostSelect {
		t.Fatalf("mode = %v, want ModeHostSelect (connecting is the near-universal first action)", m.mode)
	}
	if m.local.path == "" {
		t.Fatal("local.path is empty, want current working directory")
	}
	if len(m.local.entries) == 0 {
		t.Fatal("local.entries is empty, want the cwd's listing preloaded")
	}
	if m.remoteClient != nil {
		t.Fatal("remoteClient should be nil before any connection is made")
	}
}

func TestNewModelHostSelectCanBeDismissedToBrowse(t *testing.T) {
	m := NewModel()
	newModel, _ := m.updateHostSelect(keyMsg("esc"))
	got := newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse after Esc", got.mode)
	}
}
