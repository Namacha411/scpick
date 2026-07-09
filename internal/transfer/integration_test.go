//go:build integration

package transfer

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"

	"scpick/internal/remotefs"
	"scpick/internal/sshtest"
)

func dialTestServer(t *testing.T) *remotefs.Client {
	t.Helper()
	srv := sshtest.Start(t)
	client, err := remotefs.Dial(srv.Addr, sshtest.User, []ssh.AuthMethod{ssh.Password(sshtest.Password)}, ssh.FixedHostKey(srv.HostKey))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestIntegrationPullEndToEnd(t *testing.T) {
	client := dialTestServer(t)

	seedDir := t.TempDir()
	seedFile := filepath.Join(seedDir, "seed.txt")
	if err := os.WriteFile(seedFile, []byte("remote content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := client.Upload(seedFile, "/remote-file.txt", nil); err != nil {
		t.Fatalf("seed upload: %v", err)
	}

	destDir := t.TempDir()
	result := Pull(client, []string{"/remote-file.txt"}, destDir, alwaysYes, nil)
	if len(result.Failed) != 0 {
		t.Fatalf("Pull failed: %+v", result.Failed)
	}
	data, err := os.ReadFile(filepath.Join(destDir, "remote-file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "remote content" {
		t.Errorf("pulled content = %q, want %q", data, "remote content")
	}
}

func TestIntegrationPushEndToEnd(t *testing.T) {
	client := dialTestServer(t)

	srcDir := t.TempDir()
	localFile := writeLocalFile(t, srcDir, "push-me.txt", "local content")

	result := Push(client, []string{localFile}, "/", alwaysYes, nil)
	if len(result.Failed) != 0 {
		t.Fatalf("Push failed: %+v", result.Failed)
	}

	entries, err := client.ListDir("/")
	if err != nil {
		t.Fatalf("ListDir: %v", err)
	}
	var found bool
	for _, e := range entries {
		if e.Name == "push-me.txt" {
			found = true
		}
	}
	if !found {
		t.Fatal("pushed file not found on remote")
	}
}
