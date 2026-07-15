package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListLocalEntriesLeadsWithParentThenDirsThenFiles(t *testing.T) {
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

	wantNames := []string{parentEntryName, "adir", "zdir", "afile.txt", "bfile.txt"}
	if len(entries) != len(wantNames) {
		t.Fatalf("got %d entries, want %d: %+v", len(entries), len(wantNames), entries)
	}
	for i, want := range wantNames {
		if entries[i].Name != want {
			t.Errorf("entries[%d].Name = %q, want %q", i, entries[i].Name, want)
		}
	}
	if !entries[0].IsParent || !entries[0].IsDir {
		t.Errorf("entries[0] = %+v, want IsParent && IsDir", entries[0])
	}
	for _, i := range []int{1, 2} {
		if !entries[i].IsDir {
			t.Errorf("entries[%d] = %+v, want IsDir", i, entries[i])
		}
	}
	for _, i := range []int{3, 4} {
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
