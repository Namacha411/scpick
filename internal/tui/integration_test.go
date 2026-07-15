//go:build integration

package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/crypto/ssh"

	"scpick/internal/remotefs"
	"scpick/internal/sshtest"
	"scpick/internal/transfer"
)

func TestIntegrationUpdateConnectEventSuccessListsRemoteRoot(t *testing.T) {
	srv := sshtest.Start(t)
	client, err := remotefs.Dial(srv.Addr, sshtest.User, []ssh.AuthMethod{ssh.Password(sshtest.Password)}, ssh.FixedHostKey(srv.HostKey))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	m := NewModel()
	m.mode = ModeConnecting
	m.errMsg = "stale error"

	newModel, cmd := m.updateConnectEvent(connectResultMsg{client: client})
	got := newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse", got.mode)
	}
	if got.errMsg != "" {
		t.Fatalf("errMsg = %q, want cleared", got.errMsg)
	}
	if got.remoteClient != client {
		t.Fatal("remoteClient was not set to the connected client")
	}
	if got.remote.path != "/" {
		t.Fatalf("remote.path = %q, want %q", got.remote.path, "/")
	}
	if len(got.remote.entries) == 0 || got.remote.entries[0].Name != parentEntryName {
		t.Fatalf("remote.entries = %+v, want a leading %q entry", got.remote.entries, parentEntryName)
	}
	if cmd != nil {
		t.Fatal("expected no further listening command on terminal success")
	}
}

func TestIntegrationStartTransferWorkerPushesFileToRemote(t *testing.T) {
	srv := sshtest.Start(t)
	client, err := remotefs.Dial(srv.Addr, sshtest.User, []ssh.AuthMethod{ssh.Password(sshtest.Password)}, ssh.FixedHostKey(srv.HostKey))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	localDir := t.TempDir()
	localFile := filepath.Join(localDir, "push-me.txt")
	if err := os.WriteFile(localFile, []byte("hello from local"), 0o644); err != nil {
		t.Fatal(err)
	}

	events := make(chan tea.Msg, 100)
	progress := func(label string, done, total int64) {
		events <- transferProgressMsg{label: label, done: done, total: total}
	}
	cmd := startTransferWorker(client, false /* push */, []string{localFile}, nil, "/", fixedOverwrite(transfer.OverwriteAll), progress, events)
	cmd() // launches the background goroutine and returns immediately

	result := waitForDone(t, events)
	if len(result.Failed) != 0 {
		t.Fatalf("Failed = %+v", result.Failed)
	}
	if len(result.Succeeded) != 1 {
		t.Fatalf("Succeeded = %v, want 1 entry", result.Succeeded)
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

func TestIntegrationStartTransferWorkerPullsFileFromRemote(t *testing.T) {
	srv := sshtest.Start(t)
	client, err := remotefs.Dial(srv.Addr, sshtest.User, []ssh.AuthMethod{ssh.Password(sshtest.Password)}, ssh.FixedHostKey(srv.HostKey))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	seedDir := t.TempDir()
	seedFile := filepath.Join(seedDir, "seed.txt")
	if err := os.WriteFile(seedFile, []byte("remote content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := client.Upload(seedFile, "/remote-file.txt", nil); err != nil {
		t.Fatalf("seed upload: %v", err)
	}

	destDir := t.TempDir()
	events := make(chan tea.Msg, 100)
	progress := func(label string, done, total int64) {
		events <- transferProgressMsg{label: label, done: done, total: total}
	}
	cmd := startTransferWorker(client, true /* pull */, []string{"/remote-file.txt"}, nil, destDir, fixedOverwrite(transfer.OverwriteAll), progress, events)
	cmd()

	result := waitForDone(t, events)
	if len(result.Failed) != 0 {
		t.Fatalf("Failed = %+v", result.Failed)
	}
	data, err := os.ReadFile(filepath.Join(destDir, "remote-file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "remote content" {
		t.Fatalf("pulled content = %q, want %q", data, "remote content")
	}
}

func waitForDone(t *testing.T, events chan tea.Msg) transfer.Result {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case msg := <-events:
			if done, ok := msg.(transferDoneMsg); ok {
				return done.result
			}
		case <-deadline:
			t.Fatal("timed out waiting for transfer to finish")
			return transfer.Result{}
		}
	}
}
