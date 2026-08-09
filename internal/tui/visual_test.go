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

func TestToggleMarkOnYankedEntryRemovesItFromYankBuffer(t *testing.T) {
	m := newModelWithThreeFiles(t)
	m.local.cursor = 0 // a.txt

	newModel, _ := m.yankSelection()
	got := newModel.(model)
	aPath := joinSourcePath(0, got.local.path, "a.txt")
	if !got.yank.has(0, aPath) {
		t.Fatal("expected a.txt to be in the yank buffer after yanking")
	}

	newModel, _ = got.toggleMark()
	got = newModel.(model)
	if got.yank.has(0, aPath) {
		t.Fatal("expected Space to remove a.txt from the yank buffer")
	}
	if got.local.isSelected(0) {
		t.Fatal("expected a.txt not to become marked when un-yanked by Space")
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
	for _, i := range []int{0, 1, 2} {
		if !got.local.isSelected(i) {
			t.Fatalf("expected index %d to stay marked after yank, selected=%v", i, got.local.selected)
		}
	}
}

// markAndYankAll marks every non-".." entry in m.local (assumed to be
// indices 0..n-1) and yanks them in one shot, so all of them start out both
// marked and sitting in the yank buffer.
func markAndYankAll(t *testing.T, m model, n int) model {
	t.Helper()
	for i := range n {
		m.local.cursor = i
		newModel, _ := m.toggleMark()
		m = newModel.(model)
	}
	newModel, _ := m.yankSelection()
	return newModel.(model)
}

func TestEnterVisualOnActiveAnchorDeactivatesRangeAndRestoresOnShrink(t *testing.T) {
	m := newModelWithThreeFiles(t)
	m = markAndYankAll(t, m, 3) // 0,1,2 all marked + yanked
	paths := []string{
		joinSourcePath(0, m.local.path, "a.txt"),
		joinSourcePath(0, m.local.path, "b.txt"),
		joinSourcePath(0, m.local.path, "c.txt"),
	}

	m.local.cursor = 0 // anchor on an already-active entry: direction = deactivate
	newModel, _ := m.enterVisual()
	got := newModel.(model)

	newModel, _ = got.updateVisual(keyMsg("down")) // range [0,1]: both deactivated
	got = newModel.(model)
	for i, idx := range []int{0, 1} {
		if got.local.isSelected(idx) {
			t.Fatalf("expected index %d to be unmarked, selected=%v", idx, got.local.selected)
		}
		if got.yank.has(0, paths[i]) {
			t.Fatalf("expected index %d to be un-yanked, yank=%+v", idx, got.yank)
		}
	}
	if !got.local.isSelected(2) || !got.yank.has(0, paths[2]) {
		t.Fatal("expected index 2 (outside the range) to stay marked and yanked")
	}

	newModel, _ = got.updateVisual(keyMsg("up")) // shrink range back to [0,0]
	got = newModel.(model)
	if got.local.isSelected(0) || got.yank.has(0, paths[0]) {
		t.Fatal("expected index 0 (still in range) to stay deactivated")
	}
	if !got.local.isSelected(1) || !got.yank.has(0, paths[1]) {
		t.Fatalf("expected index 1 to be restored to marked+yanked after shrinking the range past it, selected=%v yank=%+v", got.local.selected, got.yank)
	}
}

func TestVisualEscRestoresPriorYankBuffer(t *testing.T) {
	m := newModelWithThreeFiles(t)
	m = markAndYankAll(t, m, 2) // 0,1 marked + yanked
	aPath := joinSourcePath(0, m.local.path, "a.txt")
	bPath := joinSourcePath(0, m.local.path, "b.txt")

	m.local.cursor = 0 // anchor on an active entry: direction = deactivate
	newModel, _ := m.enterVisual()
	got := newModel.(model)

	newModel, _ = got.updateVisual(keyMsg("down")) // range [0,1]: both un-yanked
	got = newModel.(model)
	if got.yank.has(0, aPath) || got.yank.has(0, bPath) {
		t.Fatal("expected both entries to be un-yanked while extending the range")
	}

	newModel, _ = got.updateVisual(keyMsg("esc"))
	got = newModel.(model)
	if !got.yank.has(0, aPath) || !got.yank.has(0, bPath) {
		t.Fatalf("expected Esc to restore the yank buffer to its pre-visual state, yank=%+v", got.yank)
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
