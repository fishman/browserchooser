package main

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// state is the frecency data we persist across runs. Counts is global per
// browser; Hosts is per-host per browser, so a site you always open in one
// browser is remembered without a configuration dialog.
type state struct {
	Counts map[string]useStat            `json:"counts"`
	Hosts  map[string]map[string]useStat `json:"hosts"`
}

func loadState() *state {
	s := &state{Counts: map[string]useStat{}, Hosts: map[string]map[string]useStat{}}
	data, err := os.ReadFile(statePath())
	if err != nil {
		return s
	}
	json.Unmarshal(data, s)
	if s.Counts == nil {
		s.Counts = map[string]useStat{}
	}
	if s.Hosts == nil {
		s.Hosts = map[string]map[string]useStat{}
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

func recordOpen(link, id string) {
	s := loadState()
	g := s.Counts[id]
	g.Count++
	g.Last = time.Now().Unix()
	s.Counts[id] = g
	if h := hostOf(link); h != "" {
		hs := s.Hosts[h]
		if hs == nil {
			hs = map[string]useStat{}
		}
		st := hs[id]
		st.Count++
		st.Last = time.Now().Unix()
		hs[id] = st
		s.Hosts[h] = hs
	}
	saveState(s)
}

// hostOf returns the lowercase hostname of a link, stripped of a leading
// "www." so github.com and www.github.com track as one site. Links without a
// scheme get one so url.Parse can find the host; unparseable input yields "".
func hostOf(link string) string {
	if !strings.Contains(link, "://") && !strings.HasPrefix(link, "//") {
		link = "https://" + link
	}
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
}

// dominantHostID returns the browser id used for at least 2/3 of a host's
// opens, once the host has been opened at least minHostUses times. Below that
// there is no reliable signal, so the picker shows as usual.
func dominantHostID(counts map[string]useStat) string {
	var total, top int
	var topID string
	for id, st := range counts {
		total += st.Count
		if st.Count > top {
			top, topID = st.Count, id
		}
	}
	if total < minHostUses || top*3 < total*2 {
		return ""
	}
	return topID
}
