package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWindowSizeMsgUpdatesModel(t *testing.T) {
	m := NewModel()
	newModel, cmd := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	got := newModel.(model)
	if got.termWidth != 200 || got.termHeight != 50 {
		t.Fatalf("termWidth/Height = %d/%d, want 200/50", got.termWidth, got.termHeight)
	}
	if cmd != nil {
		t.Fatal("expected no command from a WindowSizeMsg")
	}
}

func TestPaneWidthFillsTerminalWidth(t *testing.T) {
	m := NewModel()
	m.termWidth = 200
	if w := m.paneWidth(); w != 200/2-paneChromeWidth {
		t.Fatalf("paneWidth() = %d, want %d", w, 200/2-paneChromeWidth)
	}
}

func TestPaneWidthHasAFloor(t *testing.T) {
	m := NewModel()
	m.termWidth = 10 // absurdly narrow
	if w := m.paneWidth(); w != minPaneWidth {
		t.Fatalf("paneWidth() = %d, want floor of %d", w, minPaneWidth)
	}
}

func TestPaneContentHeightFillsTerminalHeight(t *testing.T) {
	m := NewModel()
	m.termHeight = 50
	want := 50 - paneChromeHeight - outsideChromeHeight
	if h := m.paneContentHeight(); h != want {
		t.Fatalf("paneContentHeight() = %d, want %d", h, want)
	}
}

func TestVisibleWindowKeepsCursorInView(t *testing.T) {
	start, end := visibleWindow(50, 100, 10)
	if start > 50 || end <= 50 {
		t.Fatalf("visibleWindow(50, 100, 10) = [%d, %d), cursor 50 not inside", start, end)
	}
	if end-start != 10 {
		t.Fatalf("window size = %d, want 10", end-start)
	}
}

func TestVisibleWindowShowsEverythingWhenItFits(t *testing.T) {
	start, end := visibleWindow(2, 5, 10)
	if start != 0 || end != 5 {
		t.Fatalf("visibleWindow(2, 5, 10) = [%d, %d), want [0, 5)", start, end)
	}
}

func TestTruncateNameShortensWithEllipsis(t *testing.T) {
	got := truncateName("a_very_long_filename.txt", 10)
	if len([]rune(got)) != 10 {
		t.Fatalf("truncateName length = %d, want 10", len([]rune(got)))
	}
	if got[len(got)-len("…"):] != "…" {
		t.Fatalf("truncateName = %q, want to end with an ellipsis", got)
	}
}

func TestTruncateNameLeavesShortNamesAlone(t *testing.T) {
	if got := truncateName("short.txt", 20); got != "short.txt" {
		t.Fatalf("truncateName = %q, want unchanged", got)
	}
}
