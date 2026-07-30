package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ankurCES/Flup/internal/bench"
)

func testSnapshot() *bench.Snapshot {
	return &bench.Snapshot{
		Elapsed:         10 * time.Second,
		Count:           5000,
		Codes:           map[string]int64{"2xx": 4950, "5xx": 50},
		ExactCodes:      map[int]int64{200: 4950, 500: 50},
		Errors:          map[string]int64{"timeout": 5},
		RPS:             500.0,
		ReadMBps:        1.5,
		WriteMBps:       0.3,
		ConcurrencyUsed: 20,
		Percentiles: []bench.Percentile{
			{P: 0.50, Latency: 2 * time.Millisecond},
			{P: 0.99, Latency: 15 * time.Millisecond},
		},
		Histograms: []bench.HistogramBin{
			{Mean: 2 * time.Millisecond, Count: 3000},
			{Mean: 10 * time.Millisecond, Count: 2000},
		},
	}
}

func testConfig() bench.Config {
	return bench.Config{
		URL:         "http://localhost:8080",
		Method:      "GET",
		Concurrency: 20,
		Duration:    10 * time.Second,
	}
}

func TestNewReport(t *testing.T) {
	r := NewReport(testConfig(), testSnapshot())
	if r.URL != "http://localhost:8080" {
		t.Fatalf("URL = %q", r.URL)
	}
	if r.Count != 5000 {
		t.Fatalf("Count = %d", r.Count)
	}
	if len(r.Percentiles) != 2 {
		t.Fatalf("Percentiles len = %d", len(r.Percentiles))
	}
}

func TestToJSON(t *testing.T) {
	r := NewReport(testConfig(), testSnapshot())
	b, err := r.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["url"] != "http://localhost:8080" {
		t.Fatalf("url = %v", m["url"])
	}
}

func TestToCSV(t *testing.T) {
	r := NewReport(testConfig(), testSnapshot())
	b, err := r.ToCSV()
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "url,method,concurrency") {
		t.Fatal("missing header")
	}
	if !strings.Contains(s, "http://localhost:8080") {
		t.Fatal("missing URL in data")
	}
}

func TestToMarkdown(t *testing.T) {
	r := NewReport(testConfig(), testSnapshot())
	b, err := r.ToMarkdown()
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "# Flup Benchmark Report") {
		t.Fatal("missing title")
	}
	if !strings.Contains(s, "P50") {
		t.Fatal("missing percentile label")
	}
}

func TestWriteFile(t *testing.T) {
	r := NewReport(testConfig(), testSnapshot())
	dir := t.TempDir()

	for _, fmt := range []Format{JSON, CSV, Markdown} {
		path, err := WriteFile(r, fmt, dir)
		if err != nil {
			t.Fatalf("WriteFile(%v): %v", fmt, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("empty file for format %v", fmt)
		}
		ext := filepath.Ext(path)
		if ext != "."+fmt.String() {
			t.Fatalf("ext = %q, want .%s", ext, fmt.String())
		}
	}
}
