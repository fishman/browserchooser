package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/BurntSushi/toml"
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
	got := parseChromeProfiles(data, base, "google-chrome", "Chrome", "chrome-", nil)

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

func TestParseChromiumProfiles(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "Default"), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"profile":{"info_cache":{"Default":{"name":"Person 1"}}}}`)
	got := parseChromeProfiles(data, base, "chromium", "Chromium", "chromium-", nil)
	if len(got) != 1 {
		t.Fatalf("want 1 profile, got %d", len(got))
	}
	if got[0].name != "Chromium - Person 1" || got[0].id != "chromium-default" {
		t.Errorf("got %q / %q", got[0].name, got[0].id)
	}
	if got[0].argv[0] != "chromium" || got[0].argv[1] != "--profile-directory=Default" {
		t.Errorf("argv %v", got[0].argv)
	}
}

func TestParseChromeProfilesEmpty(t *testing.T) {
	if got := parseChromeProfiles([]byte(`{"profile":{}}`), t.TempDir(), "x", "Chrome", "chrome-", nil); len(got) != 0 {
		t.Fatalf("want 0, got %d", len(got))
	}
}

func TestChromeBrowserConfigParsing(t *testing.T) {
	data := `
[[chrome.browsers]]
name = "Brave"
data_dir = "BraveSoftware/Brave-Browser"
binary = "brave-browser"
mac_binary = "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"
`
	var s settings
	if err := toml.Unmarshal([]byte(data), &s); err != nil {
		t.Fatal(err)
	}
	if len(s.Chrome.Browsers) != 1 {
		t.Fatalf("want 1 browser, got %d", len(s.Chrome.Browsers))
	}
	b := s.Chrome.Browsers[0]
	if b.Name != "Brave" || b.DataDir != "BraveSoftware/Brave-Browser" || b.Binary != "brave-browser" || b.MacBin != "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser" {
		t.Errorf("got %+v", b)
	}
}

func TestChromeFamilyFromConfig(t *testing.T) {
	f := chromeFamilyFromConfig(chromeBrowserConfig{
		Name: "Brave", DataDir: "BraveSoftware/Brave-Browser",
		Binary: "brave-browser", MacBin: "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
	})
	if f.prefix != "brave-" || f.icon != "brave" {
		t.Errorf("prefix/icon %q / %q", f.prefix, f.icon)
	}
	if f.linuxDir != "BraveSoftware/Brave-Browser" || f.linuxBins[0] != "brave-browser" {
		t.Errorf("linux dir/bin %q / %v", f.linuxDir, f.linuxBins)
	}
	if f.macBin == "" {
		t.Error("mac_binary should be kept")
	}
}

func TestBinaryCandidates(t *testing.T) {
	got := binaryCandidates("Brave", "BraveSoftware/Brave-Browser")
	want := []string{"brave", "brave-browser", "brave-stable"}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("candidate %d = %q, want %q (all: %v)", i, got[i], w, got)
		}
	}
	if len(got) < 3 {
		t.Fatalf("want at least 3 candidates, got %v", got)
	}
}

func TestDataDirCandidates(t *testing.T) {
	got := dataDirCandidates("Brave", "brave-browser")
	want := []string{"brave", "brave-browser", "brave-stable"}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("candidate %d = %q, want %q (all: %v)", i, got[i], w, got)
		}
	}
}

func TestDetectDataDir(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("data-dir detection is Linux-only")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".config")
	if _, err := os.Stat(filepath.Join(cfg, "chromium", "Local State")); err != nil {
		t.Skip("no chromium Local State to probe against")
	}
	if got := detectDataDir("Chromium", "chromium"); got != "chromium" {
		t.Errorf("detectDataDir(Chromium) = %q, want chromium", got)
	}
}
