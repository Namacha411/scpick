package localfs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListDir(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "b.txt"), "hello")
	mustWriteFile(t, filepath.Join(dir, "a.txt"), "hi")
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	entries, err := ListDir(dir)
	if err != nil {
		t.Fatalf("ListDir: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	if !entries[0].IsDir || entries[0].Name != "sub" {
		t.Errorf("entries[0] = %+v, want sub dir first", entries[0])
	}
	if entries[1].Name != "a.txt" || entries[2].Name != "b.txt" {
		t.Errorf("file order = %q, %q; want a.txt, b.txt", entries[1].Name, entries[2].Name)
	}
	if entries[1].Size != 2 {
		t.Errorf("a.txt size = %d, want 2", entries[1].Size)
	}
}

func TestListDirMissing(t *testing.T) {
	_, err := ListDir(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir available")
	}
	tests := []struct {
		in   string
		want string
	}{
		{"~", home},
		{filepath.Join("~", "docs"), filepath.Join(home, "docs")},
		{"relative/path", "relative/path"},
	}
	for _, tt := range tests {
		got, err := ExpandHome(tt.in)
		if err != nil {
			t.Fatalf("ExpandHome(%q): %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("ExpandHome(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGetParentDirAtRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		if got := GetParentDir(`C:\`); got != DrivesMarker {
			t.Errorf(`GetParentDir(C:\) = %q, want %q`, got, DrivesMarker)
		}
		if got := GetParentDir(DrivesMarker); got != DrivesMarker {
			t.Errorf("GetParentDir(marker) = %q, want %q", got, DrivesMarker)
		}
		return
	}
	if got := GetParentDir("/"); got != "/" {
		t.Errorf("GetParentDir(/) = %q, want /", got)
	}
}

func TestJoinPathDrives(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("drive letters are Windows-only")
	}
	got := JoinPath(DrivesMarker, "C:")
	want := `C:\`
	if got != want {
		t.Errorf("JoinPath(marker, C:) = %q, want %q", got, want)
	}
}

func TestListDrives(t *testing.T) {
	entries, err := ListDrives()
	if runtime.GOOS != "windows" {
		if err == nil {
			t.Fatal("expected error on non-Windows")
		}
		return
	}
	if err != nil {
		t.Fatalf("ListDrives: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one drive")
	}
}
