package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
)

// iconRoots are the base directories searched for icon themes; a package var
// so tests can point it at a fixture tree.
var iconRoots = func() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"/usr/share",
		"/usr/local/share",
		"/var/lib/flatpak/exports/share",
		filepath.Join(home, ".local", "share"),
	}
}()

func resourceFromFile(path string) fyne.Resource {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource(filepath.Base(path), data)
}

// iconResource resolves a .desktop Icon value (a theme name or a path) to an
// image resource. Theme names are looked up in hicolor and pixmaps, png first
// then svg, at the common sizes.
func iconResource(name string) fyne.Resource {
	if name == "" {
		return nil
	}
	if strings.HasPrefix(name, "/") {
		return resourceFromFile(name)
	}
	base := strings.TrimSuffix(name, filepath.Ext(name))
	for _, root := range iconRoots {
		for _, size := range []string{"256", "128", "64", "48", "32"} {
			for _, ext := range []string{"png", "svg"} {
				p := filepath.Join(root, "hicolor", size+"x"+size, "apps", base+"."+ext)
				if r := resourceFromFile(p); r != nil {
					return r
				}
			}
		}
		for _, ext := range []string{"png", "svg"} {
			p := filepath.Join(root, "pixmaps", base+"."+ext)
			if r := resourceFromFile(p); r != nil {
				return r
			}
		}
	}
	return nil
}

// appIconResource returns the largest PNG embedded in an .icns file, used for
// macOS app bundle logos.
func appIconResource(path string) fyne.Resource {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 8 || string(data[:4]) != "icns" {
		return nil
	}
	best := []byte(nil)
	bestW := -1
	i := 8
	for i+8 <= len(data) {
		n := int(binary.BigEndian.Uint32(data[i+4 : i+8]))
		if n < 8 || i+n > len(data) {
			break
		}
		p := data[i+8 : i+n]
		if len(p) > 8 && p[0] == 0x89 && p[1] == 'P' && p[2] == 'N' && p[3] == 'G' {
			w := 0
			if len(p) > 20 && string(p[12:16]) == "IHDR" {
				w = int(binary.BigEndian.Uint32(p[16:20]))
			}
			if bestW < 0 || (w > 0 && (bestW < 0 || abs(w-128) < abs(bestW-128))) {
				best, bestW = p, w
			}
		}
		i += n
	}
	if best == nil {
		return nil
	}
	return fyne.NewStaticResource(filepath.Base(path), best)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
