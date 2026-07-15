package transfer

import (
	"os"
	"path/filepath"
	"testing"

	"scpick/internal/remotefs"
)

// remoteTree is a fake remote filesystem: dir path -> its entries.
type remoteTree map[string][]remotefs.Entry

func newTreeClient(t *testing.T, tree remoteTree, fileSize int64) *fakeClient {
	t.Helper()
	return &fakeClient{
		listDir: func(path string) ([]remotefs.Entry, error) {
			entries, ok := tree[path]
			if !ok {
				t.Fatalf("ListDir(%q): no such directory in fake tree", path)
			}
			return entries, nil
		},
		stat: map[string]remotefs.Entry{},
		download: func(remotePath, localPath string, onProgress remotefs.ProgressFunc) error {
			return os.WriteFile(localPath, []byte("x"), 0o644)
		},
	}
}

func TestRecursivePullNestedTree(t *testing.T) {
	destDir := t.TempDir()
	tree := remoteTree{
		"/remote/backup": {
			{Name: "a.txt", IsDir: false, Size: 1},
			{Name: "sub", IsDir: true},
		},
		"/remote/backup/sub": {
			{Name: "b.txt", IsDir: false, Size: 1},
		},
	}
	client := newTreeClient(t, tree, 1)
	client.stat["/remote/backup/a.txt"] = remotefs.Entry{Name: "a.txt", Size: 1}
	client.stat["/remote/backup/sub/b.txt"] = remotefs.Entry{Name: "b.txt", Size: 1}

	result := RecursivePull(client, nil, []string{"/remote/backup"}, destDir, alwaysYes, nil)

	if len(result.Failed) != 0 {
		t.Fatalf("unexpected failures: %v", result.Failed)
	}
	if len(result.Succeeded) != 2 {
		t.Fatalf("Succeeded = %v, want 2 files", result.Succeeded)
	}
	if _, err := os.Stat(filepath.Join(destDir, "backup", "a.txt")); err != nil {
		t.Errorf("backup/a.txt missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "backup", "sub", "b.txt")); err != nil {
		t.Errorf("backup/sub/b.txt missing: %v", err)
	}
}

func TestRecursivePullMixedFilesAndDirs(t *testing.T) {
	destDir := t.TempDir()
	tree := remoteTree{
		"/remote/dir": {
			{Name: "c.txt", IsDir: false, Size: 1},
		},
	}
	client := newTreeClient(t, tree, 1)
	client.stat["/remote/top.txt"] = remotefs.Entry{Name: "top.txt", Size: 1}
	client.stat["/remote/dir/c.txt"] = remotefs.Entry{Name: "c.txt", Size: 1}

	result := RecursivePull(client, []string{"/remote/top.txt"}, []string{"/remote/dir"}, destDir, alwaysYes, nil)

	if len(result.Failed) != 0 {
		t.Fatalf("unexpected failures: %v", result.Failed)
	}
	if len(result.Succeeded) != 2 {
		t.Fatalf("Succeeded = %v, want 2 files", result.Succeeded)
	}
	if _, err := os.Stat(filepath.Join(destDir, "top.txt")); err != nil {
		t.Errorf("top.txt missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "dir", "c.txt")); err != nil {
		t.Errorf("dir/c.txt missing: %v", err)
	}
}

func TestRecursivePullOverwriteAllPersistsAcrossDirectories(t *testing.T) {
	destDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destDir, "backup", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(destDir, "backup", "a.txt"),
		filepath.Join(destDir, "backup", "sub", "b.txt"),
	} {
		if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tree := remoteTree{
		"/remote/backup": {
			{Name: "a.txt", IsDir: false, Size: 1},
			{Name: "sub", IsDir: true},
		},
		"/remote/backup/sub": {
			{Name: "b.txt", IsDir: false, Size: 1},
		},
	}
	client := newTreeClient(t, tree, 1)
	client.stat["/remote/backup/a.txt"] = remotefs.Entry{Name: "a.txt", Size: 1}
	client.stat["/remote/backup/sub/b.txt"] = remotefs.Entry{Name: "b.txt", Size: 1}

	confirmCalls := 0
	confirm := func(string, int64, int64) OverwriteDecision {
		confirmCalls++
		return OverwriteAll
	}

	result := RecursivePull(client, nil, []string{"/remote/backup"}, destDir, confirm, nil)

	if confirmCalls != 1 {
		t.Errorf("confirm called %d times, want 1 (yes-to-all should persist across subdirectories)", confirmCalls)
	}
	if len(result.Succeeded) != 2 {
		t.Fatalf("Succeeded = %v, want 2 files", result.Succeeded)
	}
}

func TestRecursivePushNestedTree(t *testing.T) {
	srcRoot := t.TempDir()
	backupDir := filepath.Join(srcRoot, "backup")
	subDir := filepath.Join(backupDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "b.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	var mkdirCalls []string
	var uploadCalls []string
	client := &fakeClient{
		stat: map[string]remotefs.Entry{},
		mkdirAll: func(path string) error {
			mkdirCalls = append(mkdirCalls, path)
			return nil
		},
		upload: func(localPath, remotePath string, onProgress remotefs.ProgressFunc) error {
			uploadCalls = append(uploadCalls, remotePath)
			return nil
		},
	}

	result := RecursivePush(client, nil, []string{backupDir}, "/remote/dest", alwaysYes, nil)

	if len(result.Failed) != 0 {
		t.Fatalf("unexpected failures: %v", result.Failed)
	}
	if len(result.Succeeded) != 2 {
		t.Fatalf("Succeeded = %v, want 2 files", result.Succeeded)
	}

	wantDirs := map[string]bool{"/remote/dest/backup": true, "/remote/dest/backup/sub": true}
	for _, d := range mkdirCalls {
		if !wantDirs[d] {
			t.Errorf("unexpected MkdirAll(%q)", d)
		}
		delete(wantDirs, d)
	}
	if len(wantDirs) != 0 {
		t.Errorf("missing MkdirAll calls for: %v", wantDirs)
	}

	wantUploads := map[string]bool{"/remote/dest/backup/a.txt": true, "/remote/dest/backup/sub/b.txt": true}
	for _, u := range uploadCalls {
		if !wantUploads[u] {
			t.Errorf("unexpected Upload to %q", u)
		}
		delete(wantUploads, u)
	}
	if len(wantUploads) != 0 {
		t.Errorf("missing Upload calls for: %v", wantUploads)
	}
}

func TestMergeResults(t *testing.T) {
	dest := Result{Failed: map[string]error{}}
	src := Result{
		Succeeded: []string{"a"},
		Skipped:   []string{"b"},
		Failed:    map[string]error{"c": os.ErrNotExist},
	}
	mergeResults(&dest, src)
	mergeResults(&dest, Result{Succeeded: []string{"d"}, Failed: map[string]error{}})

	if len(dest.Succeeded) != 2 || len(dest.Skipped) != 1 || len(dest.Failed) != 1 {
		t.Fatalf("merged = %+v", dest)
	}
}
