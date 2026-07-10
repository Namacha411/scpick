package transfer

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"scpick/internal/localfs"
)

// recursivePull downloads remoteFiles (as Pull does) plus, for each entry
// in remoteDirs, the whole directory tree rooted there, preserving
// structure under localDestDir. A single overwriteGate is shared across
// every file and every directory level, so an "overwrite all" answer
// persists for the whole batch, not just one directory.
func recursivePull(client remoteFile, remoteFiles, remoteDirs []string, localDestDir string, confirm ConfirmOverwrite, progress ProgressPrinter) Result {
	result := Result{Failed: make(map[string]error)}
	gate := &overwriteGate{}
	wrappedConfirm := func(destPath string, existingSize, newSize int64) OverwriteDecision {
		return gate.decide(confirm, destPath, existingSize, newSize)
	}

	if len(remoteFiles) > 0 {
		mergeResults(&result, Pull(client, remoteFiles, localDestDir, wrappedConfirm, progress))
	}
	for _, remoteDir := range remoteDirs {
		walkAndPullDir(client, remoteDir, localDestDir, wrappedConfirm, progress, &result)
	}
	return result
}

// recursivePush mirrors recursivePull for uploads.
func recursivePush(client remoteFile, localFiles, localDirs []string, remoteDestDir string, confirm ConfirmOverwrite, progress ProgressPrinter) Result {
	result := Result{Failed: make(map[string]error)}
	gate := &overwriteGate{}
	wrappedConfirm := func(destPath string, existingSize, newSize int64) OverwriteDecision {
		return gate.decide(confirm, destPath, existingSize, newSize)
	}

	if len(localFiles) > 0 {
		mergeResults(&result, Push(client, localFiles, remoteDestDir, wrappedConfirm, progress))
	}
	for _, localDir := range localDirs {
		walkAndPushDir(client, localDir, remoteDestDir, wrappedConfirm, progress, &result)
	}
	return result
}

// walkAndPullDir creates the local directory corresponding to remoteDir
// (named after remoteDir's own basename, under localParentDir), pulls the
// files at this level into it, and recurses into subdirectories.
func walkAndPullDir(client remoteFile, remoteDir, localParentDir string, confirm ConfirmOverwrite, progress ProgressPrinter, result *Result) {
	localDir := filepath.Join(localParentDir, path.Base(remoteDir))
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		result.Failed[remoteDir] = fmt.Errorf("transfer: pull %q: %w", remoteDir, err)
		return
	}

	entries, err := client.ListDir(remoteDir)
	if err != nil {
		result.Failed[remoteDir] = fmt.Errorf("transfer: pull %q: %w", remoteDir, err)
		return
	}

	var files, subDirs []string
	for _, e := range entries {
		full := path.Join(remoteDir, e.Name)
		if e.IsDir {
			subDirs = append(subDirs, full)
		} else {
			files = append(files, full)
		}
	}

	if len(files) > 0 {
		mergeResults(result, Pull(client, files, localDir, confirm, progress))
	}
	for _, subDir := range subDirs {
		walkAndPullDir(client, subDir, localDir, confirm, progress, result)
	}
}

// walkAndPushDir creates the remote directory corresponding to localDir
// (named after localDir's own basename, under remoteParentDir), pushes the
// files at this level into it, and recurses into subdirectories.
func walkAndPushDir(client remoteFile, localDir, remoteParentDir string, confirm ConfirmOverwrite, progress ProgressPrinter, result *Result) {
	remoteDir := path.Join(remoteParentDir, filepath.Base(localDir))
	if err := client.MkdirAll(remoteDir); err != nil {
		result.Failed[localDir] = fmt.Errorf("transfer: push %q: %w", localDir, err)
		return
	}

	entries, err := localfs.ListDir(localDir)
	if err != nil {
		result.Failed[localDir] = fmt.Errorf("transfer: push %q: %w", localDir, err)
		return
	}

	var files, subDirs []string
	for _, e := range entries {
		full := filepath.Join(localDir, e.Name)
		if e.IsDir {
			subDirs = append(subDirs, full)
		} else {
			files = append(files, full)
		}
	}

	if len(files) > 0 {
		mergeResults(result, Push(client, files, remoteDir, confirm, progress))
	}
	for _, subDir := range subDirs {
		walkAndPushDir(client, subDir, remoteDir, confirm, progress, result)
	}
}

// mergeResults appends src's outcome onto dest.
func mergeResults(dest *Result, src Result) {
	dest.Succeeded = append(dest.Succeeded, src.Succeeded...)
	dest.Skipped = append(dest.Skipped, src.Skipped...)
	for path, err := range src.Failed {
		dest.Failed[path] = err
	}
}
