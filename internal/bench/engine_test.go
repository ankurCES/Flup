package bench

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestRequesterHitLocalServer runs a small benchmark against an in-memory
// HTTP server and asserts the snapshot has sane numbers.
func TestRequesterHitLocalServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	cfg := Config{
		URL:         srv.URL + "/",
		Method:      "GET",
		Concurrency: 4,
		Requests:    200,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	r, err := New(cfg)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	rep := NewStreamReport()
	rep.Start()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); rep.Collect(r.Rec()) }()
	go func() { defer wg.Done(); _ = r.Run(context.Background()) }()
	wg.Wait()

	snap := rep.Snapshot()
	if snap == nil {
		t.Fatal("nil snapshot")
	}
	if snap.Count != 200 {
		t.Fatalf("expected 200 records, got %d", snap.Count)
	}
	if snap.Codes["2xx"] != 200 {
		t.Fatalf("expected 200 2xx, got %d", snap.Codes["2xx"])
	}
	if snap.RPS <= 0 {
		t.Fatalf("expected positive RPS, got %f", snap.RPS)
	}
	if snap.Latency.Mean < time.Millisecond {
		t.Fatalf("expected latency >= 1ms, got %s", snap.Latency.Mean)
	}
}

// TestPace ensures the rate parser accepts the plow formats.
func TestPace(t *testing.T) {
	for _, s := range []string{"", "infinity", "inf", "50", "50/1s", "10/ms", "1/h"} {
		if _, err := paceToLimit(s); err != nil {
			t.Errorf("pace %q: %v", s, err)
		}
	}
}

// TestPercentilesHaveAllBands checks that the streaming quantile emits a
// value for every configured band.
func TestPercentilesHaveAllBands(t *testing.T) {
	rep := NewStreamReport()
	rep.Start()
	go rep.Collect(nil) // immediate close
	for i := 0; i < 1000; i++ {
		rep.latencyQuantile.Insert(float64(i) * 1000) // ns
	}
	snap := rep.Snapshot()
	if len(snap.Percentiles) != len(quantiles) {
		t.Fatalf("expected %d percentiles, got %d", len(quantiles), len(snap.Percentiles))
	}
	if snap.Percentiles[0].Latency >= snap.Percentiles[len(snap.Percentiles)-1].Latency {
		t.Fatalf("percentile order wrong: p50=%s p99.99=%s",
			snap.Percentiles[0].Latency, snap.Percentiles[len(snap.Percentiles)-1].Latency)
	}
}
