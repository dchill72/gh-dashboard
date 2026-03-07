package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type State struct {
	Read map[string]bool `json:"read"`
	path string
}

func New() *State {
	return &State{Read: make(map[string]bool)}
}

func Load() (*State, error) {
	path, err := statePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		s := New()
		s.path = path
		return s, nil
	}
	if err != nil {
		return nil, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Read == nil {
		s.Read = make(map[string]bool)
	}
	s.path = path
	return &s, nil
}

func (s *State) Save() error {
	if s.path == "" {
		path, err := statePath()
		if err != nil {
			return err
		}
		s.path = path
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *State) MarkRead(id string) {
	s.Read[id] = true
}

func (s *State) ToggleRead(id string) {
	s.Read[id] = !s.Read[id]
}

func (s *State) IsRead(id string) bool {
	return s.Read[id]
}

func statePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "gh-dashboard")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}
