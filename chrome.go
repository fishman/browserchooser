package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
)

// chromeBase is the "User Data" dir that holds Local State and one subdir per
// profile (Default, Profile 1, ...).
func chromeBase() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
	case "windows":
		if app := os.Getenv("LOCALAPPDATA"); app != "" {
			return filepath.Join(app, "Google", "Chrome", "User Data")
		}
	}
	return filepath.Join(home, ".config", "google-chrome")
}

func chromeBin() string {
	switch runtime.GOOS {
	case "darwin":
		return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	case "windows":
		if pf := os.Getenv("ProgramFiles"); pf != "" {
			if p := filepath.Join(pf, "Google", "Chrome", "Application", "chrome.exe"); exists(p) {
				return p
			}
		}
		return "chrome"
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "google-chrome-beta", "google-chrome-unstable"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return "google-chrome"
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func chromeProfiles() []browser {
	base := chromeBase()
	data, err := os.ReadFile(filepath.Join(base, "Local State"))
	if err != nil {
		return nil
	}
	return parseChromeProfiles(data, base, chromeBin())
}

// parseChromeProfiles turns a Chrome "Local State" JSON into one browser per
// profile, launched with --profile-directory=<dir>. Profile names come from the
// info_cache map; dirs that are not present on disk are skipped.
func parseChromeProfiles(data []byte, base, bin string) []browser {
	var ls struct {
		Profile struct {
			InfoCache map[string]struct {
				Name string `json:"name"`
			} `json:"info_cache"`
		} `json:"profile"`
	}
	if json.Unmarshal(data, &ls) != nil || ls.Profile.InfoCache == nil {
		return nil
	}
	dirs := make([]string, 0, len(ls.Profile.InfoCache))
	for dir := range ls.Profile.InfoCache {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	var list []browser
	for _, dir := range dirs {
		if !dirExists(base, dir) {
			continue
		}
		list = append(list, profileBrowser("Chrome", "chrome-", "google-chrome",
			dir, ls.Profile.InfoCache[dir].Name, []string{bin, "--profile-directory=" + dir}))
	}
	return list
}
