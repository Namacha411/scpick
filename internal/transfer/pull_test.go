package transfer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"scpick/internal/remotefs"
)

func alwaysYes(string, int64, int64) OverwriteDecision { return OverwriteYes }

func TestPullSuccess(t *testing.T) {
	destDir := t.TempDir()
	client := &fakeClient{
		stat: map[string]remotefs.Entry{
			"/remote/a.txt": {Name: "a.txt", Size: 5},
		},
		download: func(remotePath, localPath string, onProgress remotefs.ProgressFunc) error {
			if onProgress != nil {
				onProgress(5, 5)
			}
			return os.WriteFile(localPath, []byte("hello"), 0o644)
		},
	}

	result := Pull(client, []string{"/remote/a.txt"}, destDir, alwaysYes, nil)

	if len(result.Failed) != 0 {
		t.Fatalf("unexpected failures: %v", result.Failed)
	}
	if len(result.Succeeded) != 1 || result.Succeeded[0] != "/remote/a.txt" {
		t.Fatalf("Succeeded = %v, want [/remote/a.txt]", result.Succeeded)
	}
	if data, err := os.ReadFile(filepath.Join(destDir, "a.txt")); err != nil || string(data) != "hello" {
		t.Fatalf("downloaded file content = %q, err = %v", data, err)
	}
}

func TestPullOverwriteSkip(t *testing.T) {
	destDir := t.TempDir()
	existingPath := filepath.Join(destDir, "a.txt")
	if err := os.WriteFile(existingPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	downloadCalled := false
	client := &fakeClient{
		stat: map[string]remotefs.Entry{
			"/remote/a.txt": {Name: "a.txt", Size: 5},
		},
		download: func(remotePath, localPath string, onProgress remotefs.ProgressFunc) error {
			downloadCalled = true
			return nil
		},
	}

	confirm := func(destPath string, existingSize, newSize int64) OverwriteDecision {
		if destPath != existingPath {
			t.Errorf("confirm destPath = %q, want %q", destPath, existingPath)
		}
		if existingSize != 3 || newSize != 5 {
			t.Errorf("confirm sizes = %d/%d, want 3/5", existingSize, newSize)
		}
		return OverwriteSkip
	}

	result := Pull(client, []string{"/remote/a.txt"}, destDir, confirm, nil)

	if downloadCalled {
		t.Error("Download should not be called when the user skips the overwrite")
	}
	if len(result.Skipped) != 1 || result.Skipped[0] != "/remote/a.txt" {
		t.Fatalf("Skipped = %v, want [/remote/a.txt]", result.Skipped)
	}
	if len(result.Succeeded) != 0 || len(result.Failed) != 0 {
		t.Fatalf("expected no successes or failures, got %+v", result)
	}
}

func TestPullOverwriteAllAppliesToRestOfBatch(t *testing.T) {
	destDir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(destDir, name), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	client := &fakeClient{
		stat: map[string]remotefs.Entry{
			"/remote/a.txt": {Name: "a.txt", Size: 5},
			"/remote/b.txt": {Name: "b.txt", Size: 5},
			"/remote/c.txt": {Name: "c.txt", Size: 5},
		},
		download: func(remotePath, localPath string, onProgress remotefs.ProgressFunc) error {
			return os.WriteFile(localPath, []byte("hello"), 0o644)
		},
	}

	confirmCalls := 0
	confirm := func(string, int64, int64) OverwriteDecision {
		confirmCalls++
		return OverwriteAll
	}

	result := Pull(client, []string{"/remote/a.txt", "/remote/b.txt", "/remote/c.txt"}, destDir, confirm, nil)

	if confirmCalls != 1 {
		t.Errorf("confirm called %d times, want 1 (yes-to-all should suppress further prompts)", confirmCalls)
	}
	if len(result.Succeeded) != 3 {
		t.Fatalf("Succeeded = %v, want 3 files", result.Succeeded)
	}
}

func TestPullPartialFailureContinues(t *testing.T) {
	destDir := t.TempDir()
	client := &fakeClient{
		stat: map[string]remotefs.Entry{
			"/remote/good.txt": {Name: "good.txt", Size: 5},
		},
		statErr: map[string]error{
			"/remote/bad.txt": errors.New("no such file"),
		},
		download: func(remotePath, localPath string, onProgress remotefs.ProgressFunc) error {
			return os.WriteFile(localPath, []byte("hello"), 0o644)
		},
	}

	result := Pull(client, []string{"/remote/bad.txt", "/remote/good.txt"}, destDir, alwaysYes, nil)

	if len(result.Succeeded) != 1 || result.Succeeded[0] != "/remote/good.txt" {
		t.Fatalf("Succeeded = %v, want [/remote/good.txt]", result.Succeeded)
	}
	if _, ok := result.Failed["/remote/bad.txt"]; !ok {
		t.Fatalf("expected /remote/bad.txt to be recorded as failed, got %+v", result.Failed)
	}
}

func TestPullDownloadErrorRecorded(t *testing.T) {
	destDir := t.TempDir()
	wantErr := errors.New("connection reset")
	client := &fakeClient{
		stat: map[string]remotefs.Entry{
			"/remote/a.txt": {Name: "a.txt", Size: 5},
		},
		download: func(remotePath, localPath string, onProgress remotefs.ProgressFunc) error {
			return wantErr
		},
	}

	result := Pull(client, []string{"/remote/a.txt"}, destDir, alwaysYes, nil)

	if len(result.Succeeded) != 0 {
		t.Fatalf("Succeeded = %v, want none", result.Succeeded)
	}
	if !errors.Is(result.Failed["/remote/a.txt"], wantErr) {
		t.Fatalf("Failed[/remote/a.txt] = %v, want wrapping %v", result.Failed["/remote/a.txt"], wantErr)
	}
}
