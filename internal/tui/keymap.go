package tui

// keyBinding is one row of the help screen: the key(s) that trigger an
// action, and a short description of what it does.
type keyBinding struct {
	Keys string
	Desc string
}

// keyBindingGroup is a named block of related keyBindings, shown together
// on the help screen.
type keyBindingGroup struct {
	Title    string
	Bindings []keyBinding
}

var helpGroups = []keyBindingGroup{
	{
		Title: "Browse",
		Bindings: []keyBinding{
			{"j / k", "move cursor down / up"},
			{"h / - / backspace", "go to parent directory"},
			{"l / enter", "open directory"},
			{"Tab", "switch focus between panes"},
			{"y", "yank marked entries (or the one under the cursor)"},
			{"p", "paste the yank into the other pane (transfer)"},
			{"Space", "toggle a mark on the entry under the cursor"},
			{"v", "enter visual mode to extend a selection range"},
			{"/", "incremental fuzzy filter"},
			{"C", "connect, or reconnect, the remote pane"},
			{"?", "show this help"},
			{"q", "quit"},
		},
	},
	{
		Title: "Visual mode",
		Bindings: []keyBinding{
			{"j / k", "extend the selection range"},
			{"y", "yank the selection and return to browse"},
			{"v", "return to browse, keeping the marks"},
			{"esc", "cancel, restoring marks from before visual mode"},
		},
	},
	{
		Title: "Filter",
		Bindings: []keyBinding{
			{"(type)", "narrow the pane to fuzzy-matching entries"},
			{"enter", "keep the filter active, return to browse"},
			{"esc", "clear the filter, return to browse"},
		},
	},
	{
		Title: "Transfer confirm",
		Bindings: []keyBinding{
			{"o", "overwrite any existing destination files in this paste"},
			{"s", "skip any existing destination files in this paste"},
			{"esc", "cancel the paste"},
		},
	},
}
