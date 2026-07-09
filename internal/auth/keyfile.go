package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

var defaultKeyFiles = []string{"id_ed25519", "id_rsa", "id_ecdsa"}

// loadKeyFileSigners reads each of the default private key files present in
// keyDir, prompting for a passphrase via passwordFunc when one is needed.
// A missing file is skipped silently. A file that exists but can't be used
// (wrong passphrase, unsupported format) is also skipped rather than
// failing the whole chain, since the point is to try every available key.
func loadKeyFileSigners(keyDir string, passwordFunc func(prompt string) (string, error)) ([]ssh.Signer, error) {
	var signers []ssh.Signer
	for _, name := range defaultKeyFiles {
		path := filepath.Join(keyDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("auth: read key %q: %w", path, err)
		}

		if signer, err := ssh.ParsePrivateKey(data); err == nil {
			signers = append(signers, signer)
			continue
		} else if !isPassphraseMissing(err) {
			continue
		}

		passphrase, err := passwordFunc(fmt.Sprintf("Passphrase for %s: ", path))
		if err != nil {
			return nil, fmt.Errorf("auth: read passphrase: %w", err)
		}
		signer, err := ssh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
		if err != nil {
			continue
		}
		signers = append(signers, signer)
	}
	return signers, nil
}

func isPassphraseMissing(err error) bool {
	var passphraseMissing *ssh.PassphraseMissingError
	return errors.As(err, &passphraseMissing)
}
