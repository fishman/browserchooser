package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
)

// chromeFamily describes one Chromium-family browser so profile detection is
// data, not one implementation per browser. Each family names its data dir and
// binary per platform; adding a browser is a new table row.
type chromeFamily struct {
	brand, prefix, icon string
	linuxDir            string
	linuxBins           []string
	macDir, macApp      string
	winDir, winExe      string
}

var chromeFamilies = []chromeFamily{
	{
		brand: "Chrome", prefix: "chrome-", icon: "google-chrome",
		linuxDir:  "google-chrome",
		linuxBins: []string{"google-chrome", "google-chrome-stable", "google-chrome-beta", "google-chrome-unstable"},
		macDir:    "Google/Chrome", macApp: "Google Chrome",
		winDir: `Google\Chrome`, winExe: "chrome.exe",
	},
	{
		brand: "Chromium", prefix: "chromium-", icon: "chromium",
		linuxDir:  "chromium",
		linuxBins: []string{"chromium", "chromium-browser"},
		macDir:    "Chromium", macApp: "Chromium",
		winDir: "Chromium", winExe: "chrome.exe",
	},
}

// familyBase is the "User Data" dir holding Local State and one subdir per
// profile (Default, Profile 1, ...), per the platform's convention.
func familyBase(f chromeFamily) string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", f.macDir)
	case "windows":
		if app := os.Getenv("LOCALAPPDATA"); app != "" {
			return filepath.Join(app, f.winDir, "User Data")
		}
	default:
		return filepath.Join(home, ".config", f.linuxDir)
	}
	return filepath.Join(home, ".config", f.linuxDir)
}

// familyBin resolves the family's launcher, checking the platform's usual
// locations and falling back to a bare command name.
func familyBin(f chromeFamily) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join("/Applications", f.macApp+".app", "Contents", "MacOS", f.macApp)
	case "windows":
		if pf := os.Getenv("ProgramFiles"); pf != "" {
			if p := filepath.Join(pf, f.winDir, "Application", f.winExe); exists(p) {
				return p
			}
		}
		return f.winExe
	default:
		for _, name := range f.linuxBins {
			if p, err := exec.LookPath(name); err == nil {
				return p
			}
		}
		return f.linuxBins[0]
	}
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// chromeProfiles returns one browser per profile for every Chromium-family
// install, so profiles that are not the default are listable and routable.
func chromeProfiles() []browser {
	var list []browser
	for _, f := range chromeFamilies {
		data, err := os.ReadFile(filepath.Join(familyBase(f), "Local State"))
		if err != nil {
			continue
		}
		list = append(list, parseChromeProfiles(data, familyBase(f), familyBin(f), f.brand, f.prefix, f.icon)...)
	}
	return list
}

// parseChromeProfiles turns a Chromium-family "Local State" JSON into one
// browser per profile, launched with --profile-directory=<dir>. Profile names
// come from the info_cache map; dirs that are not present on disk are skipped.
func parseChromeProfiles(data []byte, base, bin, brand, prefix, icon string) []browser {
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
		list = append(list, profileBrowser(brand, prefix, icon,
			dir, ls.Profile.InfoCache[dir].Name, []string{bin, "--profile-directory=" + dir}))
	}
	return list
}
