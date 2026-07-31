// Command scpick is an interactive dual-pane SCP/SFTP file transfer TUI. See
// SPEC.md for the full design.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"scpick/internal/tui"
)

const usageIntro = `scpick is an interactive dual-pane SCP/SFTP file transfer TUI — local files
on the left, remote on the right. Pick a file in one pane, yank it (y), move
to the other pane, and paste (p) to transfer it; direction (upload vs.
download) follows whichever pane you yanked from.

Usage:
  scpick

There are no subcommands; everything happens inside the TUI. The keybinding
reference below is the same one shown by pressing ? once it's running.

Flags:
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scpick", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version information and exit")
	fs.Usage = func() {
		fmt.Fprint(stderr, usageIntro)
		fs.PrintDefaults()
		fmt.Fprint(stderr, "\n")
		fmt.Fprint(stderr, keyBindingReference())
	}

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		fs.Usage()
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, versionString())
		return 0
	}

	if _, err := tea.NewProgram(tui.NewModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(stderr, "scpick: %v\n", err)
		return 1
	}
	return 0
}

// keyBindingReference renders tui.HelpGroups — the same data backing the
// in-TUI `?` screen — as plain text for `scpick --help`.
func keyBindingReference() string {
	var b strings.Builder
	for i, group := range tui.HelpGroups {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(group.Title)
		b.WriteString("\n")

		width := 0
		for _, kb := range group.Bindings {
			if len(kb.Keys) > width {
				width = len(kb.Keys)
			}
		}
		for _, kb := range group.Bindings {
			fmt.Fprintf(&b, "  %-*s  %s\n", width, kb.Keys, kb.Desc)
		}
	}
	return b.String()
}

// versionString reports the commit and build time embedded by the Go
// toolchain's automatic VCS stamping (see `go help buildvcs`), since the
// repo's tags aren't valid semver and can't drive debug.BuildInfo.Main.Version.
func versionString() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "scpick (unknown build)"
	}

	revision, buildTime, dirty := "unknown", "unknown", false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.time":
			buildTime = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	if dirty {
		revision += "-dirty"
	}
	return fmt.Sprintf("scpick %s (built %s)", revision, buildTime)
}
