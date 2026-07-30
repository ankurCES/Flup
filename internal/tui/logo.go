package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ankurCES/Flup/internal/styles"
)

// logoBlock is the multi-line ASCII logo for Flup. It's drawn in the
// header banner and the README. Keep it small — terminal width matters.
var logoBlock = strings.Join([]string{
	` _______  __    _  _______  _______  _______ `,
	`|   _  ||  |  | ||   _   ||       ||       |`,
	`|  |_| ||   |_| ||  |_|  ||_     _||   _   |`,
	`|       ||       ||       |  |   |  |  |_|  |`,
	`|       ||  _    ||       |  |   |  |       |`,
	`|   _   || | |   ||   _   |  |   |  |   _   |`,
	`|__| |__||_|  |__||__| |__|  |___|  |__| |__|`,
}, "\n")

// Logo returns the colored logo block, suitable for ~46-col wide banners.
func Logo() string {
	return lipgloss.NewStyle().Foreground(styles.Primary).Bold(true).Render(logoBlock)
}

// Banner is the header strip rendered at the top of every Flup screen:
// logo + tagline + version pills.
func Banner(version string, w int) string {
	logo := Logo()
	tag := styles.Tag.Render("LIVE TUI")
	ver := styles.TagAlt.Render("flup " + version)
	right := lipgloss.JoinHorizontal(lipgloss.Center, tag, " ", ver)
	row := lipgloss.JoinHorizontal(lipgloss.Top, logo, "  ", right)
	return styles.Header.Width(w).Render(row)
}
