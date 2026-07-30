package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ankurCES/Flup/internal/bench"
	"github.com/ankurCES/Flup/internal/styles"
)

// runView is the configuration + start/stop screen. It is the first thing
// the user sees; everything else is a read-only view.
type runView struct {
	inputs   []textinput.Model
	focus    int
	sw       stopwatch.Model
	prog     progress.Model
	running  bool
	finished bool
	err      string
	started  time.Time
	dur      time.Duration
	reqs     int64
}

// fields are the editable parameters — order = tab order
var runFields = []struct {
	label string
	ph    string
	help  string
}{
	{"URL", "http://127.0.0.1:8080/", "Target URL"},
	{"Method", "GET", "HTTP method"},
	{"Concurrency", "20", "Connections"},
	{"Duration", "10s", "Run duration (0 = until stopped)"},
	{"Requests", "0", "Total requests (0 = unbounded)"},
	{"Rate", "infinity", "Requests per time (e.g. 50/1s)"},
	{"Body", "", "@file or raw body"},
	{"Content-Type", "", "application/json"},
	{"Headers", "", "K: V, comma separated"},
	{"Timeout", "5s", "Per-request timeout"},
	{"Insecure", "false", "Skip TLS verify (true/false)"},
	{"HTTP/2", "false", "Use HTTP/2 (true/false)"},
	{"Expect Status", "", "Expected HTTP status (e.g. 200)"},
	{"Expect Body", "", "Response must contain this string"},
	{"Timeout", "30s", "Per-request timeout (e.g. 10s, 500ms)"},
	{"Keep-Alive", "true", "Reuse connections (true/false)"},
	{"Max Conns", "", "Max connections per host (0 = auto)"},
}

func newRunView() *runView {
	rv := &runView{}
	for i, f := range runFields {
		ti := textinput.New()
		ti.Placeholder = f.ph
		ti.CharLimit = 256
		ti.PromptStyle = styles.Accent
		ti.TextStyle = styles.Value
		ti.PlaceholderStyle = styles.Muted
		if i == 0 {
			ti.Focus()
			ti.Prompt = "> "
		} else {
			ti.Prompt = "  "
		}
		// sensible defaults
		switch f.label {
		case "URL":
			ti.SetValue("")
		case "Method":
			ti.SetValue("GET")
		case "Concurrency":
			ti.SetValue("20")
		case "Duration":
			ti.SetValue("10s")
		case "Requests":
			ti.SetValue("0")
		case "Rate":
			ti.SetValue("infinity")
		case "Timeout":
			ti.SetValue("5s")
		case "Insecure":
			ti.SetValue("false")
		}
		rv.inputs = append(rv.inputs, ti)
	}
	rv.sw = stopwatch.NewWithInterval(time.Millisecond * 100)
	rv.prog = progress.New(
		progress.WithGradient("#7D56F4", "#FF6AC6"),
		progress.WithoutPercentage(),
	)
	return rv
}

func (v *runView) Title() string { return "Run" }

func (v *runView) Init() tea.Cmd { return textinput.Blink }

func (v *runView) Update(msg tea.Msg, env *Env) (tea.Cmd, bool) {
	if v.running {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			env.Width, env.Height = msg.Width, msg.Height
		case stopwatch.TickMsg:
			var cmd tea.Cmd
			v.sw, cmd = v.sw.Update(msg)
			return cmd, true
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+x", "esc":
				v.stop(env)
				return nil, true
			}
		}
		// keep text inputs ticking even while running so they aren't focused
		return nil, false
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		env.Width, env.Height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			v.focus = (v.focus + 1) % len(v.inputs)
			v.refreshFocus()
			return nil, true
		case "shift+tab", "up":
			v.focus = (v.focus - 1 + len(v.inputs)) % len(v.inputs)
			v.refreshFocus()
			return nil, true
		case "ctrl+s":
			v.start(env)
			return nil, true
		}
	}
	cmds := make([]tea.Cmd, 0, len(v.inputs))
	for i := range v.inputs {
		var cmd tea.Cmd
		v.inputs[i], cmd = v.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...), false
}

func (v *runView) refreshFocus() {
	for i := range v.inputs {
		if i == v.focus {
			v.inputs[i].Focus()
			v.inputs[i].Prompt = "> "
		} else {
			v.inputs[i].Blur()
			v.inputs[i].Prompt = "  "
		}
	}
}

func (v *runView) readConfig() (bench.Config, error) {
	get := func(i int) string { return strings.TrimSpace(v.inputs[i].Value()) }
	c := bench.Config{
		URL:         get(0),
		Method:      strings.ToUpper(orDefault(get(1), "GET")),
		Body:        get(6),
		ContentType: get(7),
	}
	if c.ContentType != "" {
		c.Headers = append(c.Headers, "Content-Type: "+c.ContentType)
	}
	if h := get(8); h != "" {
		for _, p := range strings.Split(h, ",") {
			if s := strings.TrimSpace(p); s != "" {
				c.Headers = append(c.Headers, s)
			}
		}
	}
	conc, err := strconv.Atoi(orDefault(get(2), "1"))
	if err != nil || conc < 1 {
		return c, fmt.Errorf("invalid concurrency")
	}
	c.Concurrency = conc
	if d := get(3); d != "" {
		du, err := time.ParseDuration(d)
		if err != nil {
			return c, fmt.Errorf("bad duration: %w", err)
		}
		c.Duration = du
	}
	if n := get(4); n != "" {
		ni, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return c, fmt.Errorf("bad requests: %w", err)
		}
		c.Requests = ni
	}
	if r := get(5); r != "" && r != "infinity" {
		lim, err := benchPace(r)
		if err != nil {
			return c, err
		}
		c.RatePtr = lim
	}
	if t := get(9); t != "" {
		du, err := time.ParseDuration(t)
		if err != nil {
			return c, fmt.Errorf("bad timeout: %w", err)
		}
		c.Timeout = du
	}
	if k := strings.EqualFold(get(10), "true"); k {
		c.Insecure = true
	}
	if strings.EqualFold(get(11), "true") {
		c.HTTP2 = true
	}
	// Response validation
	if es := get(12); es != "" {
		sc, err := strconv.Atoi(es)
		if err != nil {
			return c, fmt.Errorf("bad expect status: %w", err)
		}
		c.ExpectStatus = sc
	}
	c.ExpectBody = get(13)
	// Connection tuning
	if ts := get(14); ts != "" {
		td, err := time.ParseDuration(ts)
		if err != nil {
			return c, fmt.Errorf("bad timeout: %w", err)
		}
		c.Timeout = td
	}
	c.KeepAlive = !strings.EqualFold(get(15), "false") // default true
	if mc := get(16); mc != "" {
		n, err := strconv.Atoi(mc)
		if err != nil {
			return c, fmt.Errorf("bad max conns: %w", err)
		}
		c.MaxConnsPerHost = n
	}
	return c, c.Validate()
}

// benchPace keeps the parser in one place so we don't import bench
// internals from the view layer.
func benchPace(s string) (*rateLimit, error) {
	// delegate to internal/bench via small shim
	lim, err := parsePace(s)
	if err != nil {
		return nil, err
	}
	return lim, nil
}

// loadConfig populates the input fields from a saved bench.Config.
func (v *runView) loadConfig(cfg bench.Config) {
	set := func(i int, val string) {
		if i < len(v.inputs) {
			v.inputs[i].SetValue(val)
		}
	}
	set(0, cfg.URL)
	set(1, cfg.Method)
	set(2, fmt.Sprint(cfg.Concurrency))
	if cfg.Duration > 0 {
		set(3, cfg.Duration.String())
	} else {
		set(3, "0s")
	}
	set(4, fmt.Sprint(cfg.Requests))
	if cfg.RatePtr != nil {
		set(5, fmt.Sprintf("%.0f/1s", float64(*cfg.RatePtr)))
	} else {
		set(5, "infinity")
	}
	set(6, cfg.Body)
	set(7, cfg.ContentType)
	// Headers (skip Content-Type since it's field 7)
	hdrs := make([]string, 0)
	for _, h := range cfg.Headers {
		if !strings.HasPrefix(strings.ToLower(h), "content-type:") {
			hdrs = append(hdrs, h)
		}
	}
	set(8, strings.Join(hdrs, ", "))
	if cfg.Timeout > 0 {
		set(9, cfg.Timeout.String())
	} else {
		set(9, "5s")
	}
	if cfg.Insecure {
		set(10, "true")
	} else {
		set(10, "false")
	}
	if cfg.HTTP2 {
		set(11, "true")
	} else {
		set(11, "false")
	}
	if cfg.ExpectStatus != 0 {
		set(12, fmt.Sprint(cfg.ExpectStatus))
	} else {
		set(12, "")
	}
	set(13, cfg.ExpectBody)
	if cfg.Timeout > 0 {
		set(14, cfg.Timeout.String())
	} else {
		set(14, "30s")
	}
	if cfg.KeepAlive {
		set(15, "true")
	} else {
		set(15, "false")
	}
	if cfg.MaxConnsPerHost > 0 {
		set(16, fmt.Sprint(cfg.MaxConnsPerHost))
	} else {
		set(16, "")
	}
}

func (v *runView) start(env *Env) {
	cfg, err := v.readConfig()
	if err != nil {
		v.err = err.Error()
		return
	}
	v.err = ""
	if err := env.Runner.Start(cfg); err != nil {
		v.err = err.Error()
		return
	}
	v.running = true
	v.finished = false
	v.started = time.Now()
	v.reqs = cfg.Requests
	v.dur = cfg.Duration
	v.sw.Reset()
	v.sw.Start()
}

func (v *runView) stop(env *Env) {
	if !v.running {
		return
	}
	<-env.Runner.Stop()
	v.running = false
	v.finished = true
	v.sw.Stop()

	// save to history
	if env.Store != nil {
		snap := env.Runner.Snapshot()
		if snap != nil {
			_ = env.Store.Save(HistoryEntry{
				Config:   env.Runner.Config(),
				Snapshot: *snap,
			})
		}
	}
}

func (v *runView) View(w, h int) string {
	if v == nil {
		return ""
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		styles.CardTitle.Render("Benchmark configuration"),
		"  ",
		styles.TagAlt.Render("tab to move • ctrl-s to start • ctrl-x to stop"),
	)

	var body strings.Builder
	body.WriteString(header + "\n\n")

	// 2-column input grid
	colW := w/2 - 2
	if colW < 30 {
		colW = 30
	}
	left := make([]string, 0, len(v.inputs)/2+1)
	right := make([]string, 0, len(v.inputs)/2+1)
	for i, f := range runFields {
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			styles.Label.Render(padR(f.label+":", 14)),
			v.inputs[i].View(),
		)
		if i%2 == 0 {
			left = append(left, row)
		} else {
			right = append(right, row)
		}
	}
	for i := 0; i < len(left) || i < len(right); i++ {
		var l, r string
		if i < len(left) {
			l = lipgloss.NewStyle().Width(colW).Render(left[i])
		}
		if i < len(right) {
			r = right[i]
		}
		body.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, l, r) + "\n")
	}

	if v.err != "" {
		body.WriteString("\n" + styles.Err.Render("✗ "+v.err) + "\n")
	}

	// status / progress
	body.WriteString("\n")
	if v.running {
		elapsed := time.Since(v.started)
		pct := 0.0
		if v.dur > 0 {
			pct = float64(elapsed) / float64(v.dur)
			if pct > 1 {
				pct = 1
			}
		} else if v.reqs > 0 && lastSnap() != nil {
			pct = float64(lastSnap().Count) / float64(v.reqs)
		}
		body.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			styles.Accent.Render("⏱  "),
			styles.Value.Render(v.sw.View()),
			"    ",
			styles.Muted.Render(formatProgressLine()),
			"\n",
		))
		body.WriteString(v.prog.ViewAs(pct) + "\n")
	} else if v.finished {
		body.WriteString(styles.OK.Render("✓ benchmark finished • results saved to history") + "\n")
	} else {
		body.WriteString(styles.Muted.Render("Press ctrl-s to begin a new benchmark.") + "\n")
	}

	return styles.Card.Width(w - 2).Render(body.String())
}

func formatProgressLine() string {
	s := lastSnap()
	if s == nil {
		return "starting…"
	}
	p50 := durShortP(s, 0, lastUseSec())
	return fmt.Sprintf("count %s   rps %s   p50 %s   err %s",
		humanInt(s.Count), humanFloat(s.RPS), p50, humanInt(sumErrors(s.Errors)))
}

func sumErrors(m map[string]int64) int64 {
	var n int64
	for _, v := range m {
		n += v
	}
	return n
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func padR(s string, n int) string {
	if len(s) >= n {
		return s + " "
	}
	return s + strings.Repeat(" ", n-len(s))
}
