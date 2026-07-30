package tui

import (
	"strconv"
	"sync"

	"github.com/ankurCES/Flup/internal/bench"
)

// shared state held by the root app and read by views. Views are mostly
// stateless but a few (Live spark, Run screen timer) need cross-frame
// data, which is why this file exists.

var (
	snapMu sync.RWMutex
	lastS  *bench.Snapshot
	useSec bool
)

func setLastSnap(s *bench.Snapshot) {
	snapMu.Lock()
	lastS = s
	snapMu.Unlock()
}

func lastSnap() *bench.Snapshot {
	snapMu.RLock()
	defer snapMu.RUnlock()
	return lastS
}

func setUseSec(v bool) {
	snapMu.Lock()
	useSec = v
	snapMu.Unlock()
}

func lastUseSec() bool {
	snapMu.RLock()
	defer snapMu.RUnlock()
	return useSec
}

func itoa(n int) string { return strconv.Itoa(n) }
