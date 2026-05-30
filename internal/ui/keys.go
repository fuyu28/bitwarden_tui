package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Tab     key.Binding
	Search  key.Binding
	Filter1 key.Binding
	Filter2 key.Binding
	Filter3 key.Binding
	Filter4 key.Binding
	Filter5 key.Binding
	Copy    key.Binding
	CopyAll key.Binding
	Toggle  key.Binding
	Sync    key.Binding
	Quit    key.Binding
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
	Down:    key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
	Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus")),
	Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Filter1: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "All")),
	Filter2: key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "Login")),
	Filter3: key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "Card")),
	Filter4: key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "Note")),
	Filter5: key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "SSH")),
	Copy:    key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy")),
	CopyAll: key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "copy all")),
	Toggle:  key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle mask")),
	Sync:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "sync")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
