package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

// genTestKeyPEM generates a throwaway ed25519 key at test time (never
// committed to the repo) and returns it PEM-encoded, optionally encrypted
// with passphrase.
func genTestKeyPEM(t *testing.T, passphrase string) []byte {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var block *pem.Block
	if passphrase == "" {
		block, err = ssh.MarshalPrivateKey(priv, "scpick test key")
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(priv, "scpick test key", []byte(passphrase))
	}
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(block)
}

func TestLoadKeyFileSignersUnencrypted(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519"), genTestKeyPEM(t, ""), 0o600); err != nil {
		t.Fatal(err)
	}

	signers, err := loadKeyFileSigners(dir, func(string) (string, error) {
		t.Fatal("passwordFunc should not be called for an unencrypted key")
		return "", nil
	})
	if err != nil {
		t.Fatalf("loadKeyFileSigners: %v", err)
	}
	if len(signers) != 1 {
		t.Fatalf("got %d signers, want 1", len(signers))
	}
}

func TestLoadKeyFileSignersEncrypted(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519"), genTestKeyPEM(t, "correct-horse"), 0o600); err != nil {
		t.Fatal(err)
	}

	calls := 0
	signers, err := loadKeyFileSigners(dir, func(string) (string, error) {
		calls++
		return "correct-horse", nil
	})
	if err != nil {
		t.Fatalf("loadKeyFileSigners: %v", err)
	}
	if len(signers) != 1 {
		t.Fatalf("got %d signers, want 1", len(signers))
	}
	if calls != 1 {
		t.Fatalf("passwordFunc called %d times, want 1", calls)
	}
}

func TestLoadKeyFileSignersWrongPassphrase(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519"), genTestKeyPEM(t, "correct-horse"), 0o600); err != nil {
		t.Fatal(err)
	}

	signers, err := loadKeyFileSigners(dir, func(string) (string, error) {
		return "wrong-passphrase", nil
	})
	if err != nil {
		t.Fatalf("loadKeyFileSigners: %v", err)
	}
	if len(signers) != 0 {
		t.Fatalf("got %d signers, want 0 for wrong passphrase", len(signers))
	}
}

func TestLoadKeyFileSignersNoneExist(t *testing.T) {
	signers, err := loadKeyFileSigners(t.TempDir(), func(string) (string, error) {
		t.Fatal("passwordFunc should not be called")
		return "", nil
	})
	if err != nil {
		t.Fatalf("loadKeyFileSigners: %v", err)
	}
	if len(signers) != 0 {
		t.Fatalf("got %d signers, want 0", len(signers))
	}
}
