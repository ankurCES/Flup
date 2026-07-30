package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap captures every keybinding visible in the bottom help bar.
// Surface them in one struct so we can render a help widget and so users
// can remap them later from the same source of truth.
type KeyMap struct {
	// Global
	Quit key.Binding
	Help key.Binding

	// View switching (tab/shift+tab/arrow keys)
	Next key.Binding
	Prev key.Binding

	// Run screen
	FocusNext key.Binding
	FocusPrev key.Binding
	Start     key.Binding
	Stop      key.Binding

	// Live chart
	Pause key.Binding

	// History screen
	Open key.Binding
	Delete key.Binding
	Back   key.Binding

	// View toggles
	ToggleUnits key.Binding // seconds vs duration
	ToggleClean key.Binding
}

// Default returns the production keybindings. Centralized so help text
// and bindings can never drift apart.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),

		Next: key.NewBinding(key.WithKeys("tab", "right"), key.WithHelp("→/tab", "next tab")),
		Prev: key.NewBinding(key.WithKeys("shift+tab", "left"), key.WithHelp("←/S-tab", "prev tab")),

		FocusNext: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
		FocusPrev: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("S-tab", "prev field")),
		Start:     key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("C-s", "start")),
		Stop:      key.NewBinding(key.WithKeys("ctrl+x", "esc"), key.WithHelp("C-x", "stop")),

		Pause: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause")),

		Open:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Delete: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Back:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),

		ToggleUnits: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "units")),
		ToggleClean: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clean")),
	}
}

// ShortGlobal returns a tight, single-line help snippet for the footer.
func (k KeyMap) ShortGlobal() []key.Binding {
	return []key.Binding{k.Prev, k.Next, k.Quit}
}
