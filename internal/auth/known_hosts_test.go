package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func genHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return sshPub
}

func TestHostKeyCallbackUnknownHostTrusted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known_hosts")
	key := genHostKey(t)

	var gotHostname, gotFingerprint string
	cb, err := HostKeyCallback(path, func(hostname, fingerprint string) bool {
		gotHostname, gotFingerprint = hostname, fingerprint
		return true
	})
	if err != nil {
		t.Fatalf("HostKeyCallback: %v", err)
	}
	if err := cb("example.com:22", &net.TCPAddr{}, key); err != nil {
		t.Fatalf("cb: %v", err)
	}
	if gotHostname != "example.com:22" {
		t.Errorf("confirm hostname = %q", gotHostname)
	}
	if want := ssh.FingerprintSHA256(key); gotFingerprint != want {
		t.Errorf("confirm fingerprint = %q, want %q", gotFingerprint, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected key to be appended to known_hosts")
	}

	cb2, err := HostKeyCallback(path, func(string, string) bool {
		t.Fatal("confirm should not be called for an already-known host")
		return false
	})
	if err != nil {
		t.Fatalf("HostKeyCallback (2nd): %v", err)
	}
	if err := cb2("example.com:22", &net.TCPAddr{}, key); err != nil {
		t.Fatalf("cb2: %v", err)
	}
}

func TestHostKeyCallbackUnknownHostRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known_hosts")
	key := genHostKey(t)

	cb, err := HostKeyCallback(path, func(string, string) bool { return false })
	if err != nil {
		t.Fatalf("HostKeyCallback: %v", err)
	}
	if err := cb("example.com:22", &net.TCPAddr{}, key); err == nil {
		t.Fatal("expected error when user rejects an unknown host key")
	}
}

func TestHostKeyCallbackMismatchNeverPrompts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known_hosts")
	original := genHostKey(t)
	different := genHostKey(t)

	cb, err := HostKeyCallback(path, func(string, string) bool { return true })
	if err != nil {
		t.Fatalf("HostKeyCallback: %v", err)
	}
	if err := cb("example.com:22", &net.TCPAddr{}, original); err != nil {
		t.Fatalf("trust original key: %v", err)
	}

	cb2, err := HostKeyCallback(path, func(string, string) bool {
		t.Fatal("confirm must never be called on a genuine mismatch")
		return false
	})
	if err != nil {
		t.Fatalf("HostKeyCallback (2nd): %v", err)
	}
	if err := cb2("example.com:22", &net.TCPAddr{}, different); err == nil {
		t.Fatal("expected error on host key mismatch")
	}
}

func TestHostKeyCallbackCreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "known_hosts")
	if _, err := HostKeyCallback(path, func(string, string) bool { return true }); err != nil {
		t.Fatalf("HostKeyCallback: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected known_hosts file to be created: %v", err)
	}
}
