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

func TestYankBufferHasMatchesFilesAndDirsOnTheirSourcePane(t *testing.T) {
	y := yankBuffer{sourcePane: 0, files: []string{"/a/f.txt"}, dirs: []string{"/a/d"}}

	if !y.has(0, "/a/f.txt") {
		t.Error("expected a file's path on its source pane to match")
	}
	if !y.has(0, "/a/d") {
		t.Error("expected a dir's path on its source pane to match")
	}
	if y.has(1, "/a/f.txt") {
		t.Error("expected no match on the other pane, even with the same path")
	}
	if y.has(0, "/a/other.txt") {
		t.Error("expected no match for a path not in the buffer")
	}
}
