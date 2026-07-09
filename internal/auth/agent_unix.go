//go:build !windows

package auth

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh/agent"
)

// dialAgent connects to the running ssh-agent via the Unix domain socket
// named by SSH_AUTH_SOCK. A failure here means "no agent available", which
// callers treat as non-fatal, falling back to key files.
func dialAgent() (agent.ExtendedAgent, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, fmt.Errorf("auth: ssh-agent: SSH_AUTH_SOCK not set")
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, fmt.Errorf("auth: ssh-agent: dial %q: %w", sock, err)
	}
	return agent.NewClient(conn), nil
}
