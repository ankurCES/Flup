package tui

import "github.com/charmbracelet/bubbletea"

// View is the contract every screen in Flup implements. The root model
// forwards Init/Update/View through whichever view is active.
type View interface {
	Init() tea.Cmd
	Update(msg tea.Msg, env *Env) (tea.Cmd, bool) // bool = handled (consumed)
	View(w, h int) string
	Title() string
}

// Env is the shared environment passed to every view:
// runtime state (running, paused, snapshot), the runner (to start/stop),
// and any view-shared services like persistence.
type Env struct {
	Width   int
	Height  int
	Runner  *Runner
	Store   *HistoryStore
	KeyMap  KeyMap
	UseSec  bool
	NoClean bool
}
