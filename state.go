package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// stateDir returns the per-user state dir for this app: XDG_STATE_HOME on
// Linux, Application Support on macOS, LOCALAPPDATA on Windows. Frecency is
// runtime state, not config, so it does not belong under .config.
func stateDir() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		if la := os.Getenv("LOCALAPPDATA"); la != "" {
			return filepath.Join(la, "browserchooser")
		}
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "browserchooser")
	default:
		if x := os.Getenv("XDG_STATE_HOME"); x != "" {
			return filepath.Join(x, "browserchooser")
		}
		return filepath.Join(home, ".local", "state", "browserchooser")
	}
	return filepath.Join(home, ".browserchooser")
}

func statePath() string {
	return filepath.Join(stateDir(), "state.json")
}

// state is the frecency data we persist across runs.
type state struct {
	Counts map[string]useStat `json:"counts"`
}

func loadState() *state {
	s := &state{Counts: map[string]useStat{}}
	data, err := os.ReadFile(statePath())
	if err != nil {
		return s
	}
	json.Unmarshal(data, s)
	if s.Counts == nil {
		s.Counts = map[string]useStat{}
	}
	return s
}

func saveState(s *state) {
	if err := os.MkdirAll(stateDir(), 0o755); err != nil {
		return
	}
	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	os.WriteFile(statePath(), data, 0o600)
}

func recordUse(id string) {
	s := loadState()
	st := s.Counts[id]
	st.Count++
	st.Last = time.Now().Unix()
	s.Counts[id] = st
	saveState(s)
}
