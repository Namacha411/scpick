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
go test ./internal/picker/... -run TestBuildDirList -v   # single test

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

# run interactively
go run ./cmd/scpick pull
go run ./cmd/scpick push
```

The CLI is built on `github.com/spf13/cobra` (`cmd/scpick/main.go`), which
provides `--help` on the root command and on `pull`/`push` for free. Host,
remote path, and local path are still chosen entirely through the
interactive picker, one directory level at a time — there are no path
arguments. The one flag that exists is `-r`/`--recursive` on both `pull` and
`push`, enabling whole-directory transfer (`scp -r` equivalent): when set,
the file browser (`internal/picker.BuildFileList`) offers a
"★ transfer this directory" marker at each level, letting the user pick the
current directory as a whole instead of only navigating into it or picking
individual files. Without `-r`, that marker is never shown, so a directory
can never reach `Pull`/`Push` — the recursive-transfer code path
(`internal/transfer/recursive.go`) is only ever exercised when the flag is
set. Changing subcommand names, or adding flags beyond this, still needs to
be asked about first per SPEC.md's Boundaries.

## Architecture

Call chain: `cmd/scpick` (routing only) → `internal/transfer` (assembles
everything: host pick → auth → connect → browse → transfer) → `internal/{sshconf,
auth, remotefs, localfs, picker}` (the actual mechanics).

- **`internal/transfer`** is the orchestrator, not a thin package. `run.go` has
  `RunPull`/`RunPush` (the real entry points cmd/scpick calls), host selection
  (`~/.ssh/config` list + manual entry), and known_hosts wiring. `browse.go` has
  the repeat-list-then-pick navigation loop shared by both remote and local
  browsing — it adapts `localfs.ListDir` and `*remotefs.Client.ListDir` to a
  common `picker.Entry` shape so one loop implementation drives both. `pull.go`/
  `push.go` do the actual per-file transfer loop: overwrite confirmation (with a
  "yes to all" `overwriteGate` that persists across the batch) and progress
  reporting, never aborting the whole batch on one file's error — failures
  accumulate in `Result.Failed` and the loop continues. `recursive.go` builds
  whole-directory transfer on top of `Pull`/`Push` rather than changing their
  loops: it walks a directory tree one level at a time (via `ListDir`/
  `localfs.ListDir`), creates the matching destination directory at each level
  (`os.MkdirAll` locally, the new `remoteFile.MkdirAll` remotely), calls the
  existing `Pull`/`Push` for that level's files, and merges every level's
  `Result` together — sharing one `overwriteGate`-wrapped confirm closure
  across the whole tree so "yes to all" persists across subdirectories, not
  just within one `Pull`/`Push` call.
- **`internal/picker`** is deliberately split in two: `picker.go` has pure,
  fully unit-tested list-generation (`BuildFileList`/`BuildDirList` — dirs
  before files, a `..` entry, and a "★ use this dir" marker for directory-pick
  mode; `BuildFileList` also takes a `recursive bool` that, when true, adds a
  "★ transfer this directory" marker for picking a whole directory as a
  transfer target), and `ui.go` is a thin, untested shim around go-fuzzyfinder
  (`PickFiles`/`PickOne`). Never merge TUI-invoking code into the pure list
  builders — that split is what makes this package testable without a
  terminal.
- **`internal/auth`**'s chain is agent-then-key-files-then-password, but
  implemented as exactly two `ssh.AuthMethod`s (not three): a single
  `ssh.PublicKeysCallback` that tries agent signers first and only falls back
  to loading (and prompting passphrases for) local key files if the agent
  produced zero signers, followed by a `ssh.PasswordCallback`. This is
  deliberate — `x/crypto/ssh` dedupes auth methods by RFC method name
  ("publickey"/"password"), so agent and key-file logic must live inside one
  callback, not as separate AuthMethod entries. ssh-agent connection is
  platform-split: `agent_unix.go` dials `SSH_AUTH_SOCK`, `agent_windows.go`
  dials the OpenSSH Windows named pipe via `github.com/Microsoft/go-winio`
  (Pageant is not supported). `known_hosts.go` wraps `knownhosts.New`'s
  callback to distinguish "unknown host" (prompts via an injectable
  `ConfirmFunc`, then appends to the file) from a genuine mismatch (always
  aborts, never prompts).
- **`internal/localfs`** hides the Windows/Linux root difference behind a
  `DrivesMarker` sentinel path (`"<drives>"`): `GetParentDir` of a Windows
  drive root returns the marker, and `ListDir(DrivesMarker)` lists available
  drive letters instead of a real directory. Platform-specific bits
  (`ListDrives`, `isDriveRoot`) live in `windows.go`/`unix.go` behind build
  tags.
- **`internal/remotefs`** wraps `pkg/sftp`; `remoteFile` in
  `internal/transfer/transfer.go` is a narrow interface over it (`ListDir`,
  `Stat`, `Download`, `Upload`) so `Pull`/`Push` are unit-testable against a
  fake without a live server — see `fake_client_test.go`.
- **`internal/sshtest`** is a real (non-`_test.go`) package containing an
  in-process SSH+SFTP server (`golang.org/x/crypto/ssh` server + `pkg/sftp`'s
  `InMemHandler`), used only by `-tags=integration` tests in `auth`,
  `remotefs`, and `transfer`. It's never imported by non-test code, so it
  never ends up in the shipped binary despite not being test-tagged itself.
- Terminal I/O gotcha (see `internal/transfer/terminal.go`): never read a
  plain prompt via `bufio.Reader` over `os.Stdin` after a `picker.Pick*` call.
  go-fuzzyfinder puts the terminal in raw mode, and depending on platform that
  isn't reliably restored to a normal cooked/echoing state afterward — the
  prompt silently drops keystrokes. Every interactive text prompt (manual host
  entry, host-key trust, overwrite confirmation) goes through `readLine()`,
  which explicitly does `term.MakeRaw` → `term.NewTerminal(...).ReadLine()` →
  `term.Restore` itself instead of trusting inherited terminal state.

## Testing Strategy

Coverage is intentionally uneven by design, not by neglect: pure logic
(`sshconf` config parsing, `picker`'s list builders, `transfer`'s
`Pull`/`Push`/`overwriteGate`) is unit-tested at or near 100%, while TUI-
driving code (`picker/ui.go`, `transfer/browse.go`'s navigation loops) and raw
terminal I/O (`transfer/terminal.go`, `transfer/defaults.go`,
`auth`'s `readPassword`/`dialAgent`) sit at 0% and are exercised manually
instead. Don't chase coverage numbers on those files — SPEC.md sets no
numeric target for the TUI/IO boundary.

Never commit private keys, passwords, or test credentials to the repo — this
holds even for test fixtures. Tests that need a private key generate one at
runtime in a `t.TempDir()` (see `internal/auth/keyfile_test.go`'s
`genTestKeyPEM`); nothing under `testdata/` is a real credential.
