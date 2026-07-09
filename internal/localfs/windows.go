//go:build windows

package localfs

import (
	"os"
	"path/filepath"
)

// ListDrives returns the letters of all mounted drives (C:, D:, ...) as
// pseudo-directory entries.
func ListDrives() ([]Entry, error) {
	var entries []Entry
	for c := 'A'; c <= 'Z'; c++ {
		drive := string(c) + ":"
		if _, err := os.Stat(drive + `\`); err == nil {
			entries = append(entries, Entry{Name: drive, IsDir: true})
		}
	}
	return entries, nil
}

func isDriveRoot(path string) bool {
	clean := filepath.Clean(path)
	return len(clean) == 3 && clean[1] == ':' && clean[2] == '\\'
}
