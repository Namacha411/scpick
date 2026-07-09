// Package localfs provides one-level-at-a-time local filesystem browsing,
// absorbing Windows/Linux path differences (drive letters vs a single root).
package localfs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DrivesMarker is a sentinel path representing "list of available drives".
// It is only meaningful on Windows; ListDir delegates to ListDrives when
// given this path.
const DrivesMarker = "<drives>"

// Entry is one file or directory returned by ListDir.
type Entry struct {
	Name  string
	IsDir bool
	Size  int64
}

// ListDir returns the contents of path, sorted with directories first, then
// alphabetically within each group. It does not include a parent ("..")
// entry; callers (internal/picker) are responsible for that.
func ListDir(path string) ([]Entry, error) {
	if path == DrivesMarker {
		return ListDrives()
	}
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("localfs: list %q: %w", path, err)
	}
	entries := make([]Entry, 0, len(dirEntries))
	for _, de := range dirEntries {
		info, err := de.Info()
		if err != nil {
			// Broken symlink or race with a concurrent delete; skip rather
			// than fail the whole listing.
			continue
		}
		entries = append(entries, Entry{Name: de.Name(), IsDir: de.IsDir(), Size: info.Size()})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

// GetParentDir returns the parent directory of path. On Windows, the parent
// of a drive root (e.g. C:\) is DrivesMarker. On Linux, the parent of "/" is
// "/" itself.
func GetParentDir(path string) string {
	if path == DrivesMarker {
		return DrivesMarker
	}
	if isDriveRoot(path) {
		return DrivesMarker
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path
	}
	return parent
}

// JoinPath joins parent and child, handling the DrivesMarker sentinel so
// that selecting a drive letter (e.g. "C:") from the drive list produces a
// usable root path (e.g. "C:\").
func JoinPath(parent, child string) string {
	if parent == DrivesMarker {
		return child + string(filepath.Separator)
	}
	return filepath.Join(parent, child)
}

// ExpandHome expands a leading "~" to the current user's home directory.
func ExpandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("localfs: expand home: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}
