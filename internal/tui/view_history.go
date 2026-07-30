package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ankurCES/Flup/internal/bench"
	"github.com/ankurCES/Flup/internal/styles"
)

// historyView lists every saved benchmark. Selecting one shows its
// summary inline below the list.
type historyView struct {
	list   list.Model
	store  *HistoryStore
	loaded HistoryEntry
}

func newHistoryView(store *HistoryStore) *historyView {
	v := &historyView{store: store}
	items := toItems(store.All())
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(styles.Primary).BorderForeground(styles.Primary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(styles.FgColor).BorderForeground(styles.Primary)
	l := list.New(items, delegate, 0, 0)
	l.Title = ""
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.DisableQuitKeybindings()
	v.list = l
	if len(items) > 0 {
		v.loaded = store.All()[0]
	}
	return v
}

func (v *historyView) Title() string { return "History" }
func (v *historyView) Init() tea.Cmd {
	return v.list.StartSpinner()
}

func (v *historyView) Update(msg tea.Msg, env *Env) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		env.Width, env.Height = msg.Width, msg.Height
		v.list.SetSize(msg.Width-4, msg.Height/2)
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := v.list.SelectedItem().(historyItem); ok {
				all := v.store.All()
				for _, e := range all {
					if e.ID == i.id {
						v.loaded = e
						break
					}
				}
				return nil, true
			}
		case "d":
			if i, ok := v.list.SelectedItem().(historyItem); ok {
				if err := v.store.Delete(i.id); err == nil {
					items := toItems(v.store.All())
					v.list.SetItems(items)
					if len(items) == 0 {
						v.loaded = HistoryEntry{}
					}
				}
				return nil, true
			}
		}
	}
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return cmd, false
}

func (v *historyView) View(w, h int) string {
	hdr := styles.CardTitle.Render("Past benchmarks — enter to load, d to delete")
	body := hdr + "\n" + v.list.View()
	if v.loaded.ID != "" {
		body += "\n\n" + renderHistorySummary(v.loaded)
	}
	return styles.Card.Width(w - 2).Render(body)
}

func renderHistorySummary(e HistoryEntry) string {
	s := e.Snapshot
	c := e.Config
	top := lipgloss.JoinHorizontal(lipgloss.Top,
		styles.Muted.Render("URL: "), styles.Value.Render(c.URL),
		styles.Muted.Render("  Method: "), styles.Value.Render(c.Method),
		styles.Muted.Render("  Concurrency: "), styles.Value.Render(itoa(c.Concurrency)),
	)
	elapsed := durShort(s.Elapsed, false)
	mid := lipgloss.JoinHorizontal(lipgloss.Top,
		styles.Muted.Render("Count: "), styles.Value.Render(humanInt(s.Count)),
		styles.Muted.Render("  RPS: "), styles.Value.Render(humanFloat(s.RPS)),
		styles.Muted.Render("  Elapsed: "), styles.Value.Render(elapsed),
		styles.Muted.Render("  Saved: "), styles.Muted.Render(e.Finished.Format("01-02 15:04")),
	)
	pct := make([]string, 0, len(s.Percentiles))
	for _, p := range s.Percentiles {
		pct = append(pct, fmt.Sprintf("P%.2f=%s", p.P*100, durShort(p.Latency, false)))
	}
	bot := styles.Muted.Render("Percentiles: " + strings.Join(pct, "  "))
	return top + "\n" + mid + "\n" + bot
}

// list.Model item wrapper
type historyItem struct {
	id     string
	title  string
	desc   string
}

func (h historyItem) Title() string       { return h.title }
func (h historyItem) Description() string { return h.desc }
func (h historyItem) FilterValue() string { return h.title + " " + h.desc }

func toItems(entries []HistoryEntry) []list.Item {
	out := make([]list.Item, 0, len(entries))
	for _, e := range entries {
		out = append(out, historyItem{
			id:    e.ID,
			title: fmt.Sprintf("%-22s  %s %s", e.Finished.Format("2006-01-02 15:04:05"), e.Config.Method, e.Config.URL),
			desc:  fmt.Sprintf("count=%s rps=%s p50=%s err=%d", humanInt(e.Snapshot.Count), humanFloat(e.Snapshot.RPS), durShortP(&e.Snapshot, 0, false), sumErrMap(e.Snapshot.Errors)),
		})
	}
	return out
}

func sumErrMap(m map[string]int64) int64 { return sumErrors(m) }

// silence unused import warning when bench isn't referenced directly
var _ = bench.Snapshot{}
