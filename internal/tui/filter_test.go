package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func newModelWithNamedFiles(t *testing.T, names ...string) model {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return newModelAt(t, dir)
}

func TestFuzzyMatchIndicesFiltersByName(t *testing.T) {
	entries := []paneEntry{
		{Name: parentEntryName, IsParent: true, IsDir: true},
		{Name: "apple.txt"},
		{Name: "banana.txt"},
		{Name: "grape.txt"},
	}
	got := fuzzyMatchIndices("nan", entries)
	if len(got) != 1 || entries[got[0]].Name != "banana.txt" {
		t.Fatalf("fuzzyMatchIndices(nan) = %v, want just banana.txt's index", got)
	}
}

func TestEnterFilterSeedsInputWithExistingQuery(t *testing.T) {
	m := newModelWithNamedFiles(t, "apple.txt", "banana.txt")
	m.local.filterQuery = "app"

	newModel, _ := m.enterFilter()
	got := newModel.(model)
	if got.mode != ModeFilter {
		t.Fatalf("mode = %v, want ModeFilter", got.mode)
	}
	if got.textInput.Value() != "app" {
		t.Fatalf("textInput.Value() = %q, want %q", got.textInput.Value(), "app")
	}
}

func TestUpdateFilterNarrowsVisibleEntriesAsYouType(t *testing.T) {
	m := newModelWithNamedFiles(t, "apple.txt", "banana.txt", "grape.txt")
	newModel, _ := m.enterFilter()
	m = newModel.(model)

	for _, r := range "grape" {
		newModel, _ = m.updateFilter(keyMsg(string(r)))
		m = newModel.(model)
	}

	if m.local.filterQuery != "grape" {
		t.Fatalf("filterQuery = %q, want %q", m.local.filterQuery, "grape")
	}
	vis := m.local.visibleIndices()
	if len(vis) != 1 || m.local.entries[vis[0]].Name != "grape.txt" {
		t.Fatalf("visibleIndices = %v, want just grape.txt", vis)
	}
}

func TestUpdateFilterEscClearsFilter(t *testing.T) {
	m := newModelWithNamedFiles(t, "apple.txt", "banana.txt")
	newModel, _ := m.enterFilter()
	m = newModel.(model)
	m.local.filterQuery = "app"

	newModel, _ = m.updateFilter(keyMsg("esc"))
	got := newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse", got.mode)
	}
	if got.local.filterQuery != "" {
		t.Fatalf("filterQuery = %q, want cleared", got.local.filterQuery)
	}
}

func TestUpdateFilterEnterKeepsFilterActive(t *testing.T) {
	m := newModelWithNamedFiles(t, "apple.txt", "banana.txt")
	newModel, _ := m.enterFilter()
	m = newModel.(model)
	m.local.filterQuery = "app"

	newModel, _ = m.updateFilter(keyMsg("enter"))
	got := newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse", got.mode)
	}
	if got.local.filterQuery != "app" {
		t.Fatalf("filterQuery = %q, want kept as %q", got.local.filterQuery, "app")
	}
}

func TestLoadPaneClearsFilterAndSelection(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	m := newModelAt(t, parent)
	m.local.filterQuery = "chi" // narrows to just "child", so it's at visible position 0
	m.local.selected = map[int]bool{1: true}
	m.local.cursor = 0

	newModel, _ := m.descend()
	got := newModel.(model)
	if got.local.path != child {
		t.Fatalf("path = %q, want %q", got.local.path, child)
	}
	if got.local.filterQuery != "" {
		t.Fatalf("filterQuery = %q, want cleared after navigation", got.local.filterQuery)
	}
	if len(got.local.selected) != 0 {
		t.Fatalf("selected = %v, want cleared after navigation", got.local.selected)
	}
}
