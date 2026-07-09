// Package transfer orchestrates the pull/push flow: browsing remote and
// local filesystems via internal/picker, confirming overwrites, and driving
// internal/remotefs transfers with progress reporting.
package transfer

import "scpick/internal/remotefs"

// remoteFile is the subset of *remotefs.Client that transfer depends on.
// Depending on this interface rather than the concrete type lets Pull/Push
// be unit tested against a fake, without a live SSH/SFTP server.
type remoteFile interface {
	ListDir(path string) ([]remotefs.Entry, error)
	Stat(path string) (remotefs.Entry, error)
	Download(remotePath, localPath string, onProgress remotefs.ProgressFunc) error
	Upload(localPath, remotePath string, onProgress remotefs.ProgressFunc) error
}

// Result is the outcome of a Pull or Push call over a batch of files.
type Result struct {
	Succeeded []string
	Skipped   []string // destination existed and the user declined to overwrite
	Failed    map[string]error
}

// OverwriteDecision is the user's answer when a destination file already
// exists.
type OverwriteDecision int

const (
	OverwriteSkip OverwriteDecision = iota
	OverwriteYes
	OverwriteAll // yes to this file and every remaining file in the batch
)

// ConfirmOverwrite is asked what to do when destPath already exists.
type ConfirmOverwrite func(destPath string, existingSize, newSize int64) OverwriteDecision

// ProgressPrinter is called during each file's transfer with the file's
// label (its base name) and the bytes copied so far vs. its total size.
type ProgressPrinter func(label string, done, total int64)

// overwriteGate applies a "yes to all" decision across an entire batch:
// once the user picks OverwriteAll, confirm is never consulted again for
// the rest of the batch.
type overwriteGate struct {
	allAccepted bool
}

func (g *overwriteGate) decide(confirm ConfirmOverwrite, destPath string, existingSize, newSize int64) OverwriteDecision {
	if g.allAccepted {
		return OverwriteYes
	}
	decision := confirm(destPath, existingSize, newSize)
	if decision == OverwriteAll {
		g.allAccepted = true
		return OverwriteYes
	}
	return decision
}
