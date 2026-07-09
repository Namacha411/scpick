package transfer

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

// Pull downloads each remote file in remoteFiles into localDestDir. It
// never aborts the whole batch on one file's failure: every error is
// recorded in Result.Failed and the loop continues to the next file.
func Pull(client remoteFile, remoteFiles []string, localDestDir string, confirm ConfirmOverwrite, progress ProgressPrinter) Result {
	result := Result{Failed: make(map[string]error)}
	gate := &overwriteGate{}

	for _, remotePath := range remoteFiles {
		name := path.Base(remotePath)
		localPath := filepath.Join(localDestDir, name)

		info, err := client.Stat(remotePath)
		if err != nil {
			result.Failed[remotePath] = fmt.Errorf("transfer: pull %q: %w", remotePath, err)
			continue
		}

		if existing, err := os.Stat(localPath); err == nil {
			if gate.decide(confirm, localPath, existing.Size(), info.Size) == OverwriteSkip {
				result.Skipped = append(result.Skipped, remotePath)
				continue
			}
		}

		err = client.Download(remotePath, localPath, func(done, total int64) {
			if progress != nil {
				progress(name, done, total)
			}
		})
		if err != nil {
			result.Failed[remotePath] = fmt.Errorf("transfer: pull %q: %w", remotePath, err)
			continue
		}
		result.Succeeded = append(result.Succeeded, remotePath)
	}
	return result
}
