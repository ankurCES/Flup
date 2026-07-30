package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ankurCES/Flup/internal/styles"
)

type percentilesView struct{}

func newPercentilesView() *percentilesView { return &percentilesView{} }
func (v *percentilesView) Title() string   { return "Percentiles" }
func (v *percentilesView) Init() tea.Cmd    { return nil }
func (v *percentilesView) Update(_ tea.Msg, _ *Env) (tea.Cmd, bool) {
	return nil, false
}

func (v *percentilesView) View(w, h int) string {
	s := lastSnap()
	hdr := styles.CardTitle.Render("Latency percentiles")

	if s == nil || len(s.Percentiles) == 0 {
		return styles.Card.Width(w - 2).Render(hdr + "\n\n" + styles.Muted.Render("no data yet"))
	}

	// P50..P99.99 in 7 cells; build a 2-row layout
	cells := make([]string, 0, len(s.Percentiles))
	maxLat := s.Percentiles[0].Latency
	for _, p := range s.Percentiles {
		if p.Latency > maxLat {
			maxLat = p.Latency
		}
	}
	for _, p := range s.Percentiles {
		lbl := fmt.Sprintf("P%.2f", p.P*100)
		if p.P == 0.5 {
			lbl = "P50"
		} else if p.P == 0.75 {
			lbl = "P75"
		} else if p.P == 0.9 {
			lbl = "P90"
		} else if p.P == 0.95 {
			lbl = "P95"
		} else if p.P == 0.99 {
			lbl = "P99"
		} else if p.P == 0.999 {
			lbl = "P99.9"
		} else if p.P == 0.9999 {
			lbl = "P99.99"
		}
		bar := styles.Bar(barLen(p.Latency, maxLat, 30), 30, styles.Primary, styles.MutedColor)
		cell := lipgloss.JoinVertical(lipgloss.Left,
			styles.Muted.Render(lbl),
			bar,
			styles.Value.Render(durShort(p.Latency, envUseSec())),
		)
		cells = append(cells, cell)
	}

	// 4 cells per row
	colW := (w - 8) / 4
	rows := make([]string, 0, len(cells)/4+1)
	for i := 0; i < len(cells); i += 4 {
		parts := make([]string, 0, 4)
		for j := 0; j < 4; j++ {
			if i+j >= len(cells) {
				parts = append(parts, "")
				continue
			}
			parts = append(parts, lipgloss.NewStyle().Width(colW).Render(cells[i+j]))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, parts...))
	}
	body := strings.Join([]string{hdr, strings.Join(rows, "\n")}, "\n")
	return styles.Card.Width(w - 2).Render(body)
}

func barLen(v, max time.Duration, width int) int {
	if max <= 0 || width <= 0 {
		return 0
	}
	n := int(int64(v) * int64(width) / int64(max))
	if n < 1 && v > 0 {
		n = 1
	}
	if n > width {
		n = width
	}
	return n
}
