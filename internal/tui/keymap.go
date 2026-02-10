package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the TUI.
type KeyMap struct {
	Send       key.Binding // Enter -- send message
	NewLine    key.Binding // Ctrl+J -- insert newline in input
	ScrollUp   key.Binding // Up/k in chat focus
	ScrollDown key.Binding // Down/j in chat focus
	FocusChat  key.Binding // Esc -- switch to chat scrolling
	FocusInput key.Binding // i -- switch to input
	Quit       key.Binding // Ctrl+C
	PermAllow  key.Binding // y -- allow permission
	PermDeny   key.Binding // n -- deny permission
	Tab        key.Binding // Tab -- toggle focus
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send message"),
		),
		NewLine: key.NewBinding(
			key.WithKeys("ctrl+j"),
			key.WithHelp("ctrl+j", "new line"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up/k", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down/j", "scroll down"),
		),
		FocusChat: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "scroll chat"),
		),
		FocusInput: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "focus input"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		PermAllow: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "allow"),
		),
		PermDeny: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "deny"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle focus"),
		),
	}
}
