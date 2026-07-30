package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ankurCES/Flup/internal/bench"
	"github.com/ankurCES/Flup/internal/history"
	"github.com/ankurCES/Flup/internal/profiles"
	"github.com/ankurCES/Flup/internal/styles"
)

// Tab IDs — keep in lockstep with the keymap.
const (
	TabRun = iota
	TabLive
	TabSummary
	TabPerc
	TabHist
	TabErrors
	TabHistory
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Every(250*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// App is the bubbletea root model. It owns the runner, the history
// store, the active view, and dispatches keys + ticks.
type App struct {
	runner  *Runner
	store   *HistoryStore
	km      KeyMap
	tab     int
	views   []View
	width   int
	height  int
	ready   bool
	export   *exportOverlay
	clip     *clipResult
	profiles *profileOverlay
	profDB   *profiles.Store
}

func NewApp() *App {
	runner := NewRunner()
	store, _ := history.Open()
	a := &App{
		runner: runner,
		store:  store,
		km:     DefaultKeyMap(),
	}
	a.views = []View{
		newRunView(),
		newLiveView(),
		newSummaryView(),
		newPercentilesView(),
		newHistogramView(),
		newErrorsView(),
		newHistoryView(store),
	}
	a.export = newExportOverlay()
	a.profiles = newProfileOverlay()
	a.profDB, _ = profiles.Open()
	return a
}

func (a *App) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(a.views)+1)
	for _, v := range a.views {
		cmds = append(cmds, v.Init())
	}
	cmds = append(cmds, tick())
	return tea.Batch(cmds...)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.ready = true
		env := a.env()
		_, _ = a.views[a.tab].Update(msg, env)
		return a, nil
	case tickMsg:
		// pull latest snapshot into shared state for the read-only views
		s := a.runner.Snapshot()
		setLastSnap(s)
		// refresh live sparkline
		if lv, ok := a.views[TabLive].(*liveView); ok {
			lv.updateRPS(s)
		}
		cmds = append(cmds, tick())
	case tea.KeyMsg:
		// Profile overlay intercepts all keys while active
		if a.profiles.active {
			consumed, loadCfg := a.profiles.handleKey(msg, func() (bench.Config, error) {
				if rv, ok := a.views[TabRun].(*runView); ok {
					return rv.readConfig()
				}
				return bench.Config{}, fmt.Errorf("no run view")
			})
			if loadCfg != nil {
				if rv, ok := a.views[TabRun].(*runView); ok {
					rv.loadConfig(*loadCfg)
				}
			}
			if consumed {
				return a, nil
			}
		}
		// Export overlay intercepts all keys while active
		if a.export.active {
			a.export.handleKey(msg, a.env())
			return a, nil
		}
		switch msg.String() {
		case "ctrl+p":
			// Open profiles overlay on Run tab
			if a.tab == TabRun && a.profDB != nil {
				a.profiles.open(a.profDB)
				return a, nil
			}
		case "y":
			// 'y' (yank) copies results to clipboard on any results tab
			if a.tab != TabRun && a.tab != TabHistory {
				cr := copySnapshotToClipboard(a.runner.Config(), a.runner.Snapshot())
				a.clip = &cr
				return a, nil
			}
		case "e":
			// 'e' opens export on any results tab (not Run/History)
			if a.tab != TabRun && a.tab != TabHistory {
				a.export.toggle()
				return a, nil
			}
		case "q", "ctrl+c":
			a.runner.Stop()
			return a, tea.Quit
		case "1":
			a.tab = TabRun
			a.refreshFocus()
			return a, nil
		case "2":
			a.tab = TabLive
			a.refreshFocus()
			return a, nil
		case "3":
			a.tab = TabSummary
			a.refreshFocus()
			return a, nil
		case "4":
			a.tab = TabPerc
			a.refreshFocus()
			return a, nil
		case "5":
			a.tab = TabHist
			a.refreshFocus()
			return a, nil
		case "6":
			a.tab = TabErrors
			a.refreshFocus()
			return a, nil
		case "7":
			a.tab = TabHistory
			a.refreshFocus()
			return a, nil
		case "tab", "right":
			a.tab = (a.tab + 1) % len(a.views)
			a.refreshFocus()
			return a, nil
		case "shift+tab", "left":
			a.tab = (a.tab - 1 + len(a.views)) % len(a.views)
			a.refreshFocus()
			return a, nil
		case "u":
			setUseSec(!lastUseSec())
			return a, nil
		}
	}

	env := a.env()
	cmd, _ := a.views[a.tab].Update(msg, env)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return a, tea.Batch(cmds...)
}

// refreshFocus blurs inputs on inactive views — they're only really used
// on the Run tab but other views have text inputs hidden so this is a
// no-op safety net.
func (a *App) refreshFocus() {
	if rv, ok := a.views[TabRun].(*runView); ok && a.tab != TabRun {
		for i := range rv.inputs {
			rv.inputs[i].Blur()
			rv.inputs[i].Prompt = "  "
		}
		rv.focus = -1
	}
}

func (a *App) View() string {
	if !a.ready {
		return styles.App.Render("loading…")
	}
	header := a.headerView()
	tabs := a.tabsView()
	footer := a.footerView()
	bodyH := a.height - lipgloss.Height(header) - lipgloss.Height(tabs) - lipgloss.Height(footer) - 1
	if bodyH < 5 {
		bodyH = 5
	}
	body := a.views[a.tab].View(a.width, bodyH)

	// Export overlay
	if a.export.active {
		body = a.export.view(a.width, bodyH)
	}
	// Profile overlay
	if a.profiles.active {
		body = a.profiles.view(a.width, bodyH)
	}

	// Export status line
	if sl := a.export.statusLine(); sl != "" {
		footer = lipgloss.JoinHorizontal(lipgloss.Top, sl, "  ", footer)
	}
	// Profile status line
	if sl := a.profiles.statusLine(); sl != "" {
		footer = lipgloss.JoinHorizontal(lipgloss.Top, sl, "  ", footer)
	}
	// Clipboard status line
	if a.clip != nil {
		if sl := a.clip.statusLine(); sl != "" {
			sty := styles.Err
			if a.clip.ok {
				sty = styles.OK
			}
			footer = lipgloss.JoinHorizontal(lipgloss.Top, sty.Render(sl), "  ", footer)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabs,
		body,
		footer,
	)
}

func (a *App) headerView() string {
	return styles.App.Width(a.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			Logo(),
			"  ",
			styles.Muted.Render("a terminal HTTP load tester"),
		),
	)
}

func (a *App) tabsView() string {
	labels := []string{"Run", "Live", "Summary", "Percentiles", "Histogram", "Errors", "History"}
	parts := make([]string, 0, len(labels))
	for i, l := range labels {
		if i == a.tab {
			parts = append(parts, styles.ActiveTab.Render(l))
		} else {
			parts = append(parts, styles.InactiveTab.Render(l))
		}
	}
	return styles.App.Width(a.width).Render(lipgloss.JoinHorizontal(lipgloss.Top, parts...))
}

func (a *App) footerView() string {
	keys := a.km.ShortGlobal()
	parts := make([]string, 0, len(keys)+2)
	for _, k := range keys {
		parts = append(parts, styles.Tag.Render(k.Help().Key), styles.TagAlt.Render(k.Help().Desc))
	}
	// Export/copy hints on results tabs
	if a.tab != TabRun && a.tab != TabHistory {
		parts = append(parts, styles.Tag.Render("e"), styles.TagAlt.Render("export"))
		parts = append(parts, styles.Tag.Render("y"), styles.TagAlt.Render("copy"))
	}
	// Profile hint on Run tab
	if a.tab == TabRun {
		parts = append(parts, styles.Tag.Render("C-p"), styles.TagAlt.Render("profiles"))
	}
	if a.runner.IsRunning() {
		parts = append(parts, styles.Tag.Render("●"), styles.OK.Render("running"))
	} else if a.runner.Snapshot() != nil {
		parts = append(parts, styles.Tag.Render("■"), styles.Muted.Render("idle"))
	}
	right := fmt.Sprintf(" %d×%d ", a.width, a.height)
	parts = append(parts, styles.Muted.Render(right))
	return styles.App.Width(a.width).Render(lipgloss.JoinHorizontal(lipgloss.Top, parts...))
}

func (a *App) env() *Env {
	return &Env{
		Width:   a.width,
		Height:  a.height,
		Runner:  a.runner,
		Store:   a.store,
		KeyMap:  a.km,
		UseSec:  lastUseSec(),
		NoClean: false,
	}
}
