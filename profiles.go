package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func firefoxBase() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Firefox")
	case "windows":
		if app := os.Getenv("APPDATA"); app != "" {
			return filepath.Join(app, "Mozilla", "Firefox")
		}
	}
	return filepath.Join(home, ".mozilla", "firefox")
}

func firefoxBin() string {
	switch runtime.GOOS {
	case "darwin":
		return "/Applications/Firefox.app/Contents/MacOS/firefox"
	default:
		return "firefox"
	}
}

func firefoxProfiles() []browser {
	base := firefoxBase()
	fx := firefoxBin()
	var list []browser
	if data, err := os.ReadFile(filepath.Join(base, "profiles.ini")); err == nil {
		list = parseFirefoxProfiles(data, base, fx)
	}
	seen := map[string]bool{}
	for _, b := range list {
		seen[b.argv[len(b.argv)-1]] = true
	}
	return append(list, firefoxDirProfiles(base, fx, seen)...)
}

// firefoxDirProfiles finds real profile directories on disk. Modern Firefox
// stores profile-grouped profiles (e.g. "Profile 1") outside profiles.ini, so
// directory scanning is the reliable cross-version source: any dir containing
// prefs.js is a launchable profile. The leading "<hash>." prefix is stripped
// from the displayed name; seen skips paths already produced from profiles.ini.
func firefoxDirProfiles(base, fx string, seen map[string]bool) []browser {
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var list []browser
	for _, de := range entries {
		if !de.IsDir() {
			continue
		}
		name := de.Name()
		if strings.Contains(name, "-backup") {
			continue
		}
		p := filepath.Join(base, name)
		if seen[p] {
			continue
		}
		if _, err := os.Stat(filepath.Join(p, "prefs.js")); err != nil {
			continue
		}
		display := name
		if i := strings.IndexByte(display, '.'); i == 8 {
			display = display[i+1:]
		}
		list = append(list, browser{
			id:   "firefox-" + sanitizeID(display),
			name: "Firefox - " + display,
			argv: []string{fx, "-profile", p},
			icon: iconResource("firefox"),
		})
	}
	return list
}

// parseFirefoxProfiles turns a profiles.ini into one browser per Profile
// section, each launched with -profile <dir>. Sections that are not profiles
// (e.g. the Install... marker) are skipped.
func parseFirefoxProfiles(data []byte, base, fx string) []browser {
	var list []browser
	for _, sec := range parseINI(string(data)) {
		if !strings.HasPrefix(sec.name, "Profile") {
			continue
		}
		p := sec.kv["Path"]
		if p == "" {
			continue
		}
		if sec.kv["IsRelative"] == "1" {
			p = filepath.Join(base, p)
		}
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			continue
		}
		name := sec.kv["Name"]
		if name == "" {
			name = filepath.Base(p)
		}
		list = append(list, browser{
			id:   "firefox-" + sanitizeID(name),
			name: "Firefox - " + name,
			argv: []string{fx, "-profile", p},
			icon: iconResource("firefox"),
		})
	}
	return list
}

type iniSection struct {
	name string
	kv   map[string]string
}

// parseINI parses the simple section/[key=value] INI format used by
// profiles.ini; it is deliberately not a general INI parser.
func parseINI(s string) []iniSection {
	var out []iniSection
	var cur *iniSection
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		if line[0] == '[' && line[len(line)-1] == ']' {
			out = append(out, iniSection{name: strings.TrimSpace(line[1 : len(line)-1]), kv: map[string]string{}})
			cur = &out[len(out)-1]
			continue
		}
		if cur == nil {
			continue
		}
		if i := strings.Index(line, "="); i > 0 {
			cur.kv[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+1:])
		}
	}
	return out
}

func sanitizeID(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}
