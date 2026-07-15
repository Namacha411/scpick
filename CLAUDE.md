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

There are no flags or subcommands: `cmd/scpick/main.go` just runs
`tea.NewProgram(tui.NewModel(), tea.WithAltScreen())`. Everything — host, remote path,
local path, upload vs. download — is chosen through the dual-pane TUI.

## Architecture

Call chain: `cmd/scpick` (starts the program, nothing else) → `internal/tui` (all state
and key handling) → `internal/{auth, remotefs, localfs, sshconf, transfer}` (the
mechanics, unchanged in spirit from before the TUI rewrite).

- **`internal/tui`**: one mode-driven `tea.Model` (`model.go`). `browse.go` handles
  normal-mode keys; `visual.go`/`filter.go` handle the `v`/`/` sub-modes; `hostselect.go`
  drives host-list/manual-entry/password/host-key-confirm as modes instead of blocking
  terminal prompts; `transfer_modal.go` drives the overwrite confirm + progress;
  `connect.go` does the actual SSH connect; `view.go` renders with `lipgloss`;
  `entry.go` holds the `paneEntry` type, listing adapters, and `sahilm/fuzzy` filter;
  `help.go`/`keymap.go` back the `?` screen.
- **Async I/O**: SSH connect and SFTP transfer block, so `connect.go`/`transfer_modal.go`
  run them in a detached goroutine behind a `tea.Cmd`, paired with a
  `waitForConnectEvent`/`waitForTransferEvent` `tea.Cmd` that blocks on a `chan tea.Msg`
  and re-arms itself until a terminal message (`connectResultMsg`/`transferDoneMsg`)
  arrives. Connect additionally round-trips through `passwordAnswer`/`hostKeyAnswer`
  channels so `ModePasswordPrompt`/`ModeHostKeyConfirm` can reply mid-connect.
- **`internal/auth`/`remotefs`/`localfs`/`sshconf`**: unchanged from before the rewrite;
  none of them ever depended on a UI. `auth.NewAuthChainWithPassword` lets the TUI
  supply a masked-input callback instead of reading the terminal directly.
- **`internal/transfer`**: `Pull`/`Push`/`RecursivePull`/`RecursivePush` and the
  overwrite/result types are unchanged pure logic (`RecursivePull`/`RecursivePush` were
  exported so `internal/tui` can call them from outside the package). The old
  sequential-prompt UI glue (`run.go`, `browse.go`, `terminal.go`, `internal/picker`) is
  gone, replaced entirely by `internal/tui`.

## Testing Strategy

- `internal/tui`'s `Update*` functions: build a `model`, send it a `tea.KeyMsg`, assert
  on the result. No terminal needed.
- Anything needing a real `*remotefs.Client` (the connect success path, transfer worker
  tests) is a `-tags=integration` test using `internal/sshtest`'s in-process SSH/SFTP
  server — kept separate because `dialHost()` touches the real `~/.ssh/known_hosts`.
- `TestViewDoesNotPanicInAnyMode` is a smoke test only; `View()`'s rendered output isn't
  otherwise asserted on.

**Known gap**: cancelling an in-flight transfer (and deleting the partial destination
file) isn't implemented — `ModeTransferProgress` has no `Esc` handling, and
`transfer.Pull`/`Push` take no cancellation signal. This predates the TUI rewrite and
remains a good follow-up.
