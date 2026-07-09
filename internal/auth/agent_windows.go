//go:build windows

package auth

import (
	"fmt"

	winio "github.com/Microsoft/go-winio"
	"golang.org/x/crypto/ssh/agent"
)

const windowsAgentPipe = `\\.\pipe\openssh-ssh-agent`

// dialAgent connects to the Windows OpenSSH agent service over its named
// pipe. Pageant is not supported. A failure here means "no agent
// available", which callers treat as non-fatal, falling back to key files.
func dialAgent() (agent.ExtendedAgent, error) {
	conn, err := winio.DialPipe(windowsAgentPipe, nil)
	if err != nil {
		return nil, fmt.Errorf("auth: ssh-agent: dial %q: %w", windowsAgentPipe, err)
	}
	return agent.NewClient(conn), nil
}
