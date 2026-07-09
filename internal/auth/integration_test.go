//go:build integration

package auth

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"scpick/internal/sshtest"
)

func TestIntegrationHostKeyCallbackTrustOnFirstUse(t *testing.T) {
	srv := sshtest.Start(t)
	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")

	cb, err := HostKeyCallback(knownHostsPath, func(hostname, fingerprint string) bool { return true })
	if err != nil {
		t.Fatalf("HostKeyCallback: %v", err)
	}
	if _, err := ssh.Dial("tcp", srv.Addr, &ssh.ClientConfig{
		User:            sshtest.User,
		Auth:            []ssh.AuthMethod{ssh.Password(sshtest.Password)},
		HostKeyCallback: cb,
	}); err != nil {
		t.Fatalf("first dial (trust-on-first-use): %v", err)
	}

	cb2, err := HostKeyCallback(knownHostsPath, func(string, string) bool {
		t.Fatal("confirm should not be called for an already-trusted host")
		return false
	})
	if err != nil {
		t.Fatalf("HostKeyCallback (2nd): %v", err)
	}
	conn2, err := ssh.Dial("tcp", srv.Addr, &ssh.ClientConfig{
		User:            sshtest.User,
		Auth:            []ssh.AuthMethod{ssh.Password(sshtest.Password)},
		HostKeyCallback: cb2,
	})
	if err != nil {
		t.Fatalf("second dial (already-known host): %v", err)
	}
	conn2.Close()
}

func TestIntegrationHostKeyCallbackMismatchAborts(t *testing.T) {
	srv := sshtest.Start(t)
	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")

	wrongKey := genHostKey(t) // from known_hosts_test.go, same package
	line := knownhosts.Line([]string{srv.Addr}, wrongKey) + "\n"
	if err := os.WriteFile(knownHostsPath, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}

	cb, err := HostKeyCallback(knownHostsPath, func(string, string) bool {
		t.Fatal("confirm must never be called on a genuine mismatch")
		return false
	})
	if err != nil {
		t.Fatalf("HostKeyCallback: %v", err)
	}
	if _, err := ssh.Dial("tcp", srv.Addr, &ssh.ClientConfig{
		User:            sshtest.User,
		Auth:            []ssh.AuthMethod{ssh.Password(sshtest.Password)},
		HostKeyCallback: cb,
	}); err == nil {
		t.Fatal("expected dial to fail due to host key mismatch")
	}
}
