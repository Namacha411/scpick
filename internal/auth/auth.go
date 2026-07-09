// Package auth builds the SSH authentication chain (ssh-agent, then private
// key files, then an interactive password prompt) and verifies host keys
// against ~/.ssh/known_hosts.
package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// agentSigner is the subset of agent.ExtendedAgent that AuthChain needs;
// narrowing it keeps the seam swappable in tests without importing the
// agent package's connection types.
type agentSigner interface {
	Signers() ([]ssh.Signer, error)
}

// AuthChain builds ssh.AuthMethod values in priority order: ssh-agent
// identities first, falling back to the user's default private key files if
// no agent (or no agent identities) are available, and finally an
// interactive password prompt.
type AuthChain struct {
	keyDir       string
	passwordFunc func(prompt string) (string, error)
	dialAgent    func() (agentSigner, error)
}

// NewAuthChain builds an AuthChain rooted at the current user's ~/.ssh
// directory.
func NewAuthChain() (*AuthChain, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("auth: new chain: %w", err)
	}
	return &AuthChain{
		keyDir:       filepath.Join(home, ".ssh"),
		passwordFunc: readPassword,
		dialAgent: func() (agentSigner, error) {
			return dialAgent()
		},
	}, nil
}

// SSHAuthMethods returns the AuthMethods to use in an ssh.ClientConfig,
// in priority order.
func (ac *AuthChain) SSHAuthMethods(user string) []ssh.AuthMethod {
	return []ssh.AuthMethod{
		ssh.PublicKeysCallback(ac.collectSigners),
		ssh.PasswordCallback(func() (string, error) {
			return ac.passwordFunc(fmt.Sprintf("%s's password: ", user))
		}),
	}
}

// collectSigners returns agent identities if any are available, otherwise
// falls back to the user's default private key files (prompting for a
// passphrase per encrypted key, via passwordFunc).
func (ac *AuthChain) collectSigners() ([]ssh.Signer, error) {
	if a, err := ac.dialAgent(); err == nil {
		if signers, err := a.Signers(); err == nil && len(signers) > 0 {
			return signers, nil
		}
	}
	return loadKeyFileSigners(ac.keyDir, ac.passwordFunc)
}

func readPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	defer fmt.Fprintln(os.Stderr)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("auth: read input: %w", err)
	}
	return string(b), nil
}
