package project

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const StateName = "state.yml"

// State holds what Runivra remembers between commands for one person on one
// computer. Unlike Config it is not meant to be shared.
type State struct {
	// Databases maps a set of custom addons folders to the database last
	// chosen while those folders were in use.
	Databases map[string]string `yaml:"databases,omitempty"`
}

// Database returns the database remembered for an addons key, or "".
func (s State) Database(key string) string {
	return s.Databases[key]
}

func (s *State) SetDatabase(key string, name string) {
	if s.Databases == nil {
		s.Databases = map[string]string{}
	}
	s.Databases[key] = name
}

func statePath(root string) string {
	return filepath.Join(root, DirName, StateName)
}

// LoadState returns an empty State when nothing has been remembered yet.
func LoadState(root string) (State, error) {
	var state State
	data, err := os.ReadFile(statePath(root))
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	err = yaml.Unmarshal(data, &state)
	return state, err
}

func SaveState(root string, state State) error {
	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, DirName), 0o755); err != nil {
		return err
	}
	// Keep the personal state out of the project's Git history.
	ignore := filepath.Join(root, DirName, ".gitignore")
	if _, err := os.Stat(ignore); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(ignore, []byte(StateName+"\n"), 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(statePath(root), data, 0o644)
}
