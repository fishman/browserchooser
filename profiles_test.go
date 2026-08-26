package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFirefoxDirProfiles(t *testing.T) {
	base := t.TempDir()
	mk := func(name string) {
		if err := os.MkdirAll(filepath.Join(base, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mk("5X601kJG.Profile 4")
	mk("2DUG7EoJ.Profile 1")
	mk("a1b2c3d4.old-backup")
	for _, name := range []string{"5X601kJG.Profile 4", "2DUG7EoJ.Profile 1", "a1b2c3d4.old-backup"} {
		if err := os.WriteFile(filepath.Join(base, name, "prefs.js"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := firefoxDirProfiles(base, "firefox", map[string]bool{})

	if len(got) != 2 {
		t.Fatalf("want 2 profiles (backup skipped), got %d", len(got))
	}
	byName := map[string]browser{}
	for _, b := range got {
		byName[b.name] = b
	}
	got4, ok := byName["Firefox - Profile 4"]
	if !ok {
		t.Fatalf("missing Profile 4 in %v", got)
	}
	if got4.id != "firefox-profile-4" {
		t.Errorf("id = %q", got4.id)
	}
	want := []string{"firefox", "-profile", filepath.Join(base, "5X601kJG.Profile 4")}
	if len(got4.argv) != len(want) || got4.argv[0] != want[0] || got4.argv[2] != want[2] {
		t.Errorf("argv = %v, want %v", got4.argv, want)
	}
}

func TestFirefoxProfileNames(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 CLI not available")
	}
	base := t.TempDir()
	groups := filepath.Join(base, "Profile Groups")
	if err := os.MkdirAll(groups, 0o755); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(groups, "abc123.sqlite")
	sql := "CREATE TABLE Profiles(path TEXT, name TEXT);" +
		"INSERT INTO Profiles VALUES('5X601kJG.Profile 4','Movies');" +
		"INSERT INTO Profiles VALUES('jBIu9IkI.Profile 2','Home');"
	if out, err := exec.Command("sqlite3", db, sql).CombinedOutput(); err != nil {
		t.Fatalf("create db: %v: %s", err, out)
	}
	got := firefoxProfileNames(base)
	if got["5X601kJG.Profile 4"] != "Movies" {
		t.Errorf("missing Movies: %v", got)
	}
	if got["jBIu9IkI.Profile 2"] != "Home" {
		t.Errorf("missing Home: %v", got)
	}
}

func TestParseFirefoxProfiles(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "abc.default-release"), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte("[Install4F96D1932A9F858E]\nDefault=profiles.ini\n\n" +
		"[Profile0]\nName=default-release\nIsRelative=1\nPath=abc.default-release\n" +
		"[Profile1]\nName=Dev Box\nIsRelative=0\nPath=/nonexistent/nowhere\n")
	got := parseFirefoxProfiles(data, base, "firefox")

	if len(got) != 1 {
		t.Fatalf("want 1 profile, got %d", len(got))
	}
	if got[0].name != "Firefox - default-release" {
		t.Errorf("name = %q", got[0].name)
	}
	if got[0].id != "firefox-default-release" {
		t.Errorf("id = %q", got[0].id)
	}
	want := []string{"firefox", "-profile", filepath.Join(base, "abc.default-release")}
	if len(got[0].argv) != len(want) || got[0].argv[0] != want[0] || got[0].argv[1] != want[1] || got[0].argv[2] != want[2] {
		t.Errorf("argv = %v, want %v", got[0].argv, want)
	}
}
