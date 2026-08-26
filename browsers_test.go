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
	rows := rankRows(browsers, stats, "", "https://x", "copy")
	if len(rows) != 5 {
		t.Fatalf("want 5 rows (4 browsers + copy), got %d", len(rows))
	}
	if rows[0].label != "Charlie" {
		t.Errorf("frecency leader should rank first, got %q", rows[0].label)
	}
	last := rows[len(rows)-1]
	if !last.copy || last.label != "copy" {
		t.Errorf("copy row should be last, got %+v", last)
	}
	filtered := rankRows(browsers, stats, "al", "https://x", "copy")
	if len(filtered) != 3 {
		t.Fatalf("'al' should match Alpha and Delta plus copy, got %d rows", len(filtered))
	}
	nocopy := rankRows(browsers, stats, "", "", "copy")
	if len(nocopy) != 4 {
		t.Fatalf("no url should skip copy row, got %d rows", len(nocopy))
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
