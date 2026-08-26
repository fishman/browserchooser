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
// yields zero-value settings, i.e. every optional feature is off.
func loadSettings() settings {
	p := settingsPath()
	if p == "" {
		return settings{}
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return settings{}
	}
	var s settings
	if toml.Unmarshal(data, &s) != nil {
		return settings{}
	}
	return s
}
