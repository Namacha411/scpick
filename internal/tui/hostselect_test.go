package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"scpick/internal/sshconf"
)

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	case "down":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestUpdateHostSelectCursorMovementIsBounded(t *testing.T) {
	m := NewModel()
	m.mode = ModeHostSelect
	m.sshHosts = []sshconf.Host{{Name: "a"}, {Name: "b"}}

	newModel, _ := m.updateHostSelect(keyMsg("down"))
	got := newModel.(model)
	if got.hostCursor != 1 {
		t.Fatalf("hostCursor = %d, want 1", got.hostCursor)
	}

	// down past the manual-entry row (index len(hosts)) must not overshoot.
	got, _ = mustModel(got.updateHostSelect(keyMsg("down")))
	got, _ = mustModel(got.updateHostSelect(keyMsg("down")))
	got, _ = mustModel(got.updateHostSelect(keyMsg("down")))
	if got.hostCursor != len(m.sshHosts) {
		t.Fatalf("hostCursor = %d, want %d (manual-entry row)", got.hostCursor, len(m.sshHosts))
	}

	got, _ = mustModel(got.updateHostSelect(keyMsg("up")))
	if got.hostCursor != len(m.sshHosts)-1 {
		t.Fatalf("hostCursor = %d, want %d", got.hostCursor, len(m.sshHosts)-1)
	}
}

func mustModel(tm tea.Model, cmd tea.Cmd) (model, tea.Cmd) {
	return tm.(model), cmd
}

func TestUpdateHostSelectEnterOnManualRowEntersManualHostMode(t *testing.T) {
	m := NewModel()
	m.mode = ModeHostSelect
	m.sshHosts = []sshconf.Host{{Name: "a"}}
	m.hostCursor = 1 // the manual-entry row

	newModel, _ := m.updateHostSelect(keyMsg("enter"))
	got := newModel.(model)
	if got.mode != ModeManualHost {
		t.Fatalf("mode = %v, want ModeManualHost", got.mode)
	}
	if got.manualHost.step != stepHostname {
		t.Fatalf("manualHost.step = %v, want stepHostname", got.manualHost.step)
	}
}

func TestUpdateHostSelectEscReturnsToBrowse(t *testing.T) {
	m := NewModel()
	m.mode = ModeHostSelect
	newModel, _ := m.updateHostSelect(keyMsg("esc"))
	got := newModel.(model)
	if got.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse", got.mode)
	}
}

func TestManualHostWizardCollectsAllThreeFieldsThenConnects(t *testing.T) {
	m := NewModel()
	m.mode = ModeManualHost
	m.manualHost = manualHostForm{step: stepHostname}
	m.textInput = newTextInput("Hostname", false)

	m.textInput.SetValue("example.com")
	newModel, _ := m.updateManualHost(keyMsg("enter"))
	got := newModel.(model)
	if got.manualHost.step != stepUser {
		t.Fatalf("step = %v, want stepUser", got.manualHost.step)
	}
	if got.manualHost.hostname != "example.com" {
		t.Fatalf("hostname = %q, want %q", got.manualHost.hostname, "example.com")
	}

	got.textInput.SetValue("alice")
	newModel, _ = got.updateManualHost(keyMsg("enter"))
	got = newModel.(model)
	if got.manualHost.step != stepPort {
		t.Fatalf("step = %v, want stepPort", got.manualHost.step)
	}

	got.textInput.SetValue("2222")
	newModel, cmd := got.updateManualHost(keyMsg("enter"))
	got = newModel.(model)
	if got.mode != ModeConnecting {
		t.Fatalf("mode = %v, want ModeConnecting", got.mode)
	}
	if got.pendingHost.Hostname != "example.com" || got.pendingHost.User != "alice" || got.pendingHost.Port != 2222 {
		t.Fatalf("pendingHost = %+v, want {example.com alice 2222}", got.pendingHost)
	}
	if cmd == nil {
		t.Fatal("expected startConnect to return a command batch")
	}
}

func TestManualHostWizardDefaultsPortTo22WhenBlank(t *testing.T) {
	m := NewModel()
	m.mode = ModeManualHost
	m.manualHost = manualHostForm{step: stepPort, hostname: "h", user: "u"}
	m.textInput = newTextInput("Port [22]", false)

	newModel, _ := m.updateManualHost(keyMsg("enter"))
	got := newModel.(model)
	if got.pendingHost.Port != 22 {
		t.Fatalf("Port = %d, want 22", got.pendingHost.Port)
	}
}

func TestManualHostWizardRejectsInvalidPort(t *testing.T) {
	m := NewModel()
	m.mode = ModeManualHost
	m.manualHost = manualHostForm{step: stepPort, hostname: "h", user: "u"}
	m.textInput = newTextInput("Port [22]", false)
	m.textInput.SetValue("not-a-number")

	newModel, _ := m.updateManualHost(keyMsg("enter"))
	got := newModel.(model)
	if got.mode != ModeManualHost {
		t.Fatalf("mode = %v, want ModeManualHost (should stay put on invalid port)", got.mode)
	}
	if got.errMsg == "" {
		t.Fatal("expected an error message for an invalid port")
	}
}

func TestUpdatePasswordPromptEnterSendsPasswordAndSwitchesToConnecting(t *testing.T) {
	m := NewModel()
	m.mode = ModePasswordPrompt
	m.passwordAnswer = make(chan string)
	m.textInput = newTextInput("", true)
	m.textInput.SetValue("hunter2")

	newModel, cmd := m.updatePasswordPrompt(keyMsg("enter"))
	got := newModel.(model)
	if got.mode != ModeConnecting {
		t.Fatalf("mode = %v, want ModeConnecting", got.mode)
	}

	done := make(chan string, 1)
	go func() { done <- <-m.passwordAnswer }()
	cmd()
	select {
	case pw := <-done:
		if pw != "hunter2" {
			t.Fatalf("password sent = %q, want %q", pw, "hunter2")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for password to be delivered")
	}
}

func TestUpdatePasswordPromptEscCancelsWithoutSubmittingBlankPassword(t *testing.T) {
	m := NewModel()
	m.mode = ModePasswordPrompt
	m.passwordAnswer = make(chan string)

	newModel, cmd := m.updatePasswordPrompt(keyMsg("esc"))
	got := newModel.(model)
	if got.mode != ModeConnecting {
		t.Fatalf("mode = %v, want ModeConnecting", got.mode)
	}

	done := make(chan bool, 1)
	go func() {
		_, ok := <-m.passwordAnswer
		done <- ok
	}()
	cmd()
	select {
	case ok := <-done:
		if ok {
			t.Fatal("expected the password channel to be closed (cancelled), not sent a value")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for cancellation")
	}
}

func TestUpdateHostKeyConfirmTrustAndReject(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want bool
	}{
		{"y", true},
		{"n", false},
		{"esc", false},
	} {
		m := NewModel()
		m.mode = ModeHostKeyConfirm
		m.hostKeyAnswer = make(chan bool)

		newModel, cmd := m.updateHostKeyConfirm(keyMsg(tc.key))
		got := newModel.(model)
		if got.mode != ModeConnecting {
			t.Fatalf("key %q: mode = %v, want ModeConnecting", tc.key, got.mode)
		}

		done := make(chan bool, 1)
		go func() { done <- <-m.hostKeyAnswer }()
		cmd()
		select {
		case answer := <-done:
			if answer != tc.want {
				t.Fatalf("key %q: answer = %v, want %v", tc.key, answer, tc.want)
			}
		case <-time.After(time.Second):
			t.Fatalf("key %q: timed out waiting for answer", tc.key)
		}
	}
}
