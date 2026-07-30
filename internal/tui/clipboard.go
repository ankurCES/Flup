package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"

	"github.com/ankurCES/Flup/internal/bench"
	"github.com/ankurCES/Flup/internal/export"
)

// clipResult holds the result of a clipboard operation for status display.
type clipResult struct {
	msg     string
	ok      bool
	created time.Time
}

func (c *clipResult) statusLine() string {
	if c == nil || c.msg == "" {
		return ""
	}
	if time.Since(c.created) > 4*time.Second {
		c.msg = ""
		return ""
	}
	return c.msg
}

// copySnapshotToClipboard formats the current snapshot as a compact text
// summary and copies it to the system clipboard.
func copySnapshotToClipboard(cfg bench.Config, snap *bench.Snapshot) clipResult {
	if snap == nil {
		return clipResult{msg: "⚠ no data to copy", created: time.Now()}
	}

	report := export.NewReport(cfg, snap)
	text := formatClipboardText(report)

	if err := clipboard.WriteAll(text); err != nil {
		return clipResult{
			msg:     fmt.Sprintf("✗ clipboard: %s", err),
			created: time.Now(),
		}
	}
	return clipResult{
		msg:     "✓ copied to clipboard",
		ok:      true,
		created: time.Now(),
	}
}

// formatClipboardText produces a clean, pasteable text summary.
func formatClipboardText(r export.Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Flup Benchmark — %s %s\n", r.Method, r.URL)
	fmt.Fprintf(&b, "Concurrency: %d  Duration: %s  Elapsed: %s\n\n", r.Concurrency, r.Duration, r.Elapsed)

	fmt.Fprintf(&b, "Requests:  %d\n", r.Count)
	fmt.Fprintf(&b, "RPS:       %.2f\n", r.RPS)
	fmt.Fprintf(&b, "Read:      %.4f MB/s\n", r.ReadMBps)
	fmt.Fprintf(&b, "Write:     %.4f MB/s\n\n", r.WriteMBps)

	b.WriteString("Latency:\n")
	fmt.Fprintf(&b, "  Min:    %s\n", r.Latency.Min)
	fmt.Fprintf(&b, "  Mean:   %s\n", r.Latency.Mean)
	fmt.Fprintf(&b, "  StdDev: %s\n", r.Latency.StdDev)
	fmt.Fprintf(&b, "  Max:    %s\n\n", r.Latency.Max)

	b.WriteString("RPS Stats:\n")
	fmt.Fprintf(&b, "  Min: %.2f  Mean: %.2f  StdDev: %.2f  Max: %.2f\n\n",
		r.RPSStats.Min, r.RPSStats.Mean, r.RPSStats.StdDev, r.RPSStats.Max)

	if len(r.Percentiles) > 0 {
		b.WriteString("Percentiles:\n")
		for _, p := range r.Percentiles {
			fmt.Fprintf(&b, "  %-8s %s\n", p.Label, p.Latency)
		}
		b.WriteString("\n")
	}

	if len(r.StatusCodes) > 0 {
		b.WriteString("Status Codes:\n")
		for code, count := range r.StatusCodes {
			fmt.Fprintf(&b, "  %s: %d\n", code, count)
		}
	}

	return b.String()
}
