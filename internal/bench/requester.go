package bench

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"
	"golang.org/x/time/rate"
)

// Requester is the load-generation engine. Run() spawns Concurrency
// goroutines, each driving fasthttp requests in a loop bounded by the
// configured Duration and/or Requests. Records flow on Rec() and the
// channel is closed when all goroutines exit.
type Requester struct {
	Cfg    Config
	client *fasthttp.Client

	recCh chan *Record
	errCh chan error

	inflight int64 // current concurrent requests
	maxReqs  int64
}

// New constructs a tuned Requester.
func New(cfg Config) (*Requester, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	c := &fasthttp.Client{
		NoDefaultUserAgentHeader:      true,
		DisableHeaderNamesNormalizing: true,
		DisablePathNormalizing:        true,
	}
	if cfg.Timeout > 0 {
		c.MaxConnDuration = cfg.Timeout
	}
	if cfg.DialTimeout > 0 {
		// fasthttp's Dial; v1.55 supports MaxIdleConnDuration but
		// dial timeout is exposed via Dial
	}
	if cfg.Insecure {
		c.TLSConfig.InsecureSkipVerify = true
	}
	if cfg.UnixSocket != "" {
		c.Dial = func(addr string) (net.Conn, error) {
			return net.DialTimeout("unix", cfg.UnixSocket, dialTimeoutOr(cfg.DialTimeout, 30*time.Second))
		}
	}
	return &Requester{
		Cfg:    cfg,
		client: c,
		recCh:  make(chan *Record, 4096),
		errCh:  make(chan error, 1),
	}, nil
}

func dialTimeoutOr(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}

// Rec is the channel the StreamReport consumes. Buffered so the hot path
// never blocks the requester unless the UI is hopelessly stuck.
func (r *Requester) Rec() <-chan *Record { return r.recCh }

// Errors returns fatal startup errors (DNS failure, bad URL). Non-fatal
// per-request errors arrive as Record.Error.
func (r *Requester) Errors() <-chan error { return r.errCh }

// Run drives the benchmark until ctx is done, or Duration / Requests met.
func (r *Requester) Run(ctx context.Context) error {
	r.maxReqs = r.Cfg.Requests

	// pre-compute request — reused across all goroutines for memory efficiency
	req, err := r.buildRequest()
	if err != nil {
		return err
	}

	var (
		limiter *rate.Limiter
		budget  int64
	)
	if r.Cfg.RatePtr != nil {
		limiter = rate.NewLimiter(*r.Cfg.RatePtr, int(r.Cfg.Concurrency))
	}
	if r.Cfg.Requests > 0 {
		budget = r.Cfg.Requests
	}

	var wg sync.WaitGroup

	// optional ramp-up: open a new connection every second until target reached
	startConns := 1
	if r.Cfg.RampUp > 0 {
		startConns = 1
	}

	for i := 0; i < startConns; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			r.runWorker(ctx, idx, req, limiter, &budget)
		}(i)
	}

	if r.Cfg.RampUp > 0 {
		// incremental ramp until we reach target concurrency
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		opened := startConns
		for {
			select {
			case <-ctx.Done():
				wg.Wait()
				close(r.recCh)
				return nil
			case <-ticker.C:
				if opened >= r.Cfg.Concurrency {
					ticker.Stop()
					continue
				}
				for i := opened; i < r.Cfg.Concurrency && i < opened+r.Cfg.RampUp; i++ {
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						r.runWorker(ctx, idx, req, limiter, &budget)
					}(i)
					opened = i + 1
				}
			}
		}
	}

	wg.Wait()
	close(r.recCh)
	return nil
}

func (r *Requester) runWorker(ctx context.Context, idx int, req *fasthttp.Request, limiter *rate.Limiter, budget *int64) {
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	for {
		// Stop conditions — checked at the top of every iteration.
		if ctx.Err() != nil {
			return
		}
		if r.Cfg.Duration > 0 {
			// checked inside the worker; cheap and clean
		}
		if budget != nil && atomic.AddInt64(budget, -1) < 0 {
			atomic.AddInt64(budget, 1)
			return
		}

		if limiter != nil {
			if err := limiter.Wait(ctx); err != nil {
				return
			}
		}

		atomic.AddInt64(&r.inflight, 1)
		start := time.Now()
		err := r.client.Do(req, resp)
		cost := time.Since(start)
		atomic.AddInt64(&r.inflight, -1)

		rec := &Record{
			Cost:             cost,
			ConcurrencyCount: int(atomic.LoadInt64(&r.inflight)),
		}
		if err == nil {
			rec.Code = resp.StatusCode()
			rec.ReadBytes = int64(len(resp.Body()))
		} else {
			rec.Error = simplifyErr(err)
		}
		// write-byte estimate: header + body length of request
		hdrBytes := 0
		req.Header.VisitAll(func(k, v []byte) {
			hdrBytes += len(k) + len(v) + 4 // ": \r\n"
		})
		rec.WriteBytes = int64(hdrBytes + len(req.Body()))

		select {
		case r.recCh <- rec:
		case <-ctx.Done():
			return
		}
	}
}

// buildRequest constructs the fasthttp request once and reuses it.
func (r *Requester) buildRequest() (*fasthttp.Request, error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(r.Cfg.Method)
	if r.Cfg.Host != "" {
		req.Header.SetHost(r.Cfg.Host)
	}
	for _, h := range r.Cfg.Headers {
		// split "K: V" exactly once; plow behavior
		for i := 0; i < len(h); i++ {
			if h[i] == ':' {
				k := h[:i]
				v := h[i+1:]
				if len(v) > 0 && v[0] == ' ' {
					v = v[1:]
				}
				req.Header.Set(k, v)
				break
			}
		}
	}
	if r.Cfg.ContentType != "" {
		req.Header.Set("Content-Type", r.Cfg.ContentType)
	}

	// URL: parse with fasthttp's URI struct
	u := fasthttp.AcquireURI()
	if err := u.Parse(nil, []byte(r.Cfg.URL)); err != nil {
		fasthttp.ReleaseURI(u)
		return nil, fmt.Errorf("bad URL %q: %w", r.Cfg.URL, err)
	}
	req.SetURI(u)
	// URL is parsed once; release uri after copy
	fasthttp.ReleaseURI(u)

	if r.Cfg.Body != "" {
		if r.Cfg.Body[0] == '@' {
			path := r.Cfg.Body[1:]
			f, err := os.Open(path)
			if err != nil {
				return nil, fmt.Errorf("body file: %w", err)
			}
			if r.Cfg.StreamBody {
				// chunked streaming: hand fasthttp a Body setter
				req.SetBodyStream(f, -1)
			} else {
				b, err := io.ReadAll(f)
				f.Close()
				if err != nil {
					return nil, err
				}
				req.SetBody(b)
			}
		} else {
			req.SetBody([]byte(r.Cfg.Body))
		}
	}
	return req, nil
}

// simplifyErr compresses fasthttp's error trees into short stable labels
// for the error view (e.g. "connection refused", "timeout").
func simplifyErr(err error) string {
	if err == nil {
		return ""
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "timeout"
	}
	s := err.Error()
	switch {
	case contains(s, "connection refused"):
		return "connection refused"
	case contains(s, "no such host"):
		return "dns"
	case contains(s, "EOF"):
		return "eof"
	case contains(s, "reset by peer"):
		return "reset"
	case contains(s, "tls: "):
		return "tls"
	}
	if len(s) > 60 {
		return s[:60] + "…"
	}
	return s
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

// indexOf avoids pulling in strings just for one search.
func indexOf(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
