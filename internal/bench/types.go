// Package bench contains the HTTP load-generation engine used by Flup.
// It is a Go-idiomatic rewrite of plow's requester/report with a few
// changes: explicit context cancel, structured errors, and a thread-safe
// snapshot API that the TUI polls.
package bench

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// Config is the validated, TUI-ready form of every plow flag. All durations
// use Go syntax. RatePtr==nil means "no rate cap".
type Config struct {
	URL           string
	Method        string
	Concurrency   int
	Duration      time.Duration // 0 = unbounded
	Requests      int64         // 0 = unbounded
	RampUp        int           // conn/sec ramp, -1 = instant
	Headers       []string      // "K: V"
	Host          string
	ContentType   string
	Body          string // raw or "@filename"
	StreamBody    bool
	Timeout       time.Duration
	DialTimeout   time.Duration
	ReqTimeout    time.Duration
	RespTimeout   time.Duration
	Insecure      bool
	CertPath      string
	KeyPath       string
	UnixSocket    string
	RatePtr       *rate.Limit // populated by Pace if Rate is set; nil = no cap
}

// Validate returns the first user-visible error in the configuration.
// Called from the Run screen before starting a benchmark.
func (c *Config) Validate() error {
	if c.URL == "" {
		return errors.New("URL is required")
	}
	if c.Concurrency < 1 {
		return errors.New("concurrency must be >= 1")
	}
	if c.Duration == 0 && c.Requests == 0 {
		// allowed — Ctrl-C stops it; plow treats this as "run forever"
	}
	if c.Duration < 0 {
		return errors.New("duration must be >= 0")
	}
	if c.Requests < 0 {
		return errors.New("requests must be >= 0")
	}
	switch strings.ToUpper(c.Method) {
	case "GET", "POST", "PUT", "DELETE", "HEAD", "PATCH", "OPTIONS":
	default:
		return fmt.Errorf("unsupported method %q", c.Method)
	}
	return nil
}

// Snapshot is the point-in-time view of a benchmark used by every chart
// and table. Mirrors plow's SnapshotReport so the TUI gets familiar
// numbers without re-doing the math.
type Snapshot struct {
	Elapsed         time.Duration
	Count           int64
	Codes           map[string]int64 // "2xx","4xx",...
	ExactCodes      map[int]int64    // 200, 201, 404, ...
	Errors          map[string]int64
	RPS             float64
	ReadMBps        float64
	WriteMBps       float64
	ConcurrencyUsed int

	Latency struct {
		Min, Mean, StdDev, Max time.Duration
	}
	RPSStats struct {
		Min, Mean, StdDev, Max float64
	}
	Percentiles []Percentile
	Histograms  []HistogramBin
}

type Percentile struct {
	P       float64
	Latency time.Duration
}

type HistogramBin struct {
	Mean  time.Duration
	Count int
}

// paceToLimit converts the human "50/1s" rate string used by plow into a
// rate.Limit. The TUI's rate field shares one impl. Exported because
// cmd/flup needs it during config parsing.
func paceToLimit(s string) (*rate.Limit, error) {
	if s == "" || s == "infinity" || s == "inf" {
		return nil, nil
	}
	ps := strings.SplitN(s, "/", 2)
	if len(ps) == 1 {
		ps = append(ps, "1s")
	}
	freq, err := strconv.Atoi(ps[0])
	if err != nil {
		return nil, fmt.Errorf("bad rate %q", s)
	}
	if freq == 0 {
		return nil, nil
	}
	switch ps[1] {
	case "ns", "us", "µs", "ms", "s", "m", "h":
		ps[1] = "1" + ps[1]
	}
	d, err := time.ParseDuration(ps[1])
	if err != nil {
		return nil, fmt.Errorf("bad rate unit %q", ps[1])
	}
	l := rate.Limit(float64(freq) / d.Seconds())
	return &l, nil
}

// Keep strconv imported when the helper above is the only caller.
var _ = strconv.Itoa
