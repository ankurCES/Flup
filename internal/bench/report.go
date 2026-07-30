package bench

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/beorn7/perks/histogram"
	"github.com/beorn7/perks/quantile"
)

// quantiles mirrors plow's reported set exactly so the TUI rows match
// muscle memory for anyone coming from plow.
var quantiles = []float64{0.50, 0.75, 0.90, 0.95, 0.99, 0.999, 0.9999}

// quantilesTarget is the precision map used by the streaming quantile.
// It mirrors client_golang / plow so latencies within ±1% of the percentile
// are accurate even on slow benchmarks.
var quantilesTarget = map[float64]float64{
	0.50: 0.01, 0.75: 0.01, 0.90: 0.001, 0.95: 0.001,
	0.99: 0.001, 0.999: 0.0001, 0.9999: 0.00001,
}

// statusSectionMap turns a raw HTTP status (e.g. 404) into its section
// label ("4xx"). Anything outside 1xx–5xx is bucketed as "other".
var statusSectionMap = map[int]string{1: "1xx", 2: "2xx", 3: "3xx", 4: "4xx", 5: "5xx"}

// stats is the running Welford accumulator used for latency + RPS stats.
type stats struct {
	count int64
	sum   float64
	sumSq float64
	min   float64
	max   float64
}

func (s *stats) update(v float64) {
	s.count++
	s.sum += v
	s.sumSq += v * v
	if v < s.min || s.count == 1 {
		s.min = v
	}
	if v > s.max || s.count == 1 {
		s.max = v
	}
}
func (s *stats) mean() float64 {
	if s.count == 0 {
		return 0
	}
	return s.sum / float64(s.count)
}
func (s *stats) stddev() float64 {
	num := float64(s.count)*s.sumSq - s.sum*s.sum
	div := float64(s.count * (s.count - 1))
	if div == 0 {
		return 0
	}
	return sqrt(num / div)
}
func (s *stats) reset() {
	s.count, s.sum, s.sumSq, s.min, s.max = 0, 0, 0, 0, 0
}

// Record is what the requester hands to the StreamReport after each
// completed HTTP exchange. Kept small — it travels through a channel at
// peak RPS.
type Record struct {
	Cost             time.Duration
	Code             int
	Error            string
	ReadBytes        int64
	WriteBytes       int64
	ConcurrencyCount int
}

// StreamReport is the live aggregator. It ingests Records on a channel,
// computes rolling stats + quantiles + histogram, and exposes a thread-
// safe Snapshot() for the TUI to poll.
type StreamReport struct {
	startNanos int64 // atomic unix nanos of test start
	doneCh     chan struct{}

	mu               sync.Mutex
	latencyStats     *stats
	rpsStats         *stats
	latencyQuantile  *quantile.Stream
	latencyHistogram *histogram.Histogram
	codes            map[int]int64
	errors           map[string]int64
	concurrency      int

	// per-second rolling window
	latencyWithinSec *stats
	rpsWithinSec     float64
	noDataWithinSec  bool

	readBytes  int64
	writeBytes int64
}

// NewStreamReport constructs an empty, ready-to-Collect aggregator.
// startNanos is set on Start() so the report is reusable across runs.
func NewStreamReport() *StreamReport {
	return &StreamReport{
		latencyQuantile:  quantile.NewTargeted(quantilesTarget),
		latencyHistogram: histogram.New(8),
		codes:            make(map[int]int64, 4),
		errors:           make(map[string]int64, 4),
		doneCh:           make(chan struct{}, 1),
		latencyStats:     &stats{},
		rpsStats:         &stats{},
		latencyWithinSec: &stats{},
	}
}

// Start records the start time. Call before pumping records.
func (s *StreamReport) Start() {
	atomic.StoreInt64(&s.startNanos, time.Now().UnixNano())
}

// Done is signaled when the source record channel is closed.
func (s *StreamReport) Done() <-chan struct{} { return s.doneCh }

// Collect consumes Records until the channel closes, then closes doneCh.
func (s *StreamReport) Collect(in <-chan *Record) {
	latencySecTemp := &stats{}

	// ticker keeps the rolling 1-sec window fresh
	go func() {
		start := time.Unix(0, atomic.LoadInt64(&s.startNanos))
		t := time.NewTicker(time.Second)
		var lastCount int64
		lastTime := start
		for {
			select {
			case <-t.C:
				s.mu.Lock()
				dc := s.latencyStats.count - lastCount
				if dc > 0 {
					rps := float64(dc) / time.Since(lastTime).Seconds()
					s.rpsStats.update(rps)
					lastCount = s.latencyStats.count
					lastTime = time.Now()
					*s.latencyWithinSec = *latencySecTemp
					s.rpsWithinSec = rps
					latencySecTemp.reset()
					s.noDataWithinSec = false
				} else {
					s.noDataWithinSec = true
				}
				s.mu.Unlock()
			case <-s.doneCh:
				return
			}
		}
	}()

	for r := range in {
		s.mu.Lock()
		latencySecTemp.update(float64(r.Cost))
		s.latencyQuantile.Insert(float64(r.Cost))
		s.latencyHistogram.Insert(float64(r.Cost))
		s.latencyStats.update(float64(r.Cost))
		if r.Code != 0 {
			s.codes[r.Code]++
		}
		if r.Error != "" {
			s.errors[r.Error]++
		}
		s.readBytes = r.ReadBytes
		s.writeBytes = r.WriteBytes
		s.concurrency = r.ConcurrencyCount
		s.mu.Unlock()
	}
	close(s.doneCh)
}

// Snapshot returns a thread-safe copy of the current state. The TUI
// calls this from its Tick handler.
func (s *StreamReport) Snapshot() *Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	start := time.Unix(0, atomic.LoadInt64(&s.startNanos))
	elapsed := time.Since(start)
	if elapsed < 0 {
		elapsed = 0
	}
	snap := &Snapshot{
		Elapsed:         elapsed,
		Count:           s.latencyStats.count,
		Codes:           make(map[string]int64, 5),
		ExactCodes:      copyIntMap(s.codes),
		Errors:          copyStrMap(s.errors),
		ReadMBps:        float64(s.readBytes) / 1024 / 1024 / elapsed.Seconds(),
		WriteMBps:       float64(s.writeBytes) / 1024 / 1024 / elapsed.Seconds(),
		ConcurrencyUsed: s.concurrency,
	}
	if snap.ConcurrencyUsed == 0 {
		snap.ConcurrencyUsed = 1
	}
	if elapsed > 0 {
		snap.RPS = float64(snap.Count) / elapsed.Seconds()
	}
	for code, n := range s.codes {
		lbl, ok := statusSectionMap[code/100]
		if !ok {
			lbl = "other"
		}
		snap.Codes[lbl] += n
	}
	snap.Latency.Min = dur(s.latencyStats.min)
	snap.Latency.Mean = dur(s.latencyStats.mean())
	snap.Latency.StdDev = dur(s.latencyStats.stddev())
	snap.Latency.Max = dur(s.latencyStats.max)
	snap.RPSStats.Min = s.rpsStats.min
	snap.RPSStats.Mean = s.rpsStats.mean()
	snap.RPSStats.StdDev = s.rpsStats.stddev()
	snap.RPSStats.Max = s.rpsStats.max

	// sort percentiles by P ascending (guaranteed by quantiles slice, but
	// we hand the result to the table renderer; keep stable for tests).
	for _, p := range quantiles {
		snap.Percentiles = append(snap.Percentiles, Percentile{
			P:       p,
			Latency: dur(s.latencyQuantile.Query(p)),
		})
	}
	for _, b := range s.latencyHistogram.Bins() {
		snap.Histograms = append(snap.Histograms, HistogramBin{
			Mean:  dur(b.Mean()),
			Count: b.Count,
		})
	}
	sort.Sort(binsByCount(snap.Histograms))
	// after sorting, restore mean-magnitude order (low→high latency)
	sort.SliceStable(snap.Histograms, func(i, j int) bool {
		return snap.Histograms[i].Mean < snap.Histograms[j].Mean
	})
	return snap
}

// SecondsSnapshot returns the 1-second rolling view used by the live
// charts. Returns nil if the previous second saw no requests.
func (s *StreamReport) SecondsSnapshot() *Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.noDataWithinSec {
		return nil
	}
	out := &Snapshot{
		Count:           s.latencyWithinSec.count,
		ReadMBps:        0, // 1-sec window; throughput per sample kept simple
		WriteMBps:       0,
		ConcurrencyUsed: s.concurrency,
		RPS:             s.rpsWithinSec,
		Latency: struct{ Min, Mean, StdDev, Max time.Duration }{
			Min:    dur(s.latencyWithinSec.min),
			Mean:   dur(s.latencyWithinSec.mean()),
			StdDev: dur(s.latencyWithinSec.stddev()),
			Max:    dur(s.latencyWithinSec.max),
		},
		ExactCodes: copyIntMap(s.codes),
	}
	return out
}

// helpers
func dur(f float64) time.Duration {
	if f < 0 {
		return 0
	}
	return time.Duration(f)
}
func copyIntMap(m map[int]int64) map[int]int64 {
	out := make(map[int]int64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
func copyStrMap(m map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// sqrt is a local wrapper so callers don't need to import math directly.
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Sqrt(x)
}

// binsByCount is a small adapter so we can sort bins descending by count
// for the histogram view (the heavy buckets on the left).
type binsByCount []HistogramBin

func (b binsByCount) Less(i, j int) bool { return b[i].Count > b[j].Count }
func (b binsByCount) Len() int           { return len(b) }
func (b binsByCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
