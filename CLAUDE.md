# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@SPEC.md

## Commands

```
# dependencies
go mod tidy

# build (current OS)
go build -o bin/scpick ./cmd/scpick

# cross-compile (must stay CGO_ENABLED=0 — no external ssh/scp/fzf binary, no cgo)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/scpick.exe ./cmd/scpick
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -o bin/scpick     ./cmd/scpick

# unit tests (no live SSH server needed)
go test ./...
go test ./internal/tui/... -run TestUpdateFilterNarrowsVisibleEntriesAsYouType -v   # single test

# integration tests (spins up an in-process SSH/SFTP server, see internal/sshtest)
go test -tags=integration ./...

# race detector needs cgo; on Windows without MSVC/mingw, zig works as CC but the
# race runtime currently fails to allocate its shadow memory on this machine —
# prefer running -race in CI (Linux) instead of chasing it locally
CC="zig cc" CGO_ENABLED=1 CGO_LDFLAGS="-lapi-ms-win-core-synch-l1-2-0" go test ./... -race

# lint / format
gofmt -l .
go vet ./...
golangci-lint run ./...
```

The CLI is a single `tea.NewProgram(tui.NewModel(), tea.WithAltScreen()).Run()` call in
`cmd/scpick/main.go` — no flags, no subcommands. Host, remote path, and local path are
all chosen through the always-visible dual-pane TUI; direction (upload vs. download) is
implied by which pane you last yanked from (`y`), never by a flag. Whole-directory
transfer is inherent to yanking a directory (always recursive, via
`transfer.RecursivePull`/`RecursivePush`) — there is no `-r`/`--recursive` flag to ask
about anymore.

## Architecture

Call chain: `cmd/scpick` (starts the bubbletea program, nothing else) →
`internal/tui` (owns all state and all key handling) → `internal/{auth, remotefs,
localfs, sshconf, transfer}` (the actual mechanics, unchanged in spirit from before the
TUI rewrite — see each package's own doc comment for its role).

- **`internal/tui`** is a `mode`-driven `tea.Model` (`model.go`): one `paneState` per
  side (`local`, `remote`), each holding its current directory, listing, cursor
  (indexing into the *visible*, possibly filtered, subset — see `paneState.
  visibleIndices()`), and a `selected map[int]bool` keyed by real entry index so marks
  survive filtering. `browse.go` dispatches Browse-mode keys (`j/k/h/l/Tab/y/p/Space/v/
  /`/`?`/`C`/`q`); `visual.go` and `filter.go` handle the two sub-modes Space/`v` and `/`
  drop into; `hostselect.go` drives host-list/manual-entry/password/host-key-confirm as
  bubbletea modes instead of blocking terminal prompts; `transfer_modal.go` drives the
  batch-wide overwrite confirm and progress display; `view.go` renders everything with
  `lipgloss`; `help.go` renders the `?` screen from the static tables in `keymap.go`;
  `entry.go` holds the unified `paneEntry` type (replacing the old `internal/picker.
  Entry`/`ListItem`) plus the local/remote listing adapters and the `sahilm/fuzzy`
  integration.
- **Async pattern** (`connect.go`, `transfer_modal.go`): SSH handshakes and SFTP
  transfers are blocking I/O that must not freeze bubbletea's `Update` loop, and their
  callbacks (`auth`'s password prompt, `auth.ConfirmFunc`, `transfer.ConfirmOverwrite`,
  `transfer.ProgressPrinter`) fire synchronously inside that blocking call. Both files
  use the same shape: a `tea.Cmd` spawns a **detached goroutine** that does the real
  work and returns immediately (so the `Cmd` itself never blocks bubbletea), while a
  paired `waitForConnectEvent`/`waitForTransferEvent` `tea.Cmd` blocks on a
  `chan tea.Msg` for whatever that goroutine sends next. Every time `Update` receives a
  non-terminal event (`passwordNeededMsg`, `hostKeyNeededMsg`, `transferProgressMsg`) it
  re-issues the wait `Cmd` so the next event gets picked up; a terminal event
  (`connectResultMsg`, `transferDoneMsg`) does not re-arm it. Connect additionally needs
  a live back-and-forth (the goroutine blocks on `<-passwordAnswer`/`<-hostKeyAnswer`
  until `Update` sends the user's reply down that channel from `ModePasswordPrompt`/
  `ModeHostKeyConfirm`); transfer does not, since the batch-wide overwrite decision
  (`o`/`s`/`Esc`, see SPEC.md's Open Questions on this being a single choice for the
  whole paste rather than per-file) is fixed *before* the transfer starts, via a
  `fixedOverwrite(decision)` closure — no round trip needed once the transfer is
  running.
- **`internal/auth`** gained one addition for the TUI: `NewAuthChainWithPassword(
  passwordFunc)` (auth.go), so a caller can supply a password callback that shows a
  masked `bubbles/textinput` modal instead of the original `NewAuthChain()`'s
  `term.ReadPassword` (kept as the default for any future non-TUI caller).
  `auth.HostKeyCallback`'s `ConfirmFunc` was already injectable and needed no change.
- **`internal/transfer`**: `Pull`, `Push`, `transfer.go`'s types (`Result`,
  `OverwriteDecision`, `ConfirmOverwrite`, `ProgressPrinter`, `overwriteGate`) are
  unchanged from before the rewrite — they never depended on the UI. `recursive.go`'s
  `recursivePull`/`recursivePush` were renamed to `RecursivePull`/`RecursivePush`
  (exported) since they're now called from the separate `internal/tui` package instead
  of from same-package orchestration code. `run.go`, `browse.go`, `terminal.go`, and the
  terminal-prompt halves of `defaults.go` (`DefaultConfirmOverwrite`/
  `DefaultProgressPrinter`) are gone — that whole layer was sequential-prompt UI glue
  superseded by `internal/tui`; `formatBytes` is the only survivor from `defaults.go`.
- **`internal/picker`** no longer exists. Its pure list-ordering logic (dirs before
  files, alphabetical within each group) lived entirely in `localfs.ListDir`/
  `remotefs.Client.ListDir` already, so `internal/tui/entry.go` only had to add the
  leading `".."` entry and the `sahilm/fuzzy` filter — the marker-based "pick a
  directory as the target" concept (`★ use this dir` / `★ transfer this directory`)
  doesn't exist in the dual-pane design, since a pane's current directory *is* the
  transfer target, always.
- **`internal/localfs`**, **`internal/remotefs`**, **`internal/sshconf`**,
  **`internal/sshtest`** are unchanged; none of them ever depended on the picker/cobra
  UI layer.

## Testing Strategy

`internal/tui`'s `Update*` functions are unit-tested the same way the rest of the
codebase tests pure logic: construct a `model`, send it a `tea.KeyMsg` (see the
`keyMsg()` helper in `hostselect_test.go`), and assert on the returned `model`'s state
— no real terminal needed, since bubbletea's `Model` is a plain struct. Tests that need
a real directory tree use `t.TempDir()` (see `newModelAt`/`newModelWithThreeFiles`/
`newModelWithNamedFiles` in `browse_test.go`/`visual_test.go`/`filter_test.go`).
`view_test.go`'s `TestViewDoesNotPanicInAnyMode` is a smoke test only — `View()`'s
rendered output is not asserted on beyond "didn't panic, isn't empty", per SPEC.md.

Anything that needs a real `*remotefs.Client` (the connect success path listing the
remote root; `startTransferWorker` actually calling `Pull`/`Push`/`RecursivePull`/
`RecursivePush` against a live SFTP session) is a `-tags=integration` test in
`internal/tui/integration_test.go`, using the same `internal/sshtest` in-process
SSH+SFTP server as `internal/auth`/`internal/remotefs`/`internal/transfer`'s own
integration tests. This split matters here specifically because `dialHost()`
(`connect.go`) resolves `~/.ssh/known_hosts` from the real user's home directory (like
the original `run.go` did) — a unit test must never exercise that path, since it would
read or write the developer's actual `known_hosts` file as a side effect.

**Known gap**: SPEC.md's Boundaries section requires deleting partial destination files
when a transfer is cancelled, but there is currently no way to cancel a transfer once
`ModeTransferProgress` starts (no `Esc` handling, no cancellation channel threaded into
`transfer.Pull`/`Push`/`Download`/`Upload`). This predates the TUI rewrite — the
original terminal-driven flow had no cancellation path either — but it remains
unimplemented and should be picked up as a follow-up.
