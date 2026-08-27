package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDarwinApps(t *testing.T) {
	dir := t.TempDir()
	writeApp := func(name, bundleID, display string, schemes []string) {
		appDir := filepath.Join(dir, name+".app", "Contents")
		if err := os.MkdirAll(appDir, 0o755); err != nil {
			t.Fatal(err)
		}
		var b strings.Builder
		b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>`)
		fmt.Fprintf(&b, "<key>CFBundleIdentifier</key><string>%s</string>", bundleID)
		fmt.Fprintf(&b, "<key>CFBundleDisplayName</key><string>%s</string>", display)
		b.WriteString("<key>CFBundleURLTypes</key><array><dict><key>CFBundleURLSchemes</key><array>")
		for _, s := range schemes {
			fmt.Fprintf(&b, "<string>%s</string>", s)
		}
		b.WriteString("</array></dict></array></dict></plist>")
		if err := os.WriteFile(filepath.Join(appDir, "Info.plist"), []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeApp("Fake Browser", "com.example.FakeBrowser", "Fake Browser", []string{"https", "http"})
	writeApp("Ftp Tool", "com.example.FtpTool", "Ftp Tool", []string{"ftp"})

	got := darwinApps([]string{dir})
	if len(got) != 1 {
		t.Fatalf("want 1 browser (ftp-only excluded), got %d: %+v", len(got), got)
	}
	if got[0].name != "Fake Browser" || got[0].id != "fake-browser" {
		t.Errorf("got %q/%q, want Fake Browser/fake-browser", got[0].name, got[0].id)
	}
	want := []string{"open", "-b", "com.example.FakeBrowser"}
	if len(got[0].argv) != len(want) || got[0].argv[0] != want[0] || got[0].argv[1] != want[1] || got[0].argv[2] != want[2] {
		t.Errorf("argv = %v, want %v", got[0].argv, want)
	}
}

func TestHandlesURL(t *testing.T) {
	if handlesURL(&appInfo{URLTypes: []struct {
		Schemes []string `plist:"CFBundleURLSchemes"`
	}{{Schemes: []string{"ftp"}}}}) {
		t.Error("ftp-only bundle should not be a browser")
	}
	if !handlesURL(&appInfo{URLTypes: []struct {
		Schemes []string `plist:"CFBundleURLSchemes"`
	}{{Schemes: []string{"https"}}}}) {
		t.Error("https bundle should be a browser")
	}
}
