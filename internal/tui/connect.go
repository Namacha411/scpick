package tui

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"scpick/internal/auth"
	"scpick/internal/remotefs"
	"scpick/internal/sshconf"
)

// passwordNeededMsg is emitted by the in-flight connect goroutine when the
// auth chain needs a password or key passphrase.
type passwordNeededMsg struct {
	prompt string
}

// hostKeyNeededMsg is emitted by the in-flight connect goroutine when an
// unknown host key needs to be trusted (or not) before the connection can
// proceed.
type hostKeyNeededMsg struct {
	hostname    string
	fingerprint string
}

// connectResultMsg is emitted once, terminally, when the connect goroutine
// finishes (successfully or not).
type connectResultMsg struct {
	client *remotefs.Client
	err    error
}

// startConnect begins connecting to host in the background, switching into
// ModeConnecting, and returns the commands that launch the worker goroutine
// and start listening for its events.
func (m model) startConnect(host sshconf.Host) (model, tea.Cmd) {
	m.pendingHost = host
	m.mode = ModeConnecting
	m.errMsg = ""
	m.connectEvents = make(chan tea.Msg)
	m.passwordAnswer = make(chan string)
	m.hostKeyAnswer = make(chan bool)

	events := m.connectEvents
	passwordAnswer := m.passwordAnswer
	hostKeyAnswer := m.hostKeyAnswer

	return m, tea.Batch(
		startConnectWorker(host, events, passwordAnswer, hostKeyAnswer),
		waitForConnectEvent(events),
	)
}

// startConnectWorker launches the actual connection attempt in its own
// goroutine and returns immediately, so it never blocks bubbletea's event
// loop. The goroutine communicates back exclusively through events.
func startConnectWorker(host sshconf.Host, events chan tea.Msg, passwordAnswer chan string, hostKeyAnswer chan bool) tea.Cmd {
	return func() tea.Msg {
		go func() {
			passwordFunc := func(prompt string) (string, error) {
				events <- passwordNeededMsg{prompt: prompt}
				pw, ok := <-passwordAnswer
				if !ok {
					return "", fmt.Errorf("password entry cancelled")
				}
				return pw, nil
			}
			confirmHostKey := func(hostname, fingerprint string) bool {
				events <- hostKeyNeededMsg{hostname: hostname, fingerprint: fingerprint}
				return <-hostKeyAnswer
			}

			client, err := dialHost(host, passwordFunc, confirmHostKey)
			events <- connectResultMsg{client: client, err: err}
		}()
		return nil
	}
}

// waitForConnectEvent blocks until the next message arrives on events. It
// must be re-issued (returned again from Update) after every non-terminal
// event, so the model keeps listening for what the connect goroutine sends
// next.
func waitForConnectEvent(events chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-events
	}
}

// sendPassword delivers the user's password entry back to the blocked
// connect goroutine without blocking the UI thread.
func sendPassword(answer chan string, password string) tea.Cmd {
	return func() tea.Msg {
		answer <- password
		return nil
	}
}

// cancelPassword unblocks the connect goroutine's password wait with a
// cancellation, distinct from an empty password.
func cancelPassword(answer chan string) tea.Cmd {
	return func() tea.Msg {
		close(answer)
		return nil
	}
}

// sendHostKeyAnswer delivers the user's trust decision back to the blocked
// connect goroutine without blocking the UI thread.
func sendHostKeyAnswer(answer chan bool, trust bool) tea.Cmd {
	return func() tea.Msg {
		answer <- trust
		return nil
	}
}

// updateConnectEvent handles messages emitted by the in-flight connect
// goroutine, regardless of which prompt mode is currently showing.
func (m model) updateConnectEvent(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case passwordNeededMsg:
		m.passwordPrompt = msg.prompt
		m.mode = ModePasswordPrompt
		m.textInput = newTextInput("", true)
		return m, waitForConnectEvent(m.connectEvents)
	case hostKeyNeededMsg:
		m.hostKeyHostname = msg.hostname
		m.hostKeyFingerprint = msg.fingerprint
		m.mode = ModeHostKeyConfirm
		return m, waitForConnectEvent(m.connectEvents)
	case connectResultMsg:
		m.connectEvents = nil
		m.passwordAnswer = nil
		m.hostKeyAnswer = nil
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.mode = ModeHostSelect
			return m, nil
		}
		if m.remoteClient != nil {
			_ = m.remoteClient.Close()
		}
		m.remoteClient = msg.client
		m.status = fmt.Sprintf("connected to %s", m.pendingHost.Name)
		m.errMsg = ""
		m.mode = ModeBrowse
		if entries, err := listRemoteEntries(m.remoteClient, "/"); err != nil {
			m.errMsg = err.Error()
		} else {
			m.remote.path = "/"
			m.remote.entries = entries
			m.remote.cursor = 0
		}
		return m, nil
	}
	return m, nil
}

// dialHost builds the auth chain and known_hosts verification for host and
// dials it, using passwordFunc and confirmHostKey instead of blocking
// terminal prompts.
func dialHost(host sshconf.Host, passwordFunc func(prompt string) (string, error), confirmHostKey auth.ConfirmFunc) (*remotefs.Client, error) {
	authChain, err := auth.NewAuthChainWithPassword(passwordFunc)
	if err != nil {
		return nil, fmt.Errorf("init auth: %w", err)
	}

	khPath, err := knownHostsPath()
	if err != nil {
		return nil, err
	}
	hostKeyCB, err := auth.HostKeyCallback(khPath, confirmHostKey)
	if err != nil {
		return nil, fmt.Errorf("known_hosts: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", host.Hostname, host.Port)
	client, err := remotefs.Dial(addr, host.User, authChain.SSHAuthMethods(host.User), hostKeyCB)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", addr, err)
	}
	return client, nil
}

func knownHostsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "known_hosts"), nil
}
