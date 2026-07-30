package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap captures every keybinding visible in the bottom help bar.
// Surface them in one struct so we can render a help widget and so users
// can remap them later from the same source of truth.
type KeyMap struct {
	// Global
	Quit key.Binding
	Help key.Binding

	// View switching (1=Run, 2=Live, 3=Summary, 4=Percentiles, 5=Histogram, 6=Errors, 7=History)
	Tab1 key.Binding
	Tab2 key.Binding
	Tab3 key.Binding
	Tab4 key.Binding
	Tab5 key.Binding
	Tab6 key.Binding
	Tab7 key.Binding
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

		Tab1: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "run")),
		Tab2: key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "live")),
		Tab3: key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "summary")),
		Tab4: key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "perc.")),
		Tab5: key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "histo.")),
		Tab6: key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "errors")),
		Tab7: key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "history")),
		Next: key.NewBinding(key.WithKeys("tab", "right"), key.WithHelp("tab", "next")),
		Prev: key.NewBinding(key.WithKeys("shift+tab", "left"), key.WithHelp("S-tab", "prev")),

		FocusNext: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
		FocusPrev: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("S-tab", "prev field")),
		Start:     key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("C-s", "start")),
		Stop:      key.NewBinding(key.WithKeys("ctrl+x", "esc"), key.WithHelp("C-x", "stop")),

		Pause: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause")),

		Open:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Delete: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Back:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),

		ToggleUnits: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "units")),
		ToggleClean: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clean")),
	}
}

// ShortGlobal returns a tight, single-line help snippet for the footer.
func (k KeyMap) ShortGlobal() []key.Binding {
	return []key.Binding{k.Tab1, k.Tab2, k.Tab3, k.Tab4, k.Tab5, k.Tab6, k.Tab7, k.Quit}
}
