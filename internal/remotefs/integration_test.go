//go:build integration

package remotefs

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"

	"scpick/internal/sshtest"
)

func TestIntegrationUploadListStatDownload(t *testing.T) {
	srv := sshtest.Start(t)
	hostKeyCB := ssh.FixedHostKey(srv.HostKey)
	methods := []ssh.AuthMethod{ssh.Password(sshtest.Password)}

	client, err := Dial(srv.Addr, sshtest.User, methods, hostKeyCB)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	localDir := t.TempDir()
	localSrc := filepath.Join(localDir, "hello.txt")
	const content = "hello sftp"
	if err := os.WriteFile(localSrc, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var progressCalls int
	if err := client.Upload(localSrc, "/hello.txt", func(done, total int64) { progressCalls++ }); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if progressCalls == 0 {
		t.Error("expected at least one progress callback during upload")
	}

	entries, err := client.ListDir("/")
	if err != nil {
		t.Fatalf("ListDir: %v", err)
	}
	var found bool
	for _, e := range entries {
		if e.Name == "hello.txt" {
			found = true
			if e.Size != int64(len(content)) {
				t.Errorf("entry size = %d, want %d", e.Size, len(content))
			}
		}
	}
	if !found {
		t.Fatal("uploaded file not found in ListDir")
	}

	stat, err := client.Stat("/hello.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if stat.Size != int64(len(content)) {
		t.Errorf("Stat size = %d, want %d", stat.Size, len(content))
	}

	localDst := filepath.Join(localDir, "downloaded.txt")
	if err := client.Download("/hello.txt", localDst, nil); err != nil {
		t.Fatalf("Download: %v", err)
	}
	data, err := os.ReadFile(localDst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("downloaded content = %q, want %q", data, content)
	}
}

func TestIntegrationBadPasswordRejected(t *testing.T) {
	srv := sshtest.Start(t)
	hostKeyCB := ssh.FixedHostKey(srv.HostKey)

	_, err := Dial(srv.Addr, sshtest.User, []ssh.AuthMethod{ssh.Password("wrong")}, hostKeyCB)
	if err == nil {
		t.Fatal("expected Dial to fail with a wrong password")
	}
}
