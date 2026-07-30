package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ankurCES/Flup/internal/styles"
)

type histogramView struct{}

func newHistogramView() *histogramView { return &histogramView{} }
func (v *histogramView) Title() string  { return "Histogram" }
func (v *histogramView) Init() tea.Cmd   { return nil }
func (v *histogramView) Update(_ tea.Msg, _ *Env) (tea.Cmd, bool) {
	return nil, false
}

func (v *histogramView) View(w, h int) string {
	if v == nil {
		return ""
	}
	s := lastSnap()
	hdr := styles.CardTitle.Render("Latency histogram")

	if s == nil || len(s.Histograms) == 0 {
		return styles.Card.Width(w - 2).Render(hdr + "\n\n" + styles.Muted.Render("no data yet"))
	}

	// Find total + max for proportions.
	var total int
	maxCount := 0
	for _, b := range s.Histograms {
		total += b.Count
		if b.Count > maxCount {
			maxCount = b.Count
		}
	}

	barW := w - 30
	if barW < 20 {
		barW = 20
	}
	lines := make([]string, 0, len(s.Histograms))
	for _, b := range s.Histograms {
		latStr := durShort(b.Mean, envUseSec())
		countStr := humanInt(int64(b.Count))
		pct := 0.0
		if maxCount > 0 {
			pct = float64(b.Count) / float64(maxCount)
		}
		filled := int(pct * float64(barW))
		bar := styles.Bar(filled, barW, styles.Primary, styles.MutedColor)
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			styles.Muted.Render(fmt.Sprintf("%-10s", latStr)),
			bar,
			" ",
			styles.Value.Render(countStr),
		)
		lines = append(lines, row)
	}

	body := strings.Join([]string{
		hdr,
		styles.Muted.Render(fmt.Sprintf("buckets: %d   total: %s", len(s.Histograms), humanInt(int64(total)))),
		strings.Join(lines, "\n"),
	}, "\n")
	return styles.Card.Width(w - 2).Render(body)
}
