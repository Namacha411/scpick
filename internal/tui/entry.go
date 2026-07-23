package tui

import (
	"path"

	"github.com/sahilm/fuzzy"

	"scpick/internal/localfs"
	"scpick/internal/remotefs"
)

// paneEntry is one row shown in a pane: either a real file/directory, or the
// leading ".." pseudo-entry that navigates to the parent directory.
type paneEntry struct {
	Name     string
	IsDir    bool
	Size     int64
	IsParent bool
}

const parentEntryName = ".."

// listLocalEntries lists dir on the local filesystem, suffixed with a ".."
// entry so it lands last and out of the way of the cursor's default resting
// place. localfs.ListDir already sorts directories first, then files, each
// group alphabetically.
func listLocalEntries(dir string) ([]paneEntry, error) {
	entries, err := localfs.ListDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]paneEntry, 0, len(entries)+1)
	for _, e := range entries {
		out = append(out, paneEntry{Name: e.Name, IsDir: e.IsDir, Size: e.Size})
	}
	out = append(out, paneEntry{Name: parentEntryName, IsDir: true, IsParent: true})
	return out, nil
}

// listRemoteEntries lists dir on the connected remote host, suffixed with a
// ".." entry so it lands last and out of the way of the cursor's default
// resting place. client.ListDir already sorts directories first, then
// files, each group alphabetically.
func listRemoteEntries(client *remotefs.Client, dir string) ([]paneEntry, error) {
	entries, err := client.ListDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]paneEntry, 0, len(entries)+1)
	for _, e := range entries {
		out = append(out, paneEntry{Name: e.Name, IsDir: e.IsDir, Size: e.Size})
	}
	out = append(out, paneEntry{Name: parentEntryName, IsDir: true, IsParent: true})
	return out, nil
}

// localParent and remoteParent compute the parent directory for the local
// and remote sides respectively. remoteParent is POSIX-only, since SFTP
// paths are always POSIX regardless of the remote host's OS.
func localParent(dir string) string {
	return localfs.GetParentDir(dir)
}

func remoteParent(dir string) string {
	if dir == "/" {
		return "/"
	}
	return path.Dir(dir)
}

// joinRemotePath joins a remote directory and a child entry name using
// POSIX rules, since SFTP paths are always POSIX regardless of the remote
// host's OS.
func joinRemotePath(dir, name string) string {
	return path.Join(dir, name)
}

// entryNames adapts []paneEntry to sahilm/fuzzy's Source interface so
// entries can be fuzzy-matched by name without building an intermediate
// []string.
type entryNames []paneEntry

func (e entryNames) String(i int) string { return e[i].Name }
func (e entryNames) Len() int            { return len(e) }

// fuzzyMatchIndices returns the indices into entries whose name
// fuzzy-matches query, best match first.
func fuzzyMatchIndices(query string, entries []paneEntry) []int {
	matches := fuzzy.FindFrom(query, entryNames(entries))
	out := make([]int, len(matches))
	for i, m := range matches {
		out[i] = m.Index
	}
	return out
}

// joinSourcePath builds the full source path of an entry named name inside
// dir, on the local or remote side depending on focus (0 = local, 1 =
// remote).
func joinSourcePath(focus int, dir, name string) string {
	if focus == 0 {
		return localfs.JoinPath(dir, name)
	}
	return joinRemotePath(dir, name)
}
