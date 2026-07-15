// Package remotefs wraps github.com/pkg/sftp to provide one-level-at-a-time
// remote directory browsing and streamed file transfer over SFTP.
package remotefs

import (
	"fmt"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Entry is one file or directory returned by ListDir or Stat.
type Entry struct {
	Name  string
	IsDir bool
	Size  int64
}

// ProgressFunc is called periodically during Download/Upload with the
// number of bytes copied so far and the total size of the file.
type ProgressFunc func(done, total int64)

// Client is a connected SSH+SFTP session to one remote host.
type Client struct {
	ssh  *ssh.Client
	sftp *sftp.Client
}

// Dial establishes an SSH connection to addr (host:port) and opens an SFTP
// session over it, authenticating with methods and verifying the host key
// with hostKeyCB.
func Dial(addr, user string, methods []ssh.AuthMethod, hostKeyCB ssh.HostKeyCallback) (*Client, error) {
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            methods,
		HostKeyCallback: hostKeyCB,
		Timeout:         15 * time.Second,
	}
	sshClient, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("remotefs: dial %s: %w", addr, err)
	}
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, fmt.Errorf("remotefs: open sftp session %s: %w", addr, err)
	}
	return &Client{ssh: sshClient, sftp: sftpClient}, nil
}

// Getwd returns the remote server's initial working directory for this
// session, which OpenSSH's sftp-server (and most SFTP servers) resolves to
// the authenticated user's home directory.
func (c *Client) Getwd() (string, error) {
	dir, err := c.sftp.Getwd()
	if err != nil {
		return "", fmt.Errorf("remotefs: getwd: %w", err)
	}
	return dir, nil
}

// MkdirAll creates path and any missing parent directories on the remote
// host, like os.MkdirAll.
func (c *Client) MkdirAll(path string) error {
	if err := c.sftp.MkdirAll(path); err != nil {
		return fmt.Errorf("remotefs: mkdir %q: %w", path, err)
	}
	return nil
}

// Close shuts down the SFTP session and the underlying SSH connection.
func (c *Client) Close() error {
	sftpErr := c.sftp.Close()
	sshErr := c.ssh.Close()
	if sftpErr != nil {
		return fmt.Errorf("remotefs: close: %w", sftpErr)
	}
	if sshErr != nil {
		return fmt.Errorf("remotefs: close: %w", sshErr)
	}
	return nil
}
