// Package history persists completed benchmark runs as JSON under the
// user's config dir so the History tab can list / re-open them later.
package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ankurCES/Flup/internal/bench"
)

// Entry is one persisted run. Used by the History tab and saved on Stop.
type Entry struct {
	ID       string         `json:"id"`
	Finished time.Time      `json:"finished"`
	Config   bench.Config   `json:"config"`
	Snapshot bench.Snapshot `json:"snapshot"`
}

// Dir resolves the default history directory, ensuring it exists.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(base, "flup", "history")
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return d, nil
}

// Store is a goroutine-safe in-memory cache of persisted entries.
type Store struct {
	dir string
	mu  sync.Mutex
	all []Entry
}

// Open enumerates the history directory and loads every entry.
func Open() (*Store, error) {
	d, err := Dir()
	if err != nil {
		return nil, err
	}
	s := &Store{dir: d}
	files, err := os.ReadDir(d)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(d, f.Name()))
		if err != nil {
			continue
		}
		var e Entry
		if err := json.Unmarshal(b, &e); err != nil {
			continue
		}
		s.all = append(s.all, e)
	}
	sort.Slice(s.all, func(i, j int) bool {
		return s.all[i].Finished.After(s.all[j].Finished)
	})
	return s, nil
}

// All returns a copy of the entries, newest first.
func (s *Store) All() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, len(s.all))
	copy(out, s.all)
	return out
}

// Save writes a new entry to disk and adds it to the cache.
func (s *Store) Save(e Entry) error {
	if e.ID == "" {
		e.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if e.Finished.IsZero() {
		e.Finished = time.Now()
	}
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	p := filepath.Join(s.dir, e.ID+".json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		return err
	}
	s.mu.Lock()
	s.all = append([]Entry{e}, s.all...)
	s.mu.Unlock()
	return nil
}

// Delete removes an entry from disk and from the cache.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	for i, e := range s.all {
		if e.ID == id {
			s.all = append(s.all[:i], s.all[i+1:]...)
			break
		}
	}
	s.mu.Unlock()
	p := filepath.Join(s.dir, id+".json")
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
