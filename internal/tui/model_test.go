package tui

import "testing"

func TestNewModelStartsInBrowseModeWithLocalPathSet(t *testing.T) {
	m := NewModel()
	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse", m.mode)
	}
	if m.local.path == "" {
		t.Fatal("local.path is empty, want current working directory")
	}
	if m.remoteClient != nil {
		t.Fatal("remoteClient should be nil before any connection is made")
	}
}
