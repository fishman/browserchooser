package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseChromeProfiles(t *testing.T) {
	base := t.TempDir()
	mk := func(name string) {
		if err := os.MkdirAll(filepath.Join(base, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mk("Default")
	mk("Profile 1")
	data := []byte(`{"profile":{"info_cache":{
		"Default":{"name":"Person 1"},
		"Profile 1":{"name":"Work"},
		"Profile 2":{"name":"Missing"}
	}}}`)
	got := parseChromeProfiles(data, base, "google-chrome")

	if len(got) != 2 {
		t.Fatalf("want 2 profiles (missing dir skipped), got %d", len(got))
	}
	if got[0].name != "Chrome - Person 1" || got[0].id != "chrome-default" {
		t.Errorf("got %q / %q", got[0].name, got[0].id)
	}
	want := []string{"google-chrome", "--profile-directory=Profile 1"}
	if got[1].name != "Chrome - Work" || len(got[1].argv) != 2 || got[1].argv[0] != want[0] || got[1].argv[1] != want[1] {
		t.Errorf("Profile 1: got %q argv %v, want argv %v", got[1].name, got[1].argv, want)
	}
}

func TestParseChromeProfilesEmpty(t *testing.T) {
	if got := parseChromeProfiles([]byte(`{"profile":{}}`), t.TempDir(), "x"); len(got) != 0 {
		t.Fatalf("want 0, got %d", len(got))
	}
}
