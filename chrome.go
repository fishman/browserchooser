package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"

	"fyne.io/fyne/v2"
)

// chromeFamily describes one Chromium-family browser so profile detection is
// data, not one implementation per browser. Each family names its data dir and
// binary per platform; adding a browser is a new table row.
type chromeFamily struct {
	brand, prefix, icon string
	linuxDir            string
	linuxBins           []string
	macDir, macApp      string
	macBin              string
	winDir, winExe      string
}

// chromeFamilies returns the built-in Chromium-family browsers plus any added
// in config.toml as [[chrome.browsers]], so a fork needs a config entry, not a
// code change.
func chromeFamilies() []chromeFamily {
	fams := []chromeFamily{
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
	for _, c := range loadSettings().Chrome.Browsers {
		if c.Name == "" {
			continue
		}
		f := chromeFamilyFromConfig(c)
		if f.linuxDir == "" {
			f.linuxDir = detectDataDir(c.Name, familyBin(f))
		}
		if f.linuxDir == "" {
			continue
		}
		f.macDir, f.winDir = f.linuxDir, f.linuxDir
		fams = append(fams, f)
	}
	return fams
}

// chromeFamilyFromConfig maps a config entry to a family: the id prefix and
// icon derive from the name, and the single data_dir applies on every platform.
// An explicit binary is honored; otherwise launcher names are derived.
func chromeFamilyFromConfig(c chromeBrowserConfig) chromeFamily {
	id := sanitizeID(c.Name)
	bins := binaryCandidates(c.Name, c.DataDir)
	if c.Binary != "" {
		bins = []string{c.Binary}
	}
	winExe := c.Binary
	if winExe == "" {
		winExe = bins[0]
	}
	return chromeFamily{
		brand: c.Name, prefix: id + "-", icon: id,
		linuxDir: c.DataDir, macDir: c.DataDir, winDir: c.DataDir,
		linuxBins: bins,
		macApp:    c.Name,
		macBin:    c.MacBin,
		winExe:    winExe,
	}
}

// binaryCandidates lists likely launcher commands for a Chromium fork, derived
// from its display name and data-dir basename, so the binary need not be
// configured. Linux finds one via LookPath; the list is never empty when the
// name is set.
func binaryCandidates(name, dataDir string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	base := sanitizeID(name)
	dir := sanitizeID(filepath.Base(dataDir))
	add(base)
	add(base + "-browser")
	add(base + "-stable")
	add(dir)
	add(dir + "-stable")
	return out
}

// dataDirCandidates lists likely ~/.config data-dir names for a Chromium fork,
// derived from its display name and resolved binary, so data_dir can be
// optional on Linux. A candidate is used only if it exists on disk.
func dataDirCandidates(name, bin string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	base := sanitizeID(name)
	binBase := sanitizeID(filepath.Base(bin))
	add(base)
	add(base + "-browser")
	add(base + "-stable")
	add(binBase)
	add(binBase + "-stable")
	return out
}

// detectDataDir finds an existing Linux data dir by probing a few derived
// candidates for a Local State file: a handful of stat calls, no directory
// scan, so a large ~/.config is never walked. Returns "" when none matches, so
// nested layouts (e.g. Brave) keep an explicit data_dir.
func detectDataDir(name, bin string) string {
	if runtime.GOOS != "linux" {
		return ""
	}
	home, _ := os.UserHomeDir()
	cfg := filepath.Join(home, ".config")
	for _, cand := range dataDirCandidates(name, bin) {
		if exists(filepath.Join(cfg, cand, "Local State")) {
			return cand
		}
	}
	return ""
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
		if f.macBin != "" {
			return f.macBin
		}
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
	for _, f := range chromeFamilies() {
		data, err := os.ReadFile(filepath.Join(familyBase(f), "Local State"))
		if err != nil {
			continue
		}
		list = append(list, parseChromeProfiles(data, familyBase(f), familyBin(f), f.brand, f.prefix, iconFor(f.icon, f.macApp))...)
	}
	return list
}

// parseChromeProfiles turns a Chromium-family "Local State" JSON into one
// browser per profile, launched with --profile-directory=<dir>. Profile names
// come from the info_cache map; dirs that are not present on disk are skipped.
func parseChromeProfiles(data []byte, base, bin, brand, prefix string, icon fyne.Resource) []browser {
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
