package transfer

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

// Push uploads each local file in localFiles into remoteDestDir. It never
// aborts the whole batch on one file's failure: every error is recorded in
// Result.Failed and the loop continues to the next file.
func Push(client remoteFile, localFiles []string, remoteDestDir string, confirm ConfirmOverwrite, progress ProgressPrinter) Result {
	result := Result{Failed: make(map[string]error)}
	gate := &overwriteGate{}

	for _, localPath := range localFiles {
		name := filepath.Base(localPath)
		remotePath := path.Join(remoteDestDir, name)

		info, err := os.Stat(localPath)
		if err != nil {
			result.Failed[localPath] = fmt.Errorf("transfer: push %q: %w", localPath, err)
			continue
		}

		if existing, err := client.Stat(remotePath); err == nil {
			if gate.decide(confirm, remotePath, existing.Size, info.Size()) == OverwriteSkip {
				result.Skipped = append(result.Skipped, localPath)
				continue
			}
		}

		err = client.Upload(localPath, remotePath, func(done, total int64) {
			if progress != nil {
				progress(name, done, total)
			}
		})
		if err != nil {
			result.Failed[localPath] = fmt.Errorf("transfer: push %q: %w", localPath, err)
			continue
		}
		result.Succeeded = append(result.Succeeded, localPath)
	}
	return result
}
