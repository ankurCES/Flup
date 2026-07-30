package tui

import (
	"fmt"
	"time"

	"github.com/ankurCES/Flup/internal/bench"
)

// durShort formats a time.Duration with adaptive units:
//   ns/µs under 1ms, ms up to 1s, s after. Keeps the live view scannable.
func durShort(d time.Duration, useSec bool) string {
	if useSec {
		return fmt.Sprintf("%.3fs", d.Seconds())
	}
	d = d.Truncate(time.Microsecond)
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Nanoseconds()/1000)
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Nanoseconds()/1_000_000)
	default:
		return fmt.Sprintf("%.3fs", d.Seconds())
	}
}

// durShortP picks the percentile at idx from a Snapshot for inline display.
func durShortP(s *bench.Snapshot, idx int, useSec bool) string {
	if s == nil || idx >= len(s.Percentiles) {
		return "—"
	}
	return durShort(s.Percentiles[idx].Latency, useSec)
}

func humanInt(n int64) string {
	switch {
	case n < 1_000:
		return fmt.Sprintf("%d", n)
	case n < 1_000_000:
		return fmt.Sprintf("%.2fk", float64(n)/1_000)
	case n < 1_000_000_000:
		return fmt.Sprintf("%.2fM", float64(n)/1_000_000)
	default:
		return fmt.Sprintf("%.2fB", float64(n)/1_000_000_000)
	}
}

func humanFloat(f float64) string {
	switch {
	case f < 1_000:
		return fmt.Sprintf("%.1f", f)
	case f < 1_000_000:
		return fmt.Sprintf("%.2fk", f/1_000)
	default:
		return fmt.Sprintf("%.2fM", f/1_000_000)
	}
}
