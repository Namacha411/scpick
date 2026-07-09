package transfer

import (
	"fmt"
	"path"

	"scpick/internal/localfs"
	"scpick/internal/picker"
)

type listFunc func(dir string) ([]picker.Entry, error)
type parentFunc func(dir string) string

func localListFunc(dir string) ([]picker.Entry, error) {
	entries, err := localfs.ListDir(dir)
	if err != nil {
		return nil, err
	}
	items := make([]picker.Entry, len(entries))
	for i, e := range entries {
		items[i] = picker.Entry{Name: e.Name, Path: localfs.JoinPath(dir, e.Name), IsDir: e.IsDir, Size: e.Size}
	}
	return items, nil
}

func remoteListFunc(client remoteFile) listFunc {
	return func(dir string) ([]picker.Entry, error) {
		entries, err := client.ListDir(dir)
		if err != nil {
			return nil, err
		}
		items := make([]picker.Entry, len(entries))
		for i, e := range entries {
			items[i] = picker.Entry{Name: e.Name, Path: path.Join(dir, e.Name), IsDir: e.IsDir, Size: e.Size}
		}
		return items, nil
	}
}

// remoteParent computes the parent of a remote (always POSIX-style) path;
// the root's parent is itself, same as on Linux.
func remoteParent(dir string) string {
	if dir == "/" {
		return "/"
	}
	return path.Dir(dir)
}

// BrowseRemoteFiles lets the user navigate the remote filesystem starting
// at startDir and select one or more files to pull. Not covered by
// automated tests (drives the picker TUI); verify manually per SPEC.md.
func BrowseRemoteFiles(client remoteFile, startDir string) ([]string, error) {
	items, err := browseFiles(startDir, remoteListFunc(client), remoteParent)
	if err != nil {
		return nil, err
	}
	return itemPaths(items), nil
}

// BrowseLocalFiles lets the user navigate the local filesystem starting at
// startDir and select one or more files to push. Not covered by automated
// tests (drives the picker TUI); verify manually per SPEC.md.
func BrowseLocalFiles(startDir string) ([]string, error) {
	items, err := browseFiles(startDir, localListFunc, localfs.GetParentDir)
	if err != nil {
		return nil, err
	}
	return itemPaths(items), nil
}

// BrowseLocalDir lets the user navigate the local filesystem starting at
// startDir and pick a destination directory for a pull. Not covered by
// automated tests (drives the picker TUI); verify manually per SPEC.md.
func BrowseLocalDir(startDir string) (string, error) {
	return browseDir(startDir, localListFunc, localfs.GetParentDir)
}

// BrowseRemoteDir lets the user navigate the remote filesystem starting at
// startDir and pick a destination directory for a push. Not covered by
// automated tests (drives the picker TUI); verify manually per SPEC.md.
func BrowseRemoteDir(client remoteFile, startDir string) (string, error) {
	return browseDir(startDir, remoteListFunc(client), remoteParent)
}

func itemPaths(items []picker.ListItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Path
	}
	return out
}

// browseFiles repeatedly lists dir, presents it in file-pick mode, and
// either navigates into a selected directory or returns the selected
// files. If both files and a directory are tagged together, the files win
// and the directory tag is ignored; if only directories are tagged, it
// navigates into the first one.
func browseFiles(startDir string, list listFunc, parent parentFunc) ([]picker.ListItem, error) {
	dir := startDir
	for {
		entries, err := list(dir)
		if err != nil {
			return nil, err
		}
		items := picker.BuildFileList(entries, parent(dir))
		selected, err := picker.PickFiles(items)
		if err != nil {
			return nil, err
		}

		var files []picker.ListItem
		var firstDir *picker.ListItem
		for i := range selected {
			if selected[i].IsDir {
				if firstDir == nil {
					firstDir = &selected[i]
				}
				continue
			}
			files = append(files, selected[i])
		}
		if len(files) > 0 {
			return files, nil
		}
		if firstDir != nil {
			dir = firstDir.Path
			continue
		}
		return nil, fmt.Errorf("transfer: no file selected")
	}
}

// browseDir repeatedly lists dir, presents it in dir-pick mode, and either
// navigates into a selected directory or returns the current directory
// once the "use this dir" marker is chosen.
func browseDir(startDir string, list listFunc, parent parentFunc) (string, error) {
	dir := startDir
	for {
		entries, err := list(dir)
		if err != nil {
			return "", err
		}
		items := picker.BuildDirList(entries, dir, parent(dir))
		selected, err := picker.PickOne(items)
		if err != nil {
			return "", err
		}
		if selected.IsMarker {
			return selected.Path, nil
		}
		dir = selected.Path
	}
}
