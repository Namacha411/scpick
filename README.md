# scpick

An interactive SCP/SFTP file transfer tool that runs as a single static
binary on both native Windows (PowerShell, no WSL/Git Bash) and Linux, with
no dependency on external `ssh`/`scp`/`fzf` binaries.

Run `scpick pull` or `scpick push` and drive the whole flow with the
keyboard: pick a host, browse the remote and local filesystems one directory
level at a time, and transfer. No path is ever typed by hand.

## Install / Build

Requires Go 1.25+.

```sh
go mod tidy
go build -o bin/scpick ./cmd/scpick
```

Cross-compile for either platform (always `CGO_ENABLED=0` — the binary must
stay dependency-free):

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/scpick.exe ./cmd/scpick
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -o bin/scpick     ./cmd/scpick
```

## Usage

```sh
scpick pull   # remote -> local
scpick push   # local -> remote
```

Both commands are fully interactive; there are no flags to memorize:

1. **Select a host** — every `Host` entry in `~/.ssh/config` is offered, plus
   a manual-entry option (hostname/user/port) for anything not in your config.
2. **Authenticate** — tried in order: ssh-agent (Unix socket, or the Windows
   OpenSSH named pipe), then `~/.ssh/id_ed25519`/`id_rsa`/`id_ecdsa`, then an
   interactive password prompt. Password and passphrase input is masked.
3. **Verify the host key** — checked against `~/.ssh/known_hosts`. An unknown
   host shows its fingerprint and asks whether to trust it; a key that
   doesn't match a known entry always aborts the connection, no prompt.
4. **Browse and select** — the source side supports multi-select (tag
   several files); the destination side lets you drill into subdirectories or
   confirm the current one with "★ use this dir". On Windows, going "up" past
   a drive root shows a list of available drive letters.
5. **Transfer** — a per-file progress bar, a prompt before overwriting an
   existing destination file (with a "yes to all" option for the rest of the
   batch), and a summary at the end. One failed file doesn't abort the rest
   of the batch.

## Development

See `SPEC.md` for the full design spec and `CLAUDE.md` for architecture notes
and common commands (build, test, lint).
