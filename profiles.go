package main

import (
	"os"
	"os/exec"
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
	names := firefoxProfileNames(base) // empty when no group DBs are readable
	if len(names) > 0 {
		// Modern profile groups exist: list every real profile dir and use the
		// real name from the DB. This is the single source, not merged with
		// profiles.ini.
		list := firefoxDirProfiles(base, fx, map[string]bool{})
		for i := range list {
			if n, ok := names[filepath.Base(list[i].argv[len(list[i].argv)-1])]; ok && n != "" {
				list[i].name = "Firefox - " + n
			}
		}
		return list
	}
	// No modern groups: fall back to the classic profiles.ini.
	var list []browser
	if data, err := os.ReadFile(filepath.Join(base, "profiles.ini")); err == nil {
		list = parseFirefoxProfiles(data, base, fx)
	}
	return list
}

// firefoxProfileNames reads the real display name for each profile directory
// from the modern Firefox profile-group sqlite DBs. Modern profiles store their
// user-facing name there, not in the directory name (a dir may be
// "<hash>.Profile 4" while its name is "Movies"). It shells out to the sqlite3
// CLI to avoid adding a sqlite binding; -readonly never writes Firefox's files.
// If the CLI is missing or a DB is locked, that entry is simply absent and the
// caller falls back to the directory-derived name.
func firefoxProfileNames(base string) map[string]string {
	names := map[string]string{}
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return names
	}
	groups, err := filepath.Glob(filepath.Join(base, "Profile Groups", "*.sqlite"))
	if err != nil {
		return names
	}
	for _, g := range groups {
		// The sqlite3 CLI escapes control bytes in text output, so split on the
		// printable column separator | instead: a profile dir name never has one.
		out, err := exec.Command("sqlite3", "-readonly", "-separator", "|", g,
			"SELECT path, name FROM Profiles;").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			if i := strings.IndexByte(line, '|'); i > 0 {
				if n := line[i+1:]; n != "" {
					names[line[:i]] = n
				}
			}
		}
	}
	return names
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
		list = append(list, profileBrowser("Firefox", "firefox-", "firefox", display, display, []string{fx, "-profile", p}))
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
		list = append(list, profileBrowser("Firefox", "firefox-", "firefox", name, name, []string{fx, "-profile", p}))
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

// profileBrowser assembles a browser entry for one profile dir, shared across
// browser families. The name falls back to the dir when empty; id is stable
// via the per-family prefix.
func profileBrowser(brand, idPrefix, icon, dir, name string, argv []string) browser {
	if name == "" {
		name = dir
	}
	return browser{
		id:   idPrefix + sanitizeID(dir),
		name: brand + " - " + name,
		argv: argv,
		icon: iconResource(icon),
	}
}

func dirExists(base, dir string) bool {
	st, err := os.Stat(filepath.Join(base, dir))
	return err == nil && st.IsDir()
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
