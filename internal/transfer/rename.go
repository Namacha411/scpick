package transfer

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// uniqueName finds the first "stem (n)ext" candidate, starting at n=1, for
// which exists reports false. Pure dotfiles (".gitignore") have no
// extension to split off in the usual sense, so they're numbered as
// ".gitignore (1)" rather than " (1).gitignore".
func uniqueName(base string, exists func(candidate string) bool) string {
	ext := path.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if stem == "" { // dotfile: path.Ext(".gitignore") == ".gitignore"
		stem, ext = base, ""
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", stem, i, ext)
		if !exists(candidate) {
			return candidate
		}
	}
}

// uniqueLocalPath returns destPath if nothing exists there, otherwise the
// first numbered variant that doesn't collide locally.
func uniqueLocalPath(destPath string) string {
	dir := filepath.Dir(destPath)
	name := uniqueName(filepath.Base(destPath), func(candidate string) bool {
		_, err := os.Stat(filepath.Join(dir, candidate))
		return err == nil
	})
	return filepath.Join(dir, name)
}

// uniqueRemotePath mirrors uniqueLocalPath against the remote side via
// client.Stat.
func uniqueRemotePath(client remoteFile, destPath string) string {
	dir := path.Dir(destPath)
	name := uniqueName(path.Base(destPath), func(candidate string) bool {
		_, err := client.Stat(path.Join(dir, candidate))
		return err == nil
	})
	return path.Join(dir, name)
}
