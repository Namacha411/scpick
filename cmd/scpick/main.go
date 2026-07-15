// Command scpick is an interactive dual-pane SCP/SFTP file transfer TUI. See
// SPEC.md for the full design.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"scpick/internal/tui"
)

func main() {
	if _, err := tea.NewProgram(tui.NewModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "scpick: %v\n", err)
		os.Exit(1)
	}
}
