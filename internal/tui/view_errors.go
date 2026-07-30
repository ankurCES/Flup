package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ankurCES/Flup/internal/styles"
)

type errorsView struct{}

func newErrorsView() *errorsView { return &errorsView{} }
func (v *errorsView) Title() string { return "Errors" }
func (v *errorsView) Init() tea.Cmd  { return nil }
func (v *errorsView) Update(_ tea.Msg, _ *Env) (tea.Cmd, bool) {
	return nil, false
}

func (v *errorsView) View(w, h int) string {
	s := lastSnap()
	hdr := styles.CardTitle.Render("Errors & non-2xx codes")

	if s == nil {
		return styles.Card.Width(w - 2).Render(hdr + "\n\n" + styles.Muted.Render("no data yet"))
	}

	// Non-2xx codes
	codes := make([][2]string, 0)
	for code, n := range s.ExactCodes {
		if code/100 == 2 {
			continue
		}
		codes = append(codes, [2]string{fmtCode(code), humanInt(n)})
	}
	sort.Slice(codes, func(i, j int) bool {
		return codes[i][0] < codes[j][0]
	})

	// Network errors
	errs := make([][2]string, 0)
	for e, n := range s.Errors {
		errs = append(errs, [2]string{e, humanInt(n)})
	}
	sort.Slice(errs, func(i, j int) bool {
		return parseInt(errs[i][1]) > parseInt(errs[j][1])
	})

	var body strings.Builder
	body.WriteString(hdr + "\n\n")

	if len(codes) > 0 {
		body.WriteString(styles.Accent.Render("Non-2xx status codes") + "\n")
		for _, c := range codes {
			clr := styles.Warn
			if c[0][0] == '5' {
				clr = styles.Err
			}
			body.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
				clr.Render(fmt.Sprintf("%-6s", c[0])),
				styles.Muted.Render("  "),
				styles.Value.Render(c[1]),
			) + "\n")
		}
	} else {
		body.WriteString(styles.OK.Render("All responses 2xx ✓") + "\n")
	}

	body.WriteString("\n")

	if len(errs) > 0 {
		body.WriteString(styles.Accent.Render("Network / transport errors") + "\n")
		for _, e := range errs {
			body.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
				styles.Muted.Render(fmt.Sprintf("%-20s", e[0])),
				styles.Err.Render(e[1]),
			) + "\n")
		}
	} else {
		body.WriteString(styles.Muted.Render("No transport errors") + "\n")
	}

	return styles.Card.Width(w - 2).Render(body.String())
}

func fmtCode(c int) string {
	return fmtInt(c)
}

// tiny helpers — uses fmt package but kept local to keep imports tidy
func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	var b strings.Builder
	if n < 0 {
		b.WriteByte('-')
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	b.Write(buf[i:])
	return b.String()
}

func parseInt(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// silence unused import warning for tea in case
var _ tea.KeyMsg