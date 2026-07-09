// Package sshtest starts an in-process SSH+SFTP server, backed by an
// in-memory filesystem, for integration tests. It exists so integration
// tests can exercise real network round trips (internal/remotefs,
// internal/auth's known_hosts flow) without an external OpenSSH install or
// Docker.
package sshtest

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net"
	"testing"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Default credentials accepted by every Server started by this package.
// These are test-only fixtures, not real secrets.
const (
	User     = "testuser"
	Password = "testpass"
)

// Server is a running in-process SSH+SFTP server.
type Server struct {
	Addr    string
	HostKey ssh.PublicKey
}

// Start launches a server listening on 127.0.0.1 with a random port,
// accepting the fixed User/Password credentials, and serving an in-memory
// filesystem over SFTP. It registers its own shutdown via tb.Cleanup.
func Start(tb testing.TB) *Server {
	tb.Helper()

	signer := generateHostSigner(tb)
	config := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if conn.User() == User && string(password) == Password {
				return nil, nil
			}
			return nil, fmt.Errorf("sshtest: invalid credentials for %q", conn.User())
		},
	}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("sshtest: listen: %v", err)
	}
	tb.Cleanup(func() { _ = listener.Close() })

	go acceptLoop(listener, config)

	return &Server{Addr: listener.Addr().String(), HostKey: signer.PublicKey()}
}

func acceptLoop(listener net.Listener, config *ssh.ServerConfig) {
	for {
		nConn, err := listener.Accept()
		if err != nil {
			return // listener closed by test cleanup
		}
		go handleConn(nConn, config)
	}
}

func handleConn(nConn net.Conn, config *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		return
	}
	defer func() { _ = sshConn.Close() }()
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go handleSession(channel, requests)
	}
}

func handleSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer func() { _ = channel.Close() }()
	for req := range requests {
		if req.Type != "subsystem" || string(req.Payload[4:]) != "sftp" {
			_ = req.Reply(false, nil)
			continue
		}
		_ = req.Reply(true, nil)
		server := sftp.NewRequestServer(channel, sftp.InMemHandler())
		_ = server.Serve()
		_ = server.Close()
		return
	}
}

func generateHostSigner(tb testing.TB) ssh.Signer {
	tb.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		tb.Fatal(err)
	}
	return signer
}
