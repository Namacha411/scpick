package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

type fakeAgent struct {
	signers []ssh.Signer
	err     error
}

func (f fakeAgent) Signers() ([]ssh.Signer, error) {
	return f.signers, f.err
}

func genSigner(t *testing.T) ssh.Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

func TestCollectSignersPrefersAgent(t *testing.T) {
	dir := t.TempDir()
	// A key file is present too, to prove the agent takes priority: its
	// passphrase prompt would fail this test if ever invoked.
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519"), genTestKeyPEM(t, "secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	ac := &AuthChain{
		keyDir: dir,
		passwordFunc: func(string) (string, error) {
			t.Fatal("passwordFunc should not be called when the agent has signers")
			return "", nil
		},
		dialAgent: func() (agentSigner, error) {
			return fakeAgent{signers: []ssh.Signer{genSigner(t)}}, nil
		},
	}

	signers, err := ac.collectSigners()
	if err != nil {
		t.Fatalf("collectSigners: %v", err)
	}
	if len(signers) != 1 {
		t.Fatalf("got %d signers, want 1 (from agent)", len(signers))
	}
}

func TestCollectSignersFallsBackWhenAgentUnavailable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519"), genTestKeyPEM(t, ""), 0o600); err != nil {
		t.Fatal(err)
	}

	ac := &AuthChain{
		keyDir:       dir,
		passwordFunc: func(string) (string, error) { return "", nil },
		dialAgent: func() (agentSigner, error) {
			return nil, fmt.Errorf("no agent running")
		},
	}

	signers, err := ac.collectSigners()
	if err != nil {
		t.Fatalf("collectSigners: %v", err)
	}
	if len(signers) != 1 {
		t.Fatalf("got %d signers, want 1 (from key file)", len(signers))
	}
}

func TestCollectSignersFallsBackWhenAgentEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519"), genTestKeyPEM(t, ""), 0o600); err != nil {
		t.Fatal(err)
	}

	ac := &AuthChain{
		keyDir:       dir,
		passwordFunc: func(string) (string, error) { return "", nil },
		dialAgent: func() (agentSigner, error) {
			return fakeAgent{signers: nil}, nil
		},
	}

	signers, err := ac.collectSigners()
	if err != nil {
		t.Fatalf("collectSigners: %v", err)
	}
	if len(signers) != 1 {
		t.Fatalf("got %d signers, want 1 (from key file fallback)", len(signers))
	}
}
