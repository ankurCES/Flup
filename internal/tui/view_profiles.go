package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ankurCES/Flup/internal/bench"
	"github.com/ankurCES/Flup/internal/profiles"
	"github.com/ankurCES/Flup/internal/styles"
)

// profileMode indicates what the overlay is doing.
type profileMode int

const (
	pmBrowse profileMode = iota // list / load / delete
	pmSave                      // save current config with a name
)

// profileOverlay is a modal that appears on the Run tab for saving and
// loading benchmark presets.
type profileOverlay struct {
	active  bool
	mode    profileMode
	store   *profiles.Store
	items   []*profiles.Profile
	cursor  int
	nameIn  textinput.Model
	msg     string
	msgTime time.Time
}

func newProfileOverlay() *profileOverlay {
	ti := textinput.New()
	ti.Placeholder = "my-api-load-test"
	ti.CharLimit = 64
	ti.Width = 30
	ti.PromptStyle = styles.Accent
	ti.TextStyle = styles.Value
	return &profileOverlay{nameIn: ti}
}

func (o *profileOverlay) open(store *profiles.Store) {
	o.active = true
	o.mode = pmBrowse
	o.store = store
	o.refresh()
	o.cursor = 0
	o.nameIn.SetValue("")
	o.nameIn.Blur()
}

func (o *profileOverlay) refresh() {
	if o.store != nil {
		o.items = o.store.List()
	}
}

func (o *profileOverlay) close() {
	o.active = false
	o.nameIn.Blur()
}

// handleKey processes input; returns (consumed, loadCfg). If loadCfg is
// non-nil the caller should populate the run view fields from it.
func (o *profileOverlay) handleKey(msg tea.KeyMsg, currentCfg func() (bench.Config, error)) (bool, *bench.Config) {
	if !o.active {
		return false, nil
	}
	switch o.mode {
	case pmBrowse:
		return o.handleBrowse(msg, currentCfg)
	case pmSave:
		return o.handleSave(msg, currentCfg)
	}
	return true, nil
}

func (o *profileOverlay) handleBrowse(msg tea.KeyMsg, currentCfg func() (bench.Config, error)) (bool, *bench.Config) {
	switch msg.String() {
	case "esc":
		o.close()
		return true, nil
	case "up", "k":
		if o.cursor > 0 {
			o.cursor--
		}
		return true, nil
	case "down", "j":
		if o.cursor < len(o.items)-1 {
			o.cursor++
		}
		return true, nil
	case "enter":
		// Load selected profile
		if o.cursor < len(o.items) {
			cfg := o.items[o.cursor].Config
			o.setMsg("✓ loaded: " + o.items[o.cursor].Name)
			o.close()
			return true, &cfg
		}
		return true, nil
	case "d":
		// Delete selected profile
		if o.cursor < len(o.items) {
			name := o.items[o.cursor].Name
			if err := o.store.Delete(name); err != nil {
				o.setMsg("✗ " + err.Error())
			} else {
				o.setMsg("✓ deleted: " + name)
				o.refresh()
				if o.cursor >= len(o.items) && o.cursor > 0 {
					o.cursor--
				}
			}
		}
		return true, nil
	case "s":
		// Switch to save mode
		o.mode = pmSave
		o.nameIn.Focus()
		o.nameIn.SetValue("")
		return true, nil
	}
	return true, nil
}

func (o *profileOverlay) handleSave(msg tea.KeyMsg, currentCfg func() (bench.Config, error)) (bool, *bench.Config) {
	switch msg.String() {
	case "esc":
		o.mode = pmBrowse
		o.nameIn.Blur()
		return true, nil
	case "enter":
		name := strings.TrimSpace(o.nameIn.Value())
		if name == "" {
			o.setMsg("⚠ name cannot be empty")
			return true, nil
		}
		cfg, err := currentCfg()
		if err != nil {
			o.setMsg("⚠ " + err.Error())
			return true, nil
		}
		p := profiles.Profile{
			Name:   name,
			Config: cfg,
		}
		if err := o.store.Save(p); err != nil {
			o.setMsg("✗ " + err.Error())
		} else {
			o.setMsg("✓ saved: " + name)
			o.refresh()
			o.mode = pmBrowse
			o.nameIn.Blur()
		}
		return true, nil
	}
	// Forward to text input
	var cmd tea.Cmd
	o.nameIn, cmd = o.nameIn.Update(msg)
	_ = cmd
	return true, nil
}

func (o *profileOverlay) setMsg(s string) {
	o.msg = s
	o.msgTime = time.Now()
}

func (o *profileOverlay) statusLine() string {
	if o.msg == "" {
		return ""
	}
	if time.Since(o.msgTime) > 4*time.Second {
		o.msg = ""
		return ""
	}
	if strings.HasPrefix(o.msg, "✓") {
		return styles.OK.Render(o.msg)
	}
	return styles.Err.Render(o.msg)
}

func (o *profileOverlay) view(w, h int) string {
	if !o.active {
		return ""
	}

	var body string
	switch o.mode {
	case pmBrowse:
		body = o.viewBrowse()
	case pmSave:
		body = o.viewSave()
	}

	boxW := 50
	if boxW > w-4 {
		boxW = w - 4
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(styles.AccentAlt).
		Padding(1, 2).
		Width(boxW).
		Render(body)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}

func (o *profileOverlay) viewBrowse() string {
	title := styles.CardTitle.Render("Profiles")

	if len(o.items) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			"",
			styles.Muted.Render("  No saved profiles."),
			"",
			styles.Muted.Render("s: save current config • esc: close"),
		)
	}

	lines := make([]string, 0, len(o.items))
	for i, p := range o.items {
		label := fmt.Sprintf("%s %s", p.Config.Method, p.Config.URL)
		if len(label) > 40 {
			label = label[:37] + "..."
		}
		name := p.Name
		row := fmt.Sprintf("%-16s %s", name, styles.Muted.Render(label))
		if i == o.cursor {
			lines = append(lines, styles.Accent.Render("▸ "+row))
		} else {
			lines = append(lines, "  "+row)
		}
	}

	help := styles.Muted.Render("↑/↓ select • enter load • d delete • s save • esc close")
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(lines, "\n"),
		"",
		help,
	)
}

func (o *profileOverlay) viewSave() string {
	title := styles.CardTitle.Render("Save Profile")
	label := styles.Label.Render("Name: ")
	input := o.nameIn.View()

	help := styles.Muted.Render("enter: save • esc: cancel")
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, label, input),
		"",
		help,
	)
}
