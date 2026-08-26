package main

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type settings struct {
	// Theme is the color scheme: "auto" follows the system (default),
	// "light" and "dark" force a variant.
	Theme   string `toml:"theme"`
	Firefox struct {
		Profiles bool `toml:"profiles"`
	} `toml:"firefox"`
	Chrome struct {
		Profiles bool                  `toml:"profiles"`
		Browsers []chromeBrowserConfig `toml:"browsers"`
	} `toml:"chrome"`
	// Rules route URLs to a browser; the first matching rule wins.
	Rules []rule `toml:"rules"`
}

// chromeBrowserConfig adds a Chromium-family browser whose profiles are read
// from its "Local State", so forks (Brave, Edge, ...) need no code change.
// DataDir and Binary are optional on Linux: the data dir is probed from
// Name/Binary-derived candidates and the launcher from PATH (macOS uses MacBin
// or the Name-derived app path). Only Name is strictly required.
type chromeBrowserConfig struct {
	Name    string `toml:"name"`
	DataDir string `toml:"data_dir"`
	Binary  string `toml:"binary"`
	MacBin  string `toml:"mac_binary"`
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
	s.Theme = "auto"
	s.Firefox.Profiles = true
	s.Chrome.Profiles = true
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
