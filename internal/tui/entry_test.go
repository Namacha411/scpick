package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListLocalEntriesTrailsWithParentAfterDirsThenFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "zdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "adir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bfile.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "afile.txt"), []byte("xy"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := listLocalEntries(dir)
	if err != nil {
		t.Fatalf("listLocalEntries: %v", err)
	}

	wantNames := []string{"adir", "zdir", "afile.txt", "bfile.txt", parentEntryName}
	if len(entries) != len(wantNames) {
		t.Fatalf("got %d entries, want %d: %+v", len(entries), len(wantNames), entries)
	}
	for i, want := range wantNames {
		if entries[i].Name != want {
			t.Errorf("entries[%d].Name = %q, want %q", i, entries[i].Name, want)
		}
	}
	last := len(entries) - 1
	if !entries[last].IsParent || !entries[last].IsDir {
		t.Errorf("entries[%d] = %+v, want IsParent && IsDir", last, entries[last])
	}
	for _, i := range []int{0, 1} {
		if !entries[i].IsDir {
			t.Errorf("entries[%d] = %+v, want IsDir", i, entries[i])
		}
	}
	for _, i := range []int{2, 3} {
		if entries[i].IsDir {
			t.Errorf("entries[%d] = %+v, want !IsDir", i, entries[i])
		}
	}
}

func TestRemoteParentOfRootIsRoot(t *testing.T) {
	if got := remoteParent("/"); got != "/" {
		t.Fatalf("remoteParent(/) = %q, want /", got)
	}
}

func TestRemoteParentOfNestedPath(t *testing.T) {
	if got := remoteParent("/a/b"); got != "/a" {
		t.Fatalf("remoteParent(/a/b) = %q, want /a", got)
	}
}

func TestJoinRemotePath(t *testing.T) {
	if got := joinRemotePath("/a", "b"); got != "/a/b" {
		t.Fatalf("joinRemotePath(/a, b) = %q, want /a/b", got)
	}
}
