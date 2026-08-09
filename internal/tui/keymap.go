package tui

// KeyBinding is one row of the help screen: the key(s) that trigger an
// action, and a short description of what it does.
type KeyBinding struct {
	Keys string
	Desc string
}

// KeyBindingGroup is a named block of related KeyBindings, shown together
// on the help screen. Exported so cmd/scpick can reuse it for `scpick --help`.
type KeyBindingGroup struct {
	Title    string
	Bindings []KeyBinding
}

// HelpGroups is the single source of truth for scpick's keybindings, used by
// both the in-TUI `?` screen and `scpick --help`.
var HelpGroups = []KeyBindingGroup{
	{
		Title: "Browse",
		Bindings: []KeyBinding{
			{"j / k", "move cursor down / up"},
			{"h / - / backspace", "go to parent directory"},
			{"l / enter", "open directory"},
			{"Tab", "switch focus between panes"},
			{"y", "yank marked entries (or the one under the cursor)"},
			{"p", "paste the yank into the other pane (transfer)"},
			{"Space", "toggle a mark on the entry under the cursor (un-yanks it too if already yanked)"},
			{"v", "enter visual mode to extend or retract a selection range"},
			{"/", "incremental fuzzy filter"},
			{"C", "connect, or reconnect, the remote pane"},
			{"?", "show this help"},
			{"q", "quit"},
		},
	},
	{
		Title: "Visual mode",
		Bindings: []KeyBinding{
			{"j / k", "extend the selection range"},
			{"y", "yank the selection and return to browse"},
			{"v", "return to browse, keeping the marks"},
			{"esc", "cancel, restoring marks from before visual mode"},
		},
	},
	{
		Title: "Filter",
		Bindings: []KeyBinding{
			{"(type)", "narrow the pane to fuzzy-matching entries"},
			{"enter", "keep the filter active, return to browse"},
			{"esc", "clear the filter, return to browse"},
		},
	},
	{
		Title: "Transfer confirm",
		Bindings: []KeyBinding{
			{"o", "overwrite this (and every later) existing destination file in the paste"},
			{"s", "skip this (and every later) existing destination file in the paste"},
			{"enter", "keep both: rename this (and every later) file with a numbered suffix"},
			{"esc", "skip this and every remaining conflicting file"},
		},
	},
}
