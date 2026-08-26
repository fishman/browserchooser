package main

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type settings struct {
	Firefox struct {
		Profiles bool `toml:"profiles"`
	} `toml:"firefox"`
}

func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "browserchooser", "config.toml")
}

// loadSettings reads the optional config.toml. A missing or malformed file
// yields defaults: firefox profiles are listed unless explicitly disabled.
func loadSettings() settings {
	s := settings{}
	s.Firefox.Profiles = true
	p := settingsPath()
	if p == "" {
		return s
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return s
	}
	// Absent keys keep their current value, so an explicit profiles=false
	// overrides the default while a missing key leaves it enabled.
	if toml.Unmarshal(data, &s) != nil {
		return s
	}
	return s
}
