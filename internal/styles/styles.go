// Package styles defines the lipgloss palette and reusable layout primitives
// for the Flup TUI. Keeping them in one file makes the visual language easy
// to tweak globally.
package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Palette
	accent    = lipgloss.Color("#7D56F4") // electric violet
	accentAlt = lipgloss.Color("#FF6AC6") // hot pink
	ok        = lipgloss.Color("#A8E6CF") // mint
	warn      = lipgloss.Color("#FFD86E") // amber
	err       = lipgloss.Color("#FF5C7A") // coral red
	muted     = lipgloss.Color("#7A7E8C") // slate
	fg        = lipgloss.Color("#E8E8F0") // near-white
	bg        = lipgloss.Color("#16161D") // deep slate
)

var (
	// Frame styles
	App = lipgloss.NewStyle().
		Padding(0, 1)

	Header = lipgloss.NewStyle().
		Foreground(fg).
		Background(bg).
		Bold(true).
		Padding(0, 1).
		MarginBottom(1)

	Footer = lipgloss.NewStyle().
		Foreground(muted).
		Padding(0, 1).
		MarginTop(1)

	// Tabs
	ActiveTab = lipgloss.NewStyle().
			Foreground(bg).
			Background(accent).
			Bold(true).
			Padding(0, 2).
			MarginRight(1)

	InactiveTab = lipgloss.NewStyle().
			Foreground(fg).
			Background(lipgloss.Color("#26263A")).
			Padding(0, 2).
			MarginRight(1)

	TabGap = lipgloss.NewStyle().MarginRight(2)

	// Cards / panels
	Card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2).
		MarginBottom(1)

	CardTitle = lipgloss.NewStyle().
			Foreground(accentAlt).
			Bold(true).
			MarginBottom(1)

	// Text variants
	Label  = lipgloss.NewStyle().Foreground(muted)
	Value  = lipgloss.NewStyle().Foreground(fg).Bold(true)
	OK     = lipgloss.NewStyle().Foreground(ok).Bold(true)
	Warn   = lipgloss.NewStyle().Foreground(warn).Bold(true)
	Err    = lipgloss.NewStyle().Foreground(err).Bold(true)
	Muted  = lipgloss.NewStyle().Foreground(muted)
	Accent = lipgloss.NewStyle().Foreground(accent).Bold(true)

	// Title / big text
	BigTitle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true).
			Padding(0, 1)

	Tag = lipgloss.NewStyle().
		Foreground(bg).
		Background(accent).
		Bold(true).
		Padding(0, 1)

	TagAlt = lipgloss.NewStyle().
		Foreground(fg).
		Background(lipgloss.Color("#26263A")).
		Padding(0, 1)

	// Public aliases used by other packages.
	Primary    = accent
	Secondary  = accentAlt
	AccentAlt  = accentAlt
	OKColor    = ok
	WarnColor  = warn
	ErrColor   = err
	MutedColor = muted
	FgColor    = fg
	BgColor    = bg

	// Status bar
	StatusBar = lipgloss.NewStyle().
			Foreground(fg).
			Background(lipgloss.Color("#1F1F2E")).
			Padding(0, 1)
)

// Bar returns a unicode bar of `width` blocks, `filled` blocks in `fg`,
// the rest in `bg`. Used by the latency histogram and percentiles views.
func Bar(filled, width int, fg, bg lipgloss.Color) string {
	if width <= 0 {
		return ""
	}
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	blk := lipgloss.NewStyle().Foreground(fg).Render
	dim := lipgloss.NewStyle().Foreground(bg).Render
	return blk(strings.Repeat("█", filled)) + dim(strings.Repeat("░", width-filled))
}
