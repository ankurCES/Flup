package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ankurCES/Flup/internal/export"
	"github.com/ankurCES/Flup/internal/styles"
)

// exportOverlay is a transient modal that appears when the user presses
// 'e' on any results tab. It lets them pick JSON / CSV / Markdown and
// writes the file.
type exportOverlay struct {
	active   bool
	cursor   int
	formats  []export.Format
	msg      string // status message
	msgTime  time.Time
}

func newExportOverlay() *exportOverlay {
	return &exportOverlay{
		formats: []export.Format{export.JSON, export.CSV, export.Markdown},
	}
}

func (o *exportOverlay) toggle() {
	o.active = !o.active
	o.cursor = 0
}

func (o *exportOverlay) handleKey(msg tea.KeyMsg, env *Env) (consumed bool) {
	if !o.active {
		return false
	}
	switch msg.String() {
	case "esc", "e":
		o.active = false
		return true
	case "up", "k":
		if o.cursor > 0 {
			o.cursor--
		}
		return true
	case "down", "j":
		if o.cursor < len(o.formats)-1 {
			o.cursor++
		}
		return true
	case "enter":
		o.doExport(env)
		return true
	}
	return true // swallow all keys while overlay is open
}

func (o *exportOverlay) doExport(env *Env) {
	snap := env.Runner.Snapshot()
	if snap == nil {
		o.msg = "⚠ no benchmark data to export"
		o.msgTime = time.Now()
		return
	}
	cfg := env.Runner.Config()
	report := export.NewReport(cfg, snap)
	format := o.formats[o.cursor]

	path, err := export.WriteFile(report, format, "")
	if err != nil {
		o.msg = fmt.Sprintf("✗ export failed: %s", err)
	} else {
		o.msg = fmt.Sprintf("✓ exported → %s", path)
	}
	o.msgTime = time.Now()
	o.active = false
}

func (o *exportOverlay) statusLine() string {
	if o.msg == "" {
		return ""
	}
	// fade after 5 seconds
	if time.Since(o.msgTime) > 5*time.Second {
		o.msg = ""
		return ""
	}
	if strings.HasPrefix(o.msg, "✓") {
		return styles.OK.Render(o.msg)
	}
	return styles.Err.Render(o.msg)
}

func (o *exportOverlay) view(w, h int) string {
	if !o.active {
		return ""
	}
	labels := []string{"JSON", "CSV", "Markdown"}
	items := make([]string, len(labels))
	for i, l := range labels {
		if i == o.cursor {
			items[i] = styles.Accent.Render("▸ " + l)
		} else {
			items[i] = styles.Muted.Render("  " + l)
		}
	}

	title := styles.CardTitle.Render("Export results")
	help := styles.Muted.Render("↑/↓ select • enter export • esc cancel")
	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(items, "\n"),
		"",
		help,
	)

	boxW := 40
	if boxW > w-4 {
		boxW = w - 4
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(styles.AccentAlt).
		Padding(1, 2).
		Width(boxW).
		Render(body)

	// Center the overlay
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}
