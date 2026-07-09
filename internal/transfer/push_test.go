package transfer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"scpick/internal/remotefs"
)

func writeLocalFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPushSuccess(t *testing.T) {
	srcDir := t.TempDir()
	localPath := writeLocalFile(t, srcDir, "a.txt", "hello")

	var uploadedTo string
	client := &fakeClient{
		statErr: map[string]error{"/remote/a.txt": errors.New("not found")},
		upload: func(lp, remotePath string, onProgress remotefs.ProgressFunc) error {
			uploadedTo = remotePath
			if onProgress != nil {
				onProgress(5, 5)
			}
			return nil
		},
	}

	result := Push(client, []string{localPath}, "/remote", alwaysYes, nil)

	if len(result.Failed) != 0 {
		t.Fatalf("unexpected failures: %v", result.Failed)
	}
	if len(result.Succeeded) != 1 || result.Succeeded[0] != localPath {
		t.Fatalf("Succeeded = %v, want [%s]", result.Succeeded, localPath)
	}
	if uploadedTo != "/remote/a.txt" {
		t.Errorf("uploaded to %q, want /remote/a.txt", uploadedTo)
	}
}

func TestPushOverwriteSkip(t *testing.T) {
	srcDir := t.TempDir()
	localPath := writeLocalFile(t, srcDir, "a.txt", "hello")

	uploadCalled := false
	client := &fakeClient{
		stat: map[string]remotefs.Entry{
			"/remote/a.txt": {Name: "a.txt", Size: 3},
		},
		upload: func(string, string, remotefs.ProgressFunc) error {
			uploadCalled = true
			return nil
		},
	}

	confirm := func(destPath string, existingSize, newSize int64) OverwriteDecision {
		if destPath != "/remote/a.txt" {
			t.Errorf("confirm destPath = %q, want /remote/a.txt", destPath)
		}
		if existingSize != 3 || newSize != 5 {
			t.Errorf("confirm sizes = %d/%d, want 3/5", existingSize, newSize)
		}
		return OverwriteSkip
	}

	result := Push(client, []string{localPath}, "/remote", confirm, nil)

	if uploadCalled {
		t.Error("Upload should not be called when the user skips the overwrite")
	}
	if len(result.Skipped) != 1 || result.Skipped[0] != localPath {
		t.Fatalf("Skipped = %v, want [%s]", result.Skipped, localPath)
	}
}

func TestPushPartialFailureContinues(t *testing.T) {
	srcDir := t.TempDir()
	goodPath := writeLocalFile(t, srcDir, "good.txt", "hello")
	badPath := filepath.Join(srcDir, "does-not-exist.txt")

	client := &fakeClient{
		statErr: map[string]error{
			"/remote/good.txt":           errors.New("not found"),
			"/remote/does-not-exist.txt": errors.New("not found"),
		},
		upload: func(string, string, remotefs.ProgressFunc) error { return nil },
	}

	result := Push(client, []string{badPath, goodPath}, "/remote", alwaysYes, nil)

	if len(result.Succeeded) != 1 || result.Succeeded[0] != goodPath {
		t.Fatalf("Succeeded = %v, want [%s]", result.Succeeded, goodPath)
	}
	if _, ok := result.Failed[badPath]; !ok {
		t.Fatalf("expected %s to be recorded as failed, got %+v", badPath, result.Failed)
	}
}

func TestPushUploadErrorRecorded(t *testing.T) {
	srcDir := t.TempDir()
	localPath := writeLocalFile(t, srcDir, "a.txt", "hello")
	wantErr := errors.New("connection reset")

	client := &fakeClient{
		statErr: map[string]error{"/remote/a.txt": errors.New("not found")},
		upload:  func(string, string, remotefs.ProgressFunc) error { return wantErr },
	}

	result := Push(client, []string{localPath}, "/remote", alwaysYes, nil)

	if len(result.Succeeded) != 0 {
		t.Fatalf("Succeeded = %v, want none", result.Succeeded)
	}
	if !errors.Is(result.Failed[localPath], wantErr) {
		t.Fatalf("Failed[%s] = %v, want wrapping %v", localPath, result.Failed[localPath], wantErr)
	}
}
