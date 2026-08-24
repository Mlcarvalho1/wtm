package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up             key.Binding
	Down           key.Binding
	New            key.Binding
	NewAttach      key.Binding
	Attach         key.Binding
	Remove         key.Binding
	Refresh        key.Binding
	Preview        key.Binding
	NextNeedsInput key.Binding
	Filter         key.Binding
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
	Refresh:        key.NewBinding(key.WithKeys("r")),
	Preview:        key.NewBinding(key.WithKeys("tab")),
	NextNeedsInput: key.NewBinding(key.WithKeys("a")),
	Filter:         key.NewBinding(key.WithKeys("/")),
	Edit:           key.NewBinding(key.WithKeys("e")),
	Quit:           key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Confirm:        key.NewBinding(key.WithKeys("enter")),
	Cancel:         key.NewBinding(key.WithKeys("esc")),
}
