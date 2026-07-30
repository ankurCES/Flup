// Package profiles provides persistent named presets for benchmark
// configurations. Profiles are stored as a JSON file in the user's
// config directory (~/.config/flup/profiles.json).
package profiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ankurCES/Flup/internal/bench"
)

// Profile is a named, saved benchmark configuration.
type Profile struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Config      bench.Config `json:"config"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Store manages a collection of named profiles on disk.
type Store struct {
	path     string
	profiles map[string]*Profile
}

// Open loads or creates the profile store.
func Open() (*Store, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "profiles.json")
	s := &Store{
		path:     path,
		profiles: make(map[string]*Profile),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return nil, err
	}
	if len(data) > 0 {
		var profiles []*Profile
		if err := json.Unmarshal(data, &profiles); err != nil {
			return nil, fmt.Errorf("corrupt profiles.json: %w", err)
		}
		for _, p := range profiles {
			s.profiles[p.Name] = p
		}
	}
	return s, nil
}

// Save persists a profile. Overwrites if name exists.
func (s *Store) Save(p Profile) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return errors.New("profile name is required")
	}
	now := time.Now()
	if existing, ok := s.profiles[p.Name]; ok {
		p.CreatedAt = existing.CreatedAt
	} else {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	s.profiles[p.Name] = &p
	return s.flush()
}

// Delete removes a profile by name.
func (s *Store) Delete(name string) error {
	if _, ok := s.profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	delete(s.profiles, name)
	return s.flush()
}

// Get retrieves a profile by name.
func (s *Store) Get(name string) (*Profile, bool) {
	p, ok := s.profiles[name]
	return p, ok
}

// List returns all profiles sorted by name.
func (s *Store) List() []*Profile {
	result := make([]*Profile, 0, len(s.profiles))
	for _, p := range s.profiles {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Count returns how many profiles are stored.
func (s *Store) Count() int { return len(s.profiles) }

func (s *Store) flush() error {
	profiles := s.List()
	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, err2 := os.UserHomeDir()
		if err2 != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "flup"), nil
	}
	return filepath.Join(dir, "flup"), nil
}
