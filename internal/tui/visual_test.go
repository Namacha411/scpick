package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func newModelWithThreeFiles(t *testing.T) model {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// entries: [a.txt, b.txt, c.txt, ..]
	return newModelAt(t, dir)
}

func TestToggleMarkAddsAndRemoves(t *testing.T) {
	m := newModelWithThreeFiles(t)
	m.local.cursor = 0 // a.txt

	newModel, _ := m.toggleMark()
	got := newModel.(model)
	if !got.local.isSelected(0) {
		t.Fatal("expected index 0 to be marked after toggling")
	}

	newModel, _ = got.toggleMark()
	got = newModel.(model)
	if got.local.isSelected(0) {
		t.Fatal("expected index 0 to be unmarked after toggling again")
	}
}

func TestToggleMarkOnParentEntryIsNoop(t *testing.T) {
	m := newModelWithThreeFiles(t)
	m.local.cursor = 3 // ".."

	newModel, _ := m.toggleMark()
	got := newModel.(model)
	if len(got.local.selected) != 0 {
		t.Fatalf("selected = %v, want empty", got.local.selected)
	}
}

func TestEnterVisualThenExtendRangeThenYank(t *testing.T) {
	m := newModelWithThreeFiles(t)
	m.local.cursor = 0 // a.txt

	newModel, _ := m.enterVisual()
	got := newModel.(model)
	if got.mode != ModeVisual {
		t.Fatalf("mode = %v, want ModeVisual", got.mode)
	}

	// extend down twice: cursor moves to c.txt (index 2), range = [0,1,2]
	newModel, _ = got.updateVisual(keyMsg("down"))
	got = newModel.(model)
	newModel, _ = got.updateVisual(keyMsg("down"))
	got = newModel.(model)

	for _, i := range []int{0, 1, 2} {
		if !got.local.isSelected(i) {
			t.Fatalf("expected index %d to be selected in range, selected=%v", i, got.local.selected)
		}
	}

	newModel, _ = got.updateVisual(keyMsg("y"))
	got = newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse after yank", got.mode)
	}
	if got.yank.count() != 3 {
		t.Fatalf("yank.count() = %d, want 3", got.yank.count())
	}
	if len(got.local.selected) != 0 {
		t.Fatalf("expected marks cleared after yank, got %v", got.local.selected)
	}
}

func TestVisualEscRestoresPriorSelection(t *testing.T) {
	m := newModelWithThreeFiles(t)
	m.local.cursor = 0
	newModel, _ := m.toggleMark() // pre-mark a.txt only
	m = newModel.(model)

	m.local.cursor = 1 // enter visual anchored at b.txt
	newModel, _ = m.enterVisual()
	got := newModel.(model)

	newModel, _ = got.updateVisual(keyMsg("down")) // extend to c.txt: marks 1,2 added
	got = newModel.(model)
	if !got.local.isSelected(2) {
		t.Fatal("expected index 2 to be selected mid-range")
	}

	newModel, _ = got.updateVisual(keyMsg("esc"))
	got = newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse", got.mode)
	}
	if !got.local.isSelected(0) || got.local.isSelected(1) || got.local.isSelected(2) {
		t.Fatalf("selected after esc = %v, want only index 0 (the pre-existing mark)", got.local.selected)
	}
}
