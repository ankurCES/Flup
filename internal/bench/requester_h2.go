package bench

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// RequesterH2 is an HTTP/2-capable requester built on net/http (stdlib).
// It has the same API surface as Requester so the TUI can swap them
// transparently. Use this when the target server speaks HTTP/2.
type RequesterH2 struct {
	Cfg    Config
	client *http.Client

	recCh chan *Record
	errCh chan error

	inflight int64
}

// NewH2 constructs an HTTP/2-capable requester.
func NewH2(cfg Config) (*RequesterH2, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	transport := &http.Transport{
		MaxIdleConns:        cfg.Concurrency * 2,
		MaxIdleConnsPerHost: cfg.Concurrency * 2,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: cfg.Insecure},
	}
	if cfg.DialTimeout > 0 {
		transport.DialContext = (&net.Dialer{Timeout: cfg.DialTimeout}).DialContext
	}
	if cfg.UnixSocket != "" {
		transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: dialTimeoutOr(cfg.DialTimeout, 30*time.Second)}).DialContext(ctx, "unix", cfg.UnixSocket)
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	return &RequesterH2{
		Cfg:    cfg,
		client: client,
		recCh:  make(chan *Record, 4096),
		errCh:  make(chan error, 1),
	}, nil
}

// Rec returns the record channel.
func (r *RequesterH2) Rec() <-chan *Record { return r.recCh }

// Errors returns fatal startup errors.
func (r *RequesterH2) Errors() <-chan error { return r.errCh }

// Run drives the benchmark.
func (r *RequesterH2) Run(ctx context.Context) error {
	body, bodyLen, err := r.readBody()
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
	for i := 0; i < r.Cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.runWorker(ctx, body, bodyLen, limiter, &budget)
		}()
	}
	wg.Wait()
	close(r.recCh)
	return nil
}

func (r *RequesterH2) runWorker(ctx context.Context, body []byte, bodyLen int, limiter *rate.Limiter, budget *int64) {
	for {
		if ctx.Err() != nil {
			return
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

		req, err := r.buildHTTPRequest(ctx, body)
		if err != nil {
			return
		}

		atomic.AddInt64(&r.inflight, 1)
		start := time.Now()
		resp, err := r.client.Do(req)
		cost := time.Since(start)
		atomic.AddInt64(&r.inflight, -1)

		rec := &Record{
			Cost:             cost,
			ConcurrencyCount: int(atomic.LoadInt64(&r.inflight)),
		}
		if err == nil {
			rec.Code = resp.StatusCode
			n, _ := io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			rec.ReadBytes = n
		} else {
			rec.Error = simplifyErr(err)
		}
		rec.WriteBytes = int64(bodyLen + 200) // estimate: headers + body

		select {
		case r.recCh <- rec:
		case <-ctx.Done():
			return
		}
	}
}

func (r *RequesterH2) buildHTTPRequest(ctx context.Context, body []byte) (*http.Request, error) {
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = newResetReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, r.Cfg.Method, r.Cfg.URL, bodyReader)
	if err != nil {
		return nil, err
	}
	if r.Cfg.Host != "" {
		req.Host = r.Cfg.Host
	}
	for _, h := range r.Cfg.Headers {
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
	return req, nil
}

func (r *RequesterH2) readBody() ([]byte, int, error) {
	if r.Cfg.Body == "" {
		return nil, 0, nil
	}
	if r.Cfg.Body[0] == '@' {
		path := r.Cfg.Body[1:]
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, 0, err
		}
		return b, len(b), nil
	}
	b := []byte(r.Cfg.Body)
	return b, len(b), nil
}

// resetReader wraps a byte slice as a reusable io.Reader.
type resetReader struct {
	data []byte
	pos  int
}

func newResetReader(b []byte) *resetReader { return &resetReader{data: b} }
func (r *resetReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
