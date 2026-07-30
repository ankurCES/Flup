package tui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ankurCES/Flup/internal/bench"
	"github.com/ankurCES/Flup/internal/styles"
)

// summaryView is the "big picture" — RPS, throughput, count, latency
// summary stats. Mimics the format of plow's terminal output, but
// colored and laid out for a TUI.
type summaryView struct{}

func newSummaryView() *summaryView { return &summaryView{} }
func (v *summaryView) Title() string { return "Summary" }
func (v *summaryView) Init() tea.Cmd  { return nil }
func (v *summaryView) Update(_ tea.Msg, _ *Env) (tea.Cmd, bool) { return nil, false }

func (v *summaryView) View(w, h int) string {
	if v == nil {
		return ""
	}
	s := lastSnap()
	hdr := styles.CardTitle.Render("Summary")
	if s == nil {
		return styles.Card.Width(w - 2).Render(hdr + "\n\n" + styles.Muted.Render("no data yet"))
	}

	elapsed := s.Elapsed
	rows := [][2]string{
		{"Elapsed", durShort(elapsed, envUseSec())},
		{"Count", humanInt(s.Count)},
		{"RPS", humanFloat(s.RPS)},
		{"Concurrency", itoa(s.ConcurrencyUsed)},
		{"Reads", fmtMB(s.ReadMBps)},
		{"Writes", fmtMB(s.WriteMBps)},
	}

	// status codes
	codePairs := make([][2]string, 0, len(s.Codes))
	for k, n := range s.Codes {
		codePairs = append(codePairs, [2]string{k, humanInt(n)})
	}
	sort.Slice(codePairs, func(i, j int) bool { return codePairs[i][0] < codePairs[j][0] })
	for _, c := range codePairs {
		rows = append(rows, [2]string{c[0], c[1]})
	}

	// latency summary
	lat := [][2]string{
		{"Latency Min", durShort(s.Latency.Min, envUseSec())},
		{"Latency Mean", durShort(s.Latency.Mean, envUseSec())},
		{"Latency StdDev", durShort(s.Latency.StdDev, envUseSec())},
		{"Latency Max", durShort(s.Latency.Max, envUseSec())},
	}
	// rps summary
	rps := [][2]string{
		{"RPS Min", humanFloat(s.RPSStats.Min)},
		{"RPS Mean", humanFloat(s.RPSStats.Mean)},
		{"RPS StdDev", humanFloat(s.RPSStats.StdDev)},
		{"RPS Max", humanFloat(s.RPSStats.Max)},
	}

	// 3-column layout
	left := labelValGrid(rows)
	mid := labelValGrid(lat)
	right := labelValGrid(rps)

	body := strings.Join([]string{
		hdr,
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width((w-8)/3).Render(left),
			lipgloss.NewStyle().Width((w-8)/3).Render(mid),
			lipgloss.NewStyle().Width((w-8)/3).Render(right),
		),
	}, "\n")

	return styles.Card.Width(w - 2).Render(body)
}

func labelValGrid(rows [][2]string) string {
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(styles.Muted.Render(padR(r[0]+":", 16)))
		b.WriteString(styles.Value.Render(r[1]))
		b.WriteString("\n")
	}
	return b.String()
}

// silence unused imports if bench isn't directly referenced
var _ = bench.Snapshot{}
