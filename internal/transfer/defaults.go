package transfer

import (
	"fmt"
	"os"
	"strings"
)

// DefaultConfirmOverwrite prompts on the terminal: "Overwrite <path>?
// [y/N/a(ll)]". Not covered by automated tests (reads a real terminal);
// the overwrite state machine itself (internal/transfer.overwriteGate) is
// unit tested with a fake ConfirmOverwrite.
func DefaultConfirmOverwrite(destPath string, existingSize, newSize int64) OverwriteDecision {
	prompt := fmt.Sprintf("%s already exists (%s -> %s). Overwrite? [y/N/a(ll)] ",
		destPath, formatBytes(existingSize), formatBytes(newSize))

	line, _ := readLine(prompt)
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return OverwriteYes
	case "a", "all":
		return OverwriteAll
	default:
		return OverwriteSkip
	}
}

// DefaultProgressPrinter renders an in-place, single-line progress bar to
// stderr for the file currently transferring.
func DefaultProgressPrinter(label string, done, total int64) {
	const width = 20
	filled := 0
	if total > 0 {
		filled = int(float64(width) * float64(done) / float64(total))
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
	fmt.Fprintf(os.Stderr, "\r%s [%s] %s/%s", label, bar, formatBytes(done), formatBytes(total))
	if total > 0 && done >= total {
		fmt.Fprintln(os.Stderr)
	}
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
