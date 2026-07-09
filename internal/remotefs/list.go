package remotefs

import (
	"fmt"
	"sort"
)

// ListDir returns the contents of a remote directory, sorted with
// directories first, then alphabetically within each group. It does not
// include a parent ("..") entry; callers (internal/picker) are responsible
// for that.
func (c *Client) ListDir(path string) ([]Entry, error) {
	infos, err := c.sftp.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("remotefs: list %q: %w", path, err)
	}
	entries := make([]Entry, 0, len(infos))
	for _, fi := range infos {
		entries = append(entries, Entry{Name: fi.Name(), IsDir: fi.IsDir(), Size: fi.Size()})
	}
	sortEntries(entries)
	return entries, nil
}

// Stat returns metadata for a single remote path.
func (c *Client) Stat(path string) (Entry, error) {
	fi, err := c.sftp.Stat(path)
	if err != nil {
		return Entry{}, fmt.Errorf("remotefs: stat %q: %w", path, err)
	}
	return Entry{Name: fi.Name(), IsDir: fi.IsDir(), Size: fi.Size()}, nil
}

func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
}
