package auth

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ConfirmFunc is asked whether an unknown host key should be trusted. It
// receives the connection hostname and the key's human-readable SHA256
// fingerprint, and returns true to trust (and persist) the key.
type ConfirmFunc func(hostname, fingerprint string) bool

// HostKeyCallback builds an ssh.HostKeyCallback backed by the known_hosts
// file at path (created empty if it doesn't exist yet). A key matching an
// existing entry is accepted silently. An unknown host invokes confirm with
// the key's fingerprint; if confirm returns true, the key is appended to
// path and the connection proceeds. A genuine mismatch against a known
// entry always aborts without consulting confirm, since that can indicate a
// MITM attack.
func HostKeyCallback(path string, confirm ConfirmFunc) (ssh.HostKeyCallback, error) {
	if err := ensureFileExists(path); err != nil {
		return nil, fmt.Errorf("auth: known_hosts: %w", err)
	}
	base, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("auth: known_hosts: %w", err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := base(hostname, remote, key)
		if err == nil {
			return nil
		}

		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) {
			return fmt.Errorf("auth: known_hosts: %w", err)
		}
		if len(keyErr.Want) > 0 {
			return fmt.Errorf("auth: host key mismatch for %s, refusing to connect (possible MITM): %w", hostname, keyErr)
		}

		fingerprint := ssh.FingerprintSHA256(key)
		if !confirm(hostname, fingerprint) {
			return fmt.Errorf("auth: host key for %s was not trusted", hostname)
		}
		return appendKnownHost(path, hostname, key)
	}, nil
}

func ensureFileExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %q: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create %q: %w", filepath.Dir(path), err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create %q: %w", path, err)
	}
	return f.Close()
}

func appendKnownHost(path, hostname string, key ssh.PublicKey) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("auth: known_hosts: append %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(knownhosts.Line([]string{hostname}, key) + "\n"); err != nil {
		return fmt.Errorf("auth: known_hosts: append %q: %w", path, err)
	}
	return nil
}
