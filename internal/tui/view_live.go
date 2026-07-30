package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ankurCES/Flup/internal/bench"
	"github.com/ankurCES/Flup/internal/styles"
)

// liveView shows a spark-style line of the last N RPS samples plus the
// rolling 1-second latency/Codes/throughput. It's the "now" view.
type liveView struct {
	paused bool
	rps    []float64 // last ~80 samples
	maxRPS float64
}

func newLiveView() *liveView { return &liveView{rps: make([]float64, 0, 80)} }

func (v *liveView) Title() string { return "Live" }
func (v *liveView) Init() tea.Cmd  { return nil }

func (v *liveView) Update(msg tea.Msg, env *Env) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "p":
			v.paused = !v.paused
			return nil, true
		}
	}
	return nil, false
}

// updateRPS appends a sample; called by the app tick.
func (v *liveView) updateRPS(s *bench.Snapshot) {
	if v.paused || s == nil {
		return
	}
	v.rps = append(v.rps, s.RPS)
	if len(v.rps) > 80 {
		v.rps = v.rps[len(v.rps)-80:]
	}
	if s.RPS > v.maxRPS {
		v.maxRPS = s.RPS
	}
}

func (v *liveView) View(w, h int) string {
	if v == nil {
		return ""
	}
	s := currentSnap()
	hdr := styles.CardTitle.Render("Live — last 1s rollup")
	if s == nil {
		return styles.Card.Width(w - 2).Render(hdr + "\n\n" + styles.Muted.Render("no data yet"))
	}

	// header metrics
	metrics := kvGrid([][2]string{
		{"RPS", humanFloat(s.RPS)},
		{"Count", humanInt(s.Count)},
		{"Elapsed", durShort(s.Elapsed, envUseSec())},
		{"Concurrency", intStr(s.ConcurrencyUsed)},
		{"Reads", fmtMB(s.ReadMBps)},
		{"Writes", fmtMB(s.WriteMBps)},
	}, 3, w-8)

	// sparkline
	sparkW := w - 8
	if sparkW < 20 {
		sparkW = 20
	}
	spark := renderSpark(v.rps, sparkW, 8)

	// status codes
	codeRows := make([]string, 0, 6)
	for _, k := range []string{"1xx", "2xx", "3xx", "4xx", "5xx", "other"} {
		c := s.Codes[k]
		if c == 0 {
			continue
		}
		clr := styles.OK
		switch k {
		case "4xx":
			clr = styles.Warn
		case "5xx":
			clr = styles.Err
		}
		codeRows = append(codeRows, lipgloss.JoinHorizontal(lipgloss.Top,
			clr.Render(k), styles.Muted.Render(" "), styles.Value.Render(humanInt(c)),
		))
	}
	codes := styles.Muted.Render("Codes: ")
	if len(codeRows) == 0 {
		codes += styles.Muted.Render("—")
	} else {
		codes += strings.Join(codeRows, styles.Muted.Render("  "))
	}

	body := strings.Join([]string{
		hdr,
		metrics,
		"",
		styles.Label.Render("RPS spark (last 80 samples):"),
		spark,
		"",
		codes,
		"",
		styles.Muted.Render("p: pause spark"),
	}, "\n")

	return styles.Card.Width(w - 2).Render(body)
}

// helpers consumed by View funcs in different files
func currentSnap() *bench.Snapshot {
	// read from a package-level pointer maintained by the app tick
	return lastSnap()
}
func envUseSec() bool { return lastUseSec() }
func intStr(n int) string {
	if n == 0 {
		return "—"
	}
	return itoa(n)
}
func fmtMB(f float64) string {
	return humanFloat(f) + " MB/s"
}

// kvGrid lays out (key,value) pairs in `cols` columns with even gutters.
func kvGrid(kv [][2]string, cols, width int) string {
	colW := width / cols
	if colW < 12 {
		colW = 12
	}
	rows := make([]string, 0)
	for i := 0; i < len(kv); i += cols {
		parts := make([]string, 0, cols)
		for j := 0; j < cols; j++ {
			if i+j >= len(kv) {
				parts = append(parts, "")
				continue
			}
			row := lipgloss.JoinHorizontal(lipgloss.Top,
				styles.Muted.Render(kv[i+j][0]+": "),
				styles.Value.Render(kv[i+j][1]),
			)
			parts = append(parts, lipgloss.NewStyle().Width(colW).Render(row))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, parts...))
	}
	return strings.Join(rows, "\n")
}

// renderSpark draws a fixed-height sparkline using block glyphs.
func renderSpark(samples []float64, w, h int) string {
	if len(samples) == 0 {
		return styles.Muted.Render(strings.Repeat("·", w))
	}
	// pad left so newest is on the right
	if len(samples) < w {
		left := strings.Repeat(" ", w-len(samples))
		pad := make([]float64, w-len(samples))
		samples = append(pad, samples...)
		_ = left
	} else if len(samples) > w {
		samples = samples[len(samples)-w:]
	}
	max := samples[0]
	for _, v := range samples {
		if v > max {
			max = v
		}
	}
	if max == 0 {
		max = 1
	}
	glyphs := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	out := make([]string, 0, w)
	for _, v := range samples {
		idx := int(float64(len(glyphs)-1) * v / max)
		if idx < 0 {
			idx = 0
		}
		if idx >= len(glyphs) {
			idx = len(glyphs) - 1
		}
		out = append(out, styles.Accent.Render(glyphs[idx]))
	}
	return strings.Join(out, "")
}
