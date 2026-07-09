//go:build !windows

package localfs

import "fmt"

// ListDrives is not meaningful outside Windows.
func ListDrives() ([]Entry, error) {
	return nil, fmt.Errorf("localfs: list drives: not supported on this platform")
}

func isDriveRoot(path string) bool {
	return false
}
