// Package picker turns one directory's worth of entries into a list for
// interactive selection, and drives github.com/ktr0731/go-fuzzyfinder to
// present that list. List generation is a pure function of its inputs so it
// can be unit tested without invoking the terminal UI; only the thin
// TUI-invoking functions in ui.go require a real terminal.
package picker

import "sort"

// Entry is one file or directory to offer in a picker list. Path is the
// full path the caller should navigate to or transfer if this entry is
// chosen; picker never joins path segments itself; a local or remote
// filesystem does so, since path syntax differs between Windows, Linux and
// SFTP.
type Entry struct {
	Name  string
	Path  string
	IsDir bool
	Size  int64
}

// ListItem is one row a picker list presents to the user.
type ListItem struct {
	Label    string
	Path     string
	IsDir    bool
	IsMarker bool // true only for the dir-pick "use this dir" entry
}

const (
	parentLabel          = ".."
	useThisDirLabel      = "★ use this dir"
	transferThisDirLabel = "★ transfer this directory"
)

// BuildFileList produces the list for file-pick mode: a leading ".." entry
// pointing at parentDir, then (if recursive) a "transfer this directory"
// marker for currentDir, then directories (alphabetical), then files
// (alphabetical). The marker is only present when recursive is true, so a
// directory can never be picked as a transfer target outside recursive mode.
func BuildFileList(entries []Entry, parentDir, currentDir string, recursive bool) []ListItem {
	items := make([]ListItem, 0, len(entries)+2)
	items = append(items, ListItem{Label: parentLabel, Path: parentDir, IsDir: true})
	if recursive {
		items = append(items, ListItem{Label: transferThisDirLabel, Path: currentDir, IsMarker: true})
	}

	dirs, files := splitAndSort(entries)
	for _, e := range dirs {
		items = append(items, ListItem{Label: e.Name, Path: e.Path, IsDir: true})
	}
	for _, e := range files {
		items = append(items, ListItem{Label: e.Name, Path: e.Path, IsDir: false})
	}
	return items
}

// BuildDirList produces the list for dir-pick mode: a "use this dir" marker
// for currentDir, a ".." entry pointing at parentDir, then subdirectories
// only (files are never valid transfer destinations).
func BuildDirList(entries []Entry, currentDir, parentDir string) []ListItem {
	items := make([]ListItem, 0, len(entries)+2)
	items = append(items, ListItem{Label: useThisDirLabel, Path: currentDir, IsMarker: true})
	items = append(items, ListItem{Label: parentLabel, Path: parentDir, IsDir: true})

	dirs, _ := splitAndSort(entries)
	for _, e := range dirs {
		items = append(items, ListItem{Label: e.Name, Path: e.Path, IsDir: true})
	}
	return items
}

func splitAndSort(entries []Entry) (dirs, files []Entry) {
	for _, e := range entries {
		if e.IsDir {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	sortByName(dirs)
	sortByName(files)
	return dirs, files
}

func sortByName(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
}
