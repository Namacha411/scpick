# Contributing to scpick

Thanks for considering a contribution! Bug reports, feature requests, and
pull requests are all welcome.

## Getting started

Requires Go 1.25+.

```sh
go mod tidy
go build -o bin/scpick ./cmd/scpick
```

Cross-compiling must stay dependency-free (`CGO_ENABLED=0`, no cgo, no
external `ssh`/`scp`/`fzf` binary):

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/scpick.exe ./cmd/scpick
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -o bin/scpick     ./cmd/scpick
```

See `SPEC.md` for the full design spec and `CLAUDE.md` for an architecture
tour of the codebase.

## Before opening a PR

```sh
gofmt -l .          # must report nothing
go vet ./...
go test ./...                    # unit tests, no live SSH server needed
go test -tags=integration ./...  # spins up an in-process SSH/SFTP server
golangci-lint run ./...          # if you have it installed
```

- Work on a branch, not `main`; keep PRs focused on one change.
- Match the existing style: `gofmt`-clean, errors wrapped rather than
  swallowed, `bubbletea` `Update` logic split per mode (see `internal/tui`).
- Add or update tests alongside behavior changes — see "Testing Strategy" in
  `CLAUDE.md` for where each kind of test belongs.
- Core keybindings (`y`/`p`/`v`/`/`/`Tab`, etc.), new external dependencies,
  and support for protocols beyond SCP/SFTP are design decisions — please
  open an issue to discuss before investing time in an implementation.

## Reporting bugs

Please include your OS, `scpick` version (or commit hash), and steps to
reproduce. If it's transfer- or connection-related, mention the SSH
server/version if known.
