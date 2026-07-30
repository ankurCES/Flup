package tui

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ankurCES/Flup/internal/bench"
)

// benchRequester is the common subset both Requester and RequesterH2 expose.
type benchRequester interface {
	Rec() <-chan *bench.Record
	Errors() <-chan error
	Run(ctx context.Context) error
}

// Runner owns the live benchmark lifecycle: the configured requester,
// its goroutine pool, the snapshot report, and a single cancel func the
// TUI uses to Stop(). A second Start() while running is a no-op.
type Runner struct {
	mu       sync.Mutex
	cancel   context.CancelFunc
	req      benchRequester
	report   *bench.StreamReport
	state    int32 // 0 idle, 1 running, 2 stopping
	startCh  chan struct{}
	finalCh  chan struct{}

	// Latest snapshot — read by views, written by the dispatcher.
	lastMu sync.RWMutex
	last   *bench.Snapshot

	historySaved bool
	cfg          bench.Config
}

// NewRunner constructs an idle runner.
func NewRunner() *Runner { return &Runner{} }

// IsRunning reports whether a benchmark is in flight.
func (r *Runner) IsRunning() bool { return atomic.LoadInt32(&r.state) == 1 }

// Start kicks off a benchmark. Returns an error if config is invalid or
// if a benchmark is already in flight.
func (r *Runner) Start(cfg bench.Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if atomic.LoadInt32(&r.state) != 0 {
		return nil // already running
	}

	// Choose requester: HTTP/2 uses net/http; default uses fasthttp.
	var req benchRequester
	if cfg.HTTP2 {
		h2, err := bench.NewH2(cfg)
		if err != nil {
			return err
		}
		req = h2
	} else {
		fh, err := bench.New(cfg)
		if err != nil {
			return err
		}
		req = fh
	}
	rep := bench.NewStreamReport()
	rep.Start()

	// Enforce cfg.Duration: use WithTimeout so the context auto-cancels
	// when the configured duration elapses.
	var ctx context.Context
	var cancel context.CancelFunc
	if cfg.Duration > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), cfg.Duration)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	r.cancel = cancel
	r.req = req
	r.report = rep
	r.cfg = cfg
	r.historySaved = false
	atomic.StoreInt32(&r.state, 1)
	r.startCh = make(chan struct{})
	r.finalCh = make(chan struct{})

	go func() {
		rep.Collect(req.Rec())
		close(r.finalCh)
		atomic.StoreInt32(&r.state, 0)
	}()

	go func() {
		_ = req.Run(ctx)
		cancel()
	}()

	// Pump snapshots every ~250ms into last; the views read it.
	go r.pump()
	return nil
}

// Stop signals the benchmark to wind down and returns a channel that
// closes when the final record has been ingested.
func (r *Runner) Stop() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
	}
	if r.state == 0 {
		// already idle — return a never-closing channel? better: a pre-closed one
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	atomic.StoreInt32(&r.state, 2)
	if r.finalCh == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return r.finalCh
}

// Snapshot returns the most recent poll (or nil if none yet).
func (r *Runner) Snapshot() *bench.Snapshot {
	r.lastMu.RLock()
	defer r.lastMu.RUnlock()
	return r.last
}

// Config returns the cfg used by the current/last run.
func (r *Runner) Config() bench.Config {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cfg
}

// pump copies the report's view every 250ms.
func (r *Runner) pump() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if r.report != nil {
				s := r.report.Snapshot()
				r.lastMu.Lock()
				r.last = s
				r.lastMu.Unlock()
			}
		case <-r.finalCh:
			if r.report != nil {
				s := r.report.Snapshot()
				r.lastMu.Lock()
				r.last = s
				r.lastMu.Unlock()
			}
			return
		}
	}
}
