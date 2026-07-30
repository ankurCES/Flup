package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// rateLimit is just an alias so view files don't import rate directly.
type rateLimit = rate.Limit

// parsePace turns "50/1s" or "infinity" into a *rate.Limit, mirroring
// plow's --rate flag exactly so users copy-pasting examples "just work".
func parsePace(s string) (*rate.Limit, error) {
	if s == "" || s == "infinity" || s == "inf" {
		return nil, nil
	}
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 1 {
		parts = append(parts, "1s")
	}
	freq, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("bad rate %q", s)
	}
	if freq == 0 {
		return nil, nil
	}
	switch parts[1] {
	case "ns", "us", "µs", "ms", "s", "m", "h":
		parts[1] = "1" + parts[1]
	}
	d, err := time.ParseDuration(parts[1])
	if err != nil {
		return nil, fmt.Errorf("bad rate unit %q", parts[1])
	}
	l := rate.Limit(float64(freq) / d.Seconds())
	return &l, nil
}
