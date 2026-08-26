package main

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"

	"github.com/expr-lang/expr"
	"github.com/jeandeaual/go-locale"
	"github.com/rkoesters/xdg/desktop"
	"github.com/rkoesters/xdg/keyfile"
	"howett.net/plist"
)

type browser struct {
	id   string
	name string
	argv []string
	icon fyne.Resource
}

func (b *browser) open(url string) error {
	argv := expandURL(b.argv, url)
	if len(argv) == 0 {
		return errors.New("empty command")
	}
	return exec.Command(argv[0], argv[1:]...).Start()
}

type useStat struct {
	Count int
	Last  int64
}

type rule struct {
	Expr    string `json:"expr"`
	Browser string `json:"browser"`
}

type row struct {
	label string
	b     *browser
}

func detectBrowsers() []browser {
	switch runtime.GOOS {
	case "darwin":
		return detectDarwin()
	case "windows":
		return detectWindows()
	default:
		return detectLinux()
	}
}

func fallbackBrowser() browser {
	switch runtime.GOOS {
	case "windows":
		return browser{id: "default", name: "System default browser", argv: []string{"cmd", "/c", "start", `""`}}
	case "darwin":
		return browser{id: "default", name: "System default browser", argv: []string{"open"}}
	default:
		return browser{id: "default", name: "System default browser", argv: []string{"xdg-open"}}
	}
}

func detectLinux() []browser {
	home, _ := os.UserHomeDir()
	dirs := []string{
		"/usr/share/applications",
		"/usr/local/share/applications",
		"/var/lib/flatpak/exports/share/applications",
		filepath.Join(home, ".local", "share", "applications"),
		filepath.Join(home, ".local", "share", "flatpak", "exports", "share", "applications"),
	}
	loc := keyfile.DefaultLocale()
	if l, err := locale.GetLanguage(); err == nil && l != "" {
		if pl, err := keyfile.ParseLocale(l); err == nil {
			loc = pl
		}
	}
	seen := map[string]bool{}
	var list []browser
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, de := range entries {
			if de.IsDir() || !strings.HasSuffix(de.Name(), ".desktop") {
				continue
			}
			f, err := os.Open(filepath.Join(dir, de.Name()))
			if err != nil {
				continue
			}
			e, err := desktop.NewWithLocale(f, loc)
			f.Close()
			if err != nil || !isBrowserEntry(e) {
				continue
			}
			if e.Name == "" {
				e.Name = strings.TrimSuffix(de.Name(), ".desktop")
			}
			argv := execArgv(e.Exec)
			if len(argv) == 0 {
				continue
			}
			id := filepath.Base(argv[0])
			if seen[id] {
				continue
			}
			seen[id] = true
			list = append(list, browser{id: id, name: e.Name, argv: argv, icon: iconResource(e.Icon)})
		}
	}
	return list
}

func isBrowserEntry(e *desktop.Entry) bool {
	if e.Type != desktop.Application || e.Hidden || e.NoDisplay || e.Exec == "" {
		return false
	}
	for _, m := range e.MimeType {
		if m == "x-scheme-handler/http" || m == "x-scheme-handler/https" || m == "text/html" {
			return true
		}
	}
	for _, c := range e.Categories {
		if c == "WebBrowser" {
			return true
		}
	}
	return false
}

func detectDarwin() []browser {
	names := []string{
		"Safari", "Google Chrome", "Firefox", "Microsoft Edge",
		"Brave Browser", "Arc", "Opera", "Vivaldi", "Chromium",
	}
	var list []browser
	for _, name := range names {
		id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		path := appBundlePath(name)
		if path == "" {
			continue
		}
		b := browser{id: id, name: name, argv: []string{"open", "-a", name}}
		b.icon = appIconResource(filepath.Join(path, "Contents", "Resources", "AppIcon.icns"))
		if info, err := readInfoPlist(path); err == nil {
			if info.BundleID != "" {
				b.argv = []string{"open", "-b", info.BundleID}
			}
			if info.DisplayName != "" {
				b.name = info.DisplayName
			}
		}
		list = append(list, b)
	}
	return list
}

func appBundlePath(name string) string {
	home, _ := os.UserHomeDir()
	for _, base := range []string{"/Applications", filepath.Join(home, "Applications")} {
		p := filepath.Join(base, name+".app")
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return ""
}

type appInfo struct {
	BundleID    string `plist:"CFBundleIdentifier"`
	DisplayName string `plist:"CFBundleDisplayName"`
	Name        string `plist:"CFBundleName"`
}

func readInfoPlist(appPath string) (*appInfo, error) {
	data, err := os.ReadFile(filepath.Join(appPath, "Contents", "Info.plist"))
	if err != nil {
		return nil, err
	}
	var info appInfo
	if _, err := plist.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func detectWindows() []browser {
	env := func(k string) string { return os.Getenv(k) }
	pf := env("ProgramFiles")
	pf32 := env("ProgramFiles(x86)")
	la := env("LocalAppData")
	type cand struct{ name, id, path string }
	cands := []cand{
		{"Google Chrome", "google-chrome", la + `\Google\Chrome\Application\chrome.exe`},
		{"Google Chrome", "google-chrome", pf + `\Google\Chrome\Application\chrome.exe`},
		{"Microsoft Edge", "microsoft-edge", pf32 + `\Microsoft\Edge\Application\msedge.exe`},
		{"Firefox", "firefox", pf + `\Mozilla Firefox\firefox.exe`},
		{"Brave", "brave", la + `\BraveSoftware\Brave-Browser\Application\brave.exe`},
		{"Opera", "opera", pf + `\Opera\opera.exe`},
		{"Opera", "opera", la + `\Programs\Opera\opera.exe`},
		{"Vivaldi", "vivaldi", la + `\Vivaldi\Application\vivaldi.exe`},
		{"Chromium", "chromium", la + `\Chromium\Application\chrome.exe`},
	}
	seen := map[string]bool{}
	var list []browser
	for _, c := range cands {
		if c.path == "" || seen[c.id] {
			continue
		}
		if _, err := os.Stat(c.path); err != nil {
			continue
		}
		seen[c.id] = true
		list = append(list, browser{id: c.id, name: c.name, argv: []string{c.path}})
	}
	return list
}

// execArgv splits a .desktop Exec line, keeping URL field codes (%u/%U/%f/%F)
// and dropping every other field code the spec defines.
func execArgv(execLine string) []string {
	raw := splitExec(execLine)
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		if len(t) >= 2 && t[0] == '%' && !isURLField(t) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// splitExec splits on unquoted spaces, honoring the desktop-entry quoting rules.
func splitExec(s string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQuote = !inQuote
		case c == ' ' && !inQuote:
			flush()
		case c == '\\' && i+1 < len(s):
			i++
			cur.WriteByte(s[i])
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return out
}

// expandURL substitutes url for URL field codes, or appends it when absent.
func expandURL(argv []string, url string) []string {
	out := make([]string, 0, len(argv)+1)
	used := false
	for _, a := range argv {
		if isURLField(a) {
			used = true
			if url != "" {
				out = append(out, url)
			}
			continue
		}
		out = append(out, a)
	}
	if url != "" && !used {
		out = append(out, url)
	}
	return out
}

func isURLField(t string) bool {
	switch t {
	case "%u", "%U", "%f", "%F":
		return true
	}
	return false
}

// fuzzyScore ranks name against query as a subsequence match.
func fuzzyScore(name, query string) (float64, bool) {
	q := strings.ToLower(query)
	if q == "" {
		return 0, true
	}
	n := strings.ToLower(name)
	qi := 0
	var score float64
	last := -2
	for i := 0; i < len(n) && qi < len(q); i++ {
		if n[i] != q[qi] {
			continue
		}
		switch {
		case i == last+1:
			score += 10
		case qi == 0 && i == 0:
			score += 5
		case qi == 0:
			score += 2
		}
		if qi > 0 && isWordBoundary(n, i) {
			score += 3
		}
		last = i
		qi++
	}
	if qi < len(q) {
		return 0, false
	}
	return score, true
}

func isWordBoundary(s string, i int) bool {
	if i == 0 {
		return true
	}
	c := s[i-1]
	return c == ' ' || c == '-' || c == '_' || c == '.'
}

// frecencyScore mixes frequency and recency: count+1 so an unseen browser
// still ranks above zero, decayed by a 30-day half life.
func frecencyScore(count int, last int64) float64 {
	age := time.Since(time.Unix(last, 0))
	if age < 0 {
		age = 0
	}
	return float64(count+1) * math.Pow(0.5, float64(age)/float64(frecencyHalfLife))
}

// rankRows returns at most 4 matching browsers ordered by fuzzy then frecency.
// The copy-link row is not part of the match set; the UI pins it at key 5.
func rankRows(browsers []browser, stats map[string]useStat, query string) []row {
	type scored struct {
		b browser
		s float64
	}
	sc := make([]scored, 0, len(browsers))
	for _, b := range browsers {
		f, ok := fuzzyScore(b.name, query)
		if !ok {
			continue
		}
		st := stats[b.id]
		sc = append(sc, scored{b: b, s: f + frecencyScore(st.Count, st.Last)})
	}
	sort.SliceStable(sc, func(i, j int) bool {
		if sc[i].s != sc[j].s {
			return sc[i].s > sc[j].s
		}
		return sc[i].b.name < sc[j].b.name
	})
	rows := make([]row, 0, maxRows)
	for i, x := range sc {
		if i >= maxRows {
			break
		}
		b := x.b
		rows = append(rows, row{label: b.name, b: &b})
	}
	return rows
}

func rulesPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "browserchooser", "rules.json")
}

func loadRules() []rule {
	p := rulesPath()
	if p == "" {
		return nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var rules []rule
	if json.Unmarshal(data, &rules) != nil {
		return nil
	}
	return rules
}

// matchRule returns the first browser whose rule expression matches the url.
func matchRule(url string) (*browser, bool) {
	rules := loadRules()
	if len(rules) == 0 {
		return nil, false
	}
	browsers := detectBrowsers()
	byID := map[string]*browser{}
	for i := range browsers {
		byID[browsers[i].id] = &browsers[i]
	}
	for _, r := range rules {
		if evalRule(r, url) {
			if b, found := byID[r.Browser]; found {
				return b, true
			}
		}
	}
	return nil, false
}

func evalRule(r rule, url string) bool {
	if r.Expr == "" {
		return false
	}
	env := map[string]any{"link": url}
	program, err := expr.Compile(r.Expr, expr.Env(env), expr.AsBool())
	if err != nil {
		return false
	}
	out, err := expr.Run(program, env)
	if err != nil {
		return false
	}
	ok, _ := out.(bool)
	return ok
}
