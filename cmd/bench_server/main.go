package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// bench_server is a tiny echo HTTP server used for smoke-testing Flup.
// Not part of the shipped binary — lives in cmd/bench_server.

func main() {
	addr := flag.String("addr", ":18080", "listen address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// minimal latency injection so percentile view has variety
		switch r.URL.Path {
		case "/slow":
			time.Sleep(50 * time.Millisecond)
		case "/slow2":
			time.Sleep(120 * time.Millisecond)
		}
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("bench_server listening on %s", *addr)
	log.Fatal(srv.ListenAndServe())
}
