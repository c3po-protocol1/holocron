package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the TUI.
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Quit    key.Binding
	Help    key.Binding
	Refresh key.Binding
	Active  key.Binding
	Enter   key.Binding
	Back    key.Binding
	Top     key.Binding
	Bottom  key.Binding
	Follow  key.Binding
	Group   key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Active: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "toggle active filter"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "open detail"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", "back"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		Follow: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "toggle follow"),
		),
		Group: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "cycle group mode"),
		),
	}
}
