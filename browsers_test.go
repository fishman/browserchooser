package main

import (
	"testing"
	"time"
)

func TestFuzzySubsequence(t *testing.T) {
	cases := []struct {
		name, query string
		ok          bool
	}{
		{"Firefox", "ff", true},
		{"Firefox", "fx", true},
		{"Google Chrome", "gc", true},
		{"Google Chrome", "ch", true},
		{"Firefox", "zz", false},
		{"Firefox", "firefox nightly", false},
	}
	for _, c := range cases {
		if _, ok := fuzzyScore(c.name, c.query); ok != c.ok {
			t.Errorf("fuzzy(%q, %q) ok=%v want %v", c.name, c.query, ok, c.ok)
		}
	}
}

func TestFuzzyOrdering(t *testing.T) {
	consecutive, ok1 := fuzzyScore("ab", "ab")
	scattered, ok2 := fuzzyScore("xaby", "ab")
	if !ok1 || !ok2 {
		t.Fatal("both should match")
	}
	if consecutive <= scattered {
		t.Errorf("consecutive match (%v) should outrank scattered (%v)", consecutive, scattered)
	}
}

func TestFrecency(t *testing.T) {
	newer := time.Now().Add(-time.Minute).Unix()
	older := time.Now().Add(-30 * 24 * time.Hour).Unix()
	if frecencyScore(10, newer) <= frecencyScore(10, older) {
		t.Error("newer usage should outrank older at equal count")
	}
	if frecencyScore(9, newer) <= frecencyScore(2, newer) {
		t.Error("higher count should outrank lower at equal recency")
	}
}

func TestHostOf(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://github.com/foo", "github.com"},
		{"github.com", "github.com"},
		{"http://example.com:8080/x", "example.com"},
		{"https://www.github.com", "github.com"},
		{"www.example.com", "example.com"},
		{"", ""},
		{"not a url", ""},
	}
	for _, c := range cases {
		if got := hostOf(c.in); got != c.want {
			t.Errorf("hostOf(%q) = %q want %q", c.in, got, c.want)
		}
	}
}

func TestDominantHostID(t *testing.T) {
	if got := dominantHostID(map[string]useStat{"firefox": {Count: 2}}); got != "" {
		t.Errorf("2 opens below min threshold, want none, got %q", got)
	}
	if got := dominantHostID(map[string]useStat{"firefox": {Count: 3}}); got != "firefox" {
		t.Errorf("3 opens one browser, want firefox, got %q", got)
	}
	if got := dominantHostID(map[string]useStat{"firefox": {Count: 2}, "chrome": {Count: 1}}); got != "firefox" {
		t.Errorf("2/3 majority, want firefox, got %q", got)
	}
	if got := dominantHostID(map[string]useStat{"firefox": {Count: 2}, "chrome": {Count: 2}}); got != "" {
		t.Errorf("split 2/2 not dominant, want none, got %q", got)
	}
	if got := dominantHostID(nil); got != "" {
		t.Errorf("empty counts, want none, got %q", got)
	}
}

func TestExpandURL(t *testing.T) {
	cases := []struct {
		argv []string
		url  string
		want []string
	}{
		{[]string{"firefox", "%u"}, "https://x", []string{"firefox", "https://x"}},
		{[]string{"xdg-open"}, "https://x", []string{"xdg-open", "https://x"}},
		{[]string{"firefox", "%u"}, "", []string{"firefox"}},
		{[]string{"flatpak", "run", "org.x", "--", "%U"}, "https://x", []string{"flatpak", "run", "org.x", "--", "https://x"}},
	}
	for _, c := range cases {
		if got := expandURL(c.argv, c.url); !eq(got, c.want) {
			t.Errorf("expandURL(%v, %q) = %v want %v", c.argv, c.url, got, c.want)
		}
	}
}

func TestSplitExecQuotes(t *testing.T) {
	got := splitExec(`/opt/My Browser/bin/browser --foo "bar baz"`)
	want := []string{"/opt/My", "Browser/bin/browser", "--foo", "bar baz"}
	if !eq(got, want) {
		t.Errorf("splitExec = %v want %v", got, want)
	}
}

func TestExecArgv(t *testing.T) {
	got := execArgv(`/usr/lib/firefox/firefox -new-window %u`)
	want := []string{"/usr/lib/firefox/firefox", "-new-window", "%u"}
	if !eq(got, want) {
		t.Errorf("execArgv = %v want %v", got, want)
	}
	got = execArgv(`env MOZ_APP_LAUNCHER=firefox /usr/bin/firefox %f %i %c`)
	want = []string{"env", "MOZ_APP_LAUNCHER=firefox", "/usr/bin/firefox", "%f"}
	if !eq(got, want) {
		t.Errorf("execArgv field-strip = %v want %v", got, want)
	}
}

func TestRankRows(t *testing.T) {
	browsers := []browser{
		{id: "a", name: "Alpha"},
		{id: "b", name: "Bravo"},
		{id: "c", name: "Charlie"},
		{id: "d", name: "Delta"},
		{id: "e", name: "Echo"},
	}
	stats := map[string]useStat{
		"c": {Count: 20},
		"b": {Count: 5},
	}
	rows := rankRows(browsers, stats, "")
	if len(rows) != 4 {
		t.Fatalf("want 4 browsers, got %d", len(rows))
	}
	if rows[0].label != "Charlie" {
		t.Errorf("frecency leader should rank first, got %q", rows[0].label)
	}
	filtered := rankRows(browsers, stats, "al")
	if len(filtered) != 2 {
		t.Fatalf("'al' should match Alpha and Delta, got %d rows", len(filtered))
	}
	if filtered[0].label != "Alpha" {
		t.Errorf("fuzzy match should rank Alpha first, got %q", filtered[0].label)
	}
}

func TestEvalRule(t *testing.T) {
	if !evalRule(rule{Expr: `link contains "github.com"`}, "https://github.com/foo") {
		t.Error("contains rule should match")
	}
	if evalRule(rule{Expr: `link contains "github.com"`}, "https://example.com") {
		t.Error("contains rule should not match")
	}
	if !evalRule(rule{Expr: `link startsWith "https://"`}, "https://x") {
		t.Error("startsWith rule should match")
	}
	if evalRule(rule{Expr: `this is not valid expr @@@`}, "https://x") {
		t.Error("invalid expression should not match")
	}
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
