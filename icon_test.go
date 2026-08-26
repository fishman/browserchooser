package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/theme"
)

func TestNordTheme(t *testing.T) {
	dark := &nordTheme{variant: theme.VariantDark}
	light := &nordTheme{variant: theme.VariantLight}
	if c := dark.Color(theme.ColorNameBackground, theme.VariantDark); c != nordPolar1 {
		t.Errorf("dark background = %v, want %v", c, nordPolar1)
	}
	if c := light.Color(theme.ColorNameBackground, theme.VariantLight); c != nordSnow3 {
		t.Errorf("light background = %v, want %v", c, nordSnow3)
	}
	if c := dark.Color(theme.ColorNameForeground, theme.VariantDark); c != nordSnow1 {
		t.Errorf("dark foreground = %v, want %v", c, nordSnow1)
	}
	if c := light.Color(theme.ColorNameForeground, theme.VariantLight); c != nordPolar1 {
		t.Errorf("light foreground = %v, want %v", c, nordPolar1)
	}
}

func TestIconResourceName(t *testing.T) {
	old := iconRoots
	defer func() { iconRoots = old }()

	dir := t.TempDir()
	p := filepath.Join(dir, "icons", "hicolor", "scalable", "apps")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, "firefox.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	iconRoots = []string{dir}

	r := iconResource("firefox")
	if r == nil {
		t.Fatal("iconResource should resolve the themed name")
	}
	if r.Name() != "firefox.svg" {
		t.Errorf("icon name = %q, want firefox.svg", r.Name())
	}
	if iconResource("missing") != nil {
		t.Error("unknown icon name should resolve to nil")
	}
}

func TestAppIconResource(t *testing.T) {
	pngData := makeTestPNG(t)
	icns := makeTestICNS(pngData)
	path := filepath.Join(t.TempDir(), "AppIcon.icns")
	if err := os.WriteFile(path, icns, 0o644); err != nil {
		t.Fatal(err)
	}

	r := appIconResource(path)
	if r == nil {
		t.Fatal("appIconResource should extract the embedded png")
	}
	if !bytes.Equal(r.Content(), pngData) {
		t.Error("extracted png should match the embedded chunk")
	}
}

func makeTestPNG(t *testing.T) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func makeTestICNS(pngData []byte) []byte {
	total := 8 + 8 + len(pngData)
	out := make([]byte, total)
	copy(out[:4], "icns")
	binary.BigEndian.PutUint32(out[4:8], uint32(total))
	copy(out[8:12], "ic07")
	binary.BigEndian.PutUint32(out[12:16], uint32(8+len(pngData)))
	copy(out[16:], pngData)
	return out
}
