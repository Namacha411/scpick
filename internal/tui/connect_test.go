package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"scpick/internal/sshconf"
)

func TestUpdateConnectEventPasswordNeededSwitchesMode(t *testing.T) {
	m := NewModel()
	m.mode = ModeConnecting
	m.connectEvents = make(chan tea.Msg)

	newModel, cmd := m.updateConnectEvent(passwordNeededMsg{prompt: "alice's password: "})
	got := newModel.(model)
	if got.mode != ModePasswordPrompt {
		t.Fatalf("mode = %v, want ModePasswordPrompt", got.mode)
	}
	if got.passwordPrompt != "alice's password: " {
		t.Fatalf("passwordPrompt = %q", got.passwordPrompt)
	}
	if cmd == nil {
		t.Fatal("expected updateConnectEvent to re-arm the event listener")
	}
}

func TestUpdateConnectEventHostKeyNeededSwitchesMode(t *testing.T) {
	m := NewModel()
	m.mode = ModeConnecting
	m.connectEvents = make(chan tea.Msg)

	newModel, cmd := m.updateConnectEvent(hostKeyNeededMsg{hostname: "example.com", fingerprint: "SHA256:abc"})
	got := newModel.(model)
	if got.mode != ModeHostKeyConfirm {
		t.Fatalf("mode = %v, want ModeHostKeyConfirm", got.mode)
	}
	if got.hostKeyHostname != "example.com" || got.hostKeyFingerprint != "SHA256:abc" {
		t.Fatalf("hostKeyHostname/Fingerprint = %q/%q", got.hostKeyHostname, got.hostKeyFingerprint)
	}
	if cmd == nil {
		t.Fatal("expected updateConnectEvent to re-arm the event listener")
	}
}

func TestUpdateConnectEventErrorReturnsToHostSelect(t *testing.T) {
	m := NewModel()
	m.mode = ModeConnecting
	m.pendingHost = sshconf.Host{Name: "example"}

	newModel, cmd := m.updateConnectEvent(connectResultMsg{err: errors.New("boom")})
	got := newModel.(model)
	if got.mode != ModeHostSelect {
		t.Fatalf("mode = %v, want ModeHostSelect", got.mode)
	}
	if got.errMsg != "boom" {
		t.Fatalf("errMsg = %q, want %q", got.errMsg, "boom")
	}
	if cmd != nil {
		t.Fatal("expected no further listening command on terminal error")
	}
}

// The success path of updateConnectEvent (connectResultMsg with a non-nil
// client) also lists the remote's initial directory, which needs a real
// *remotefs.Client; it is covered by TestIntegrationUpdateConnectEventSuccess
// in integration_test.go instead of here.
