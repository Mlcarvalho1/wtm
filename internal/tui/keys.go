package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up             key.Binding
	Down           key.Binding
	New            key.Binding
	NewAttach      key.Binding
	Attach         key.Binding
	Remove         key.Binding
	Merge          key.Binding
	Refresh        key.Binding
	Tab            key.Binding
	NextNeedsInput key.Binding
	Filter         key.Binding
	Palette        key.Binding
	Help           key.Binding
	Edit           key.Binding
	Quit           key.Binding
	Confirm        key.Binding
	Cancel         key.Binding
}

var keys = keyMap{
	Up:             key.NewBinding(key.WithKeys("k", "up")),
	Down:           key.NewBinding(key.WithKeys("j", "down")),
	New:            key.NewBinding(key.WithKeys("n")),
	NewAttach:      key.NewBinding(key.WithKeys("N")),
	Attach:         key.NewBinding(key.WithKeys("enter")),
	Remove:         key.NewBinding(key.WithKeys("x")),
	Merge:          key.NewBinding(key.WithKeys("m")),
	Refresh:        key.NewBinding(key.WithKeys("r")),
	Tab:            key.NewBinding(key.WithKeys("tab")),
	NextNeedsInput: key.NewBinding(key.WithKeys("a")),
	Filter:         key.NewBinding(key.WithKeys("/")),
	Palette:        key.NewBinding(key.WithKeys(":")),
	Help:           key.NewBinding(key.WithKeys("?")),
	Edit:           key.NewBinding(key.WithKeys("e")),
	Quit:           key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Confirm:        key.NewBinding(key.WithKeys("enter")),
	Cancel:         key.NewBinding(key.WithKeys("esc")),
}

// keybindHelp is the ordered [key, label] list shown in the '?' overlay —
// the terminal equivalent of the design's KEYBINDS table.
var keybindHelp = [][2]string{
	{"j / k", "move cursor"},
	{"enter", "attach session"},
	{"esc esc", "detach (or ctrl-b d)"},
	{"tab", "cycle pane"},
	{"n", "new worktree"},
	{"N", "new + launch claude"},
	{"m", "merge into base"},
	{"x", "remove worktree"},
	{"s", "diff: toggle split/unified view"},
	{"[ / ]", "diff: prev/next file"},
	{"a", "next needs-input"},
	{"/", "filter list"},
	{":", "command palette"},
	{"e", "open in $EDITOR"},
	{"r", "refresh git status"},
	{"?", "this panel"},
	{"q", "quit (sessions persist)"},
}
