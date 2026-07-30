// Package export formats benchmark results as JSON, CSV, or Markdown
// for sharing, CI integration, or archival.
package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ankurCES/Flup/internal/bench"
)

// Format identifies an export format.
type Format int

const (
	JSON Format = iota
	CSV
	Markdown
)

// String returns the file extension for the format.
func (f Format) String() string {
	switch f {
	case JSON:
		return "json"
	case CSV:
		return "csv"
	case Markdown:
		return "md"
	default:
		return "txt"
	}
}

// Report bundles the config + snapshot for export.
type Report struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Concurrency int               `json:"concurrency"`
	Duration    string            `json:"duration"`
	Timestamp   time.Time         `json:"timestamp"`
	Elapsed     string            `json:"elapsed"`
	Count       int64             `json:"total_requests"`
	RPS         float64           `json:"rps"`
	ReadMBps    float64           `json:"read_mbps"`
	WriteMBps   float64           `json:"write_mbps"`
	Latency     LatencyReport     `json:"latency"`
	RPSStats    RPSStatsReport    `json:"rps_stats"`
	StatusCodes map[string]int64  `json:"status_codes"`
	Errors      map[string]int64  `json:"errors,omitempty"`
	Percentiles []PercentileRow   `json:"percentiles"`
	Histograms  []HistogramRow    `json:"histograms,omitempty"`
}

// LatencyReport is the latency summary sub-object.
type LatencyReport struct {
	Min    string  `json:"min"`
	Mean   string  `json:"mean"`
	StdDev string  `json:"stddev"`
	Max    string  `json:"max"`
	MinNs  int64   `json:"min_ns"`
	MeanNs int64   `json:"mean_ns"`
	MaxNs  int64   `json:"max_ns"`
}

// RPSStatsReport is the RPS statistics sub-object.
type RPSStatsReport struct {
	Min    float64 `json:"min"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"stddev"`
	Max    float64 `json:"max"`
}

// PercentileRow is one row in the percentile table.
type PercentileRow struct {
	Percentile float64 `json:"percentile"`
	Label      string  `json:"label"`
	Latency    string  `json:"latency"`
	LatencyNs  int64   `json:"latency_ns"`
}

// HistogramRow is one histogram bucket.
type HistogramRow struct {
	BucketMean   string `json:"bucket_mean"`
	BucketMeanNs int64  `json:"bucket_mean_ns"`
	Count        int    `json:"count"`
}

// NewReport builds an export Report from a Config + Snapshot.
func NewReport(cfg bench.Config, snap *bench.Snapshot) Report {
	r := Report{
		URL:         cfg.URL,
		Method:      cfg.Method,
		Concurrency: cfg.Concurrency,
		Duration:    cfg.Duration.String(),
		Timestamp:   time.Now(),
		Elapsed:     snap.Elapsed.Truncate(time.Millisecond).String(),
		Count:       snap.Count,
		RPS:         snap.RPS,
		ReadMBps:    snap.ReadMBps,
		WriteMBps:   snap.WriteMBps,
		StatusCodes: snap.Codes,
		Errors:      snap.Errors,
		Latency: LatencyReport{
			Min:    durStr(snap.Latency.Min),
			Mean:   durStr(snap.Latency.Mean),
			StdDev: durStr(snap.Latency.StdDev),
			Max:    durStr(snap.Latency.Max),
			MinNs:  int64(snap.Latency.Min),
			MeanNs: int64(snap.Latency.Mean),
			MaxNs:  int64(snap.Latency.Max),
		},
		RPSStats: RPSStatsReport{
			Min:    snap.RPSStats.Min,
			Mean:   snap.RPSStats.Mean,
			StdDev: snap.RPSStats.StdDev,
			Max:    snap.RPSStats.Max,
		},
	}
	for _, p := range snap.Percentiles {
		r.Percentiles = append(r.Percentiles, PercentileRow{
			Percentile: p.P * 100,
			Label:      pLabel(p.P),
			Latency:    durStr(p.Latency),
			LatencyNs:  int64(p.Latency),
		})
	}
	for _, h := range snap.Histograms {
		r.Histograms = append(r.Histograms, HistogramRow{
			BucketMean:   durStr(h.Mean),
			BucketMeanNs: int64(h.Mean),
			Count:        h.Count,
		})
	}
	return r
}

// ToJSON marshals the report as indented JSON.
func (r Report) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// ToCSV writes the report as a CSV with header row + one data row for
// the summary, then percentile rows, then histogram rows.
func (r Report) ToCSV() ([]byte, error) {
	var b strings.Builder
	w := csv.NewWriter(&b)

	// Summary header
	_ = w.Write([]string{
		"url", "method", "concurrency", "duration", "elapsed",
		"total_requests", "rps",
		"latency_min", "latency_mean", "latency_stddev", "latency_max",
		"rps_min", "rps_mean", "rps_stddev", "rps_max",
		"read_mbps", "write_mbps",
	})
	_ = w.Write([]string{
		r.URL, r.Method, fmt.Sprint(r.Concurrency), r.Duration, r.Elapsed,
		fmt.Sprint(r.Count), fmt.Sprintf("%.2f", r.RPS),
		r.Latency.Min, r.Latency.Mean, r.Latency.StdDev, r.Latency.Max,
		fmt.Sprintf("%.2f", r.RPSStats.Min), fmt.Sprintf("%.2f", r.RPSStats.Mean),
		fmt.Sprintf("%.2f", r.RPSStats.StdDev), fmt.Sprintf("%.2f", r.RPSStats.Max),
		fmt.Sprintf("%.4f", r.ReadMBps), fmt.Sprintf("%.4f", r.WriteMBps),
	})

	// Status codes
	_ = w.Write([]string{})
	_ = w.Write([]string{"status_code", "count"})
	for k, v := range r.StatusCodes {
		_ = w.Write([]string{k, fmt.Sprint(v)})
	}

	// Percentiles
	_ = w.Write([]string{})
	_ = w.Write([]string{"percentile", "latency", "latency_ns"})
	for _, p := range r.Percentiles {
		_ = w.Write([]string{p.Label, p.Latency, fmt.Sprint(p.LatencyNs)})
	}

	w.Flush()
	return []byte(b.String()), w.Error()
}

// ToMarkdown renders the report as a human-readable Markdown document.
func (r Report) ToMarkdown() ([]byte, error) {
	var b strings.Builder

	b.WriteString("# Flup Benchmark Report\n\n")
	fmt.Fprintf(&b, "**URL:** `%s`  \n", r.URL)
	fmt.Fprintf(&b, "**Method:** `%s`  \n", r.Method)
	fmt.Fprintf(&b, "**Concurrency:** %d  \n", r.Concurrency)
	fmt.Fprintf(&b, "**Duration:** %s  \n", r.Duration)
	fmt.Fprintf(&b, "**Timestamp:** %s  \n\n", r.Timestamp.Format(time.RFC3339))

	// Summary table
	b.WriteString("## Summary\n\n")
	b.WriteString("| Metric | Value |\n|--------|-------|\n")
	fmt.Fprintf(&b, "| Elapsed | %s |\n", r.Elapsed)
	fmt.Fprintf(&b, "| Total Requests | %d |\n", r.Count)
	fmt.Fprintf(&b, "| RPS | %.2f |\n", r.RPS)
	fmt.Fprintf(&b, "| Read Throughput | %.4f MB/s |\n", r.ReadMBps)
	fmt.Fprintf(&b, "| Write Throughput | %.4f MB/s |\n\n", r.WriteMBps)

	// Latency
	b.WriteString("## Latency\n\n")
	b.WriteString("| Stat | Value |\n|------|-------|\n")
	fmt.Fprintf(&b, "| Min | %s |\n", r.Latency.Min)
	fmt.Fprintf(&b, "| Mean | %s |\n", r.Latency.Mean)
	fmt.Fprintf(&b, "| StdDev | %s |\n", r.Latency.StdDev)
	fmt.Fprintf(&b, "| Max | %s |\n\n", r.Latency.Max)

	// RPS Stats
	b.WriteString("## RPS Statistics\n\n")
	b.WriteString("| Stat | Value |\n|------|-------|\n")
	fmt.Fprintf(&b, "| Min | %.2f |\n", r.RPSStats.Min)
	fmt.Fprintf(&b, "| Mean | %.2f |\n", r.RPSStats.Mean)
	fmt.Fprintf(&b, "| StdDev | %.2f |\n", r.RPSStats.StdDev)
	fmt.Fprintf(&b, "| Max | %.2f |\n\n", r.RPSStats.Max)

	// Percentiles
	b.WriteString("## Latency Percentiles\n\n")
	b.WriteString("| Percentile | Latency |\n|------------|----------|\n")
	for _, p := range r.Percentiles {
		fmt.Fprintf(&b, "| %s | %s |\n", p.Label, p.Latency)
	}
	b.WriteString("\n")

	// Status codes
	b.WriteString("## Status Codes\n\n")
	b.WriteString("| Code | Count |\n|------|-------|\n")
	for k, v := range r.StatusCodes {
		fmt.Fprintf(&b, "| %s | %d |\n", k, v)
	}
	b.WriteString("\n")

	// Errors
	if len(r.Errors) > 0 {
		b.WriteString("## Errors\n\n")
		b.WriteString("| Error | Count |\n|-------|-------|\n")
		for k, v := range r.Errors {
			fmt.Fprintf(&b, "| %s | %d |\n", k, v)
		}
		b.WriteString("\n")
	}

	b.WriteString("---\n*Generated by [Flup](https://github.com/ankurCES/Flup)*\n")
	return []byte(b.String()), nil
}

// WriteFile exports the report to a file. dir is the output directory;
// if empty, uses the current working directory. Returns the path written.
func WriteFile(r Report, format Format, dir string) (string, error) {
	var data []byte
	var err error
	switch format {
	case JSON:
		data, err = r.ToJSON()
	case CSV:
		data, err = r.ToCSV()
	case Markdown:
		data, err = r.ToMarkdown()
	default:
		return "", fmt.Errorf("unknown format %d", format)
	}
	if err != nil {
		return "", err
	}

	if dir == "" {
		dir, _ = os.Getwd()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	ts := time.Now().Format("20060102-150405")
	name := fmt.Sprintf("flup-%s.%s", ts, format.String())
	path := filepath.Join(dir, name)
	return path, os.WriteFile(path, data, 0o644)
}

// helpers
func durStr(d time.Duration) string {
	d = d.Truncate(time.Microsecond)
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Nanoseconds()/1000)
	case d < time.Second:
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	default:
		return fmt.Sprintf("%.3fs", d.Seconds())
	}
}

func pLabel(p float64) string {
	switch p {
	case 0.5:
		return "P50"
	case 0.75:
		return "P75"
	case 0.9:
		return "P90"
	case 0.95:
		return "P95"
	case 0.99:
		return "P99"
	case 0.999:
		return "P99.9"
	case 0.9999:
		return "P99.99"
	default:
		return fmt.Sprintf("P%.2f", p*100)
	}
}
