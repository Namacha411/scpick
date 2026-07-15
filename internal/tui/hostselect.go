package tui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"scpick/internal/sshconf"
)

const manualEntryLabel = "(enter host manually)"

func newTextInput(placeholder string, password bool) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	if password {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '*'
	}
	return ti
}

// updateHostSelect handles key input while choosing a host from
// ~/.ssh/config, or the trailing "enter manually" option.
func (m model) updateHostSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	last := len(m.sshHosts) // index of the manual-entry row
	switch msg.String() {
	case "j", "down":
		if m.hostCursor < last {
			m.hostCursor++
		}
	case "k", "up":
		if m.hostCursor > 0 {
			m.hostCursor--
		}
	case "esc":
		m.mode = ModeBrowse
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		if m.hostCursor == last {
			m.manualHost = manualHostForm{step: stepHostname}
			m.textInput = newTextInput("Hostname", false)
			m.mode = ModeManualHost
			return m, nil
		}
		return m.startConnect(m.sshHosts[m.hostCursor])
	}
	return m, nil
}

// updateManualHost drives the three-step (hostname, user, port) wizard for
// entering a host that isn't in ~/.ssh/config.
func (m model) updateManualHost(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = ModeHostSelect
		return m, nil
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		return m.advanceManualHost()
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) advanceManualHost() (tea.Model, tea.Cmd) {
	value := m.textInput.Value()
	switch m.manualHost.step {
	case stepHostname:
		m.manualHost.hostname = value
		m.manualHost.step = stepUser
		m.textInput = newTextInput("User", false)
		return m, nil
	case stepUser:
		m.manualHost.user = value
		m.manualHost.step = stepPort
		m.textInput = newTextInput("Port [22]", false)
		return m, nil
	default: // stepPort
		port := 22
		if value != "" {
			p, err := strconv.Atoi(value)
			if err != nil {
				m.errMsg = fmt.Sprintf("invalid port %q", value)
				return m, nil
			}
			port = p
		}
		host := sshconf.Host{
			Name:     m.manualHost.hostname,
			Hostname: m.manualHost.hostname,
			User:     m.manualHost.user,
			Port:     port,
		}
		return m.startConnect(host)
	}
}

// updatePasswordPrompt collects a masked password/passphrase and hands it
// back to the waiting connect goroutine.
func (m model) updatePasswordPrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		answer := m.passwordAnswer
		m.mode = ModeConnecting
		return m, cancelPassword(answer)
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		answer := m.passwordAnswer
		password := m.textInput.Value()
		m.mode = ModeConnecting
		return m, sendPassword(answer, password)
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// updateHostKeyConfirm asks whether to trust an unknown host key.
func (m model) updateHostKeyConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		answer := m.hostKeyAnswer
		m.mode = ModeConnecting
		return m, sendHostKeyAnswer(answer, true)
	case "n", "N", "esc":
		answer := m.hostKeyAnswer
		m.mode = ModeConnecting
		return m, sendHostKeyAnswer(answer, false)
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}
