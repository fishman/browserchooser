package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
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

// iconCache memoizes resolved theme icons so a full-tree fallback walk happens
// at most once per name, not on every startup render.
var iconCache = map[string]fyne.Resource{}

// iconResource resolves a .desktop Icon value (a theme name or a path) to an
// image resource. Theme names hit the common pixmaps/hicolor spots directly;
// only a miss falls back to walking the tree. Prefers png over svg over xpm.
func iconResource(name string) fyne.Resource {
	if name == "" {
		return nil
	}
	if strings.HasPrefix(name, "/") {
		return resourceFromFile(name)
	}
	if r, ok := iconCache[name]; ok {
		return r
	}
	base := strings.TrimSuffix(name, filepath.Ext(name))
	r := lookupIcon(base)
	iconCache[name] = r
	return r
}

// iconFor resolves a browser's icon per platform from its own data: on macOS
// the app bundle's icns (theme names don't exist there), elsewhere the theme
// name. The branches are exclusive per OS, so a mac bundle is never probed on
// Linux or vice versa. macApp is the bundle name (e.g. "Google Chrome"); when
// it can't be found on macOS the icon is nil.
func iconFor(themeName, macApp string) fyne.Resource {
	if runtime.GOOS != "darwin" {
		return iconResource(themeName)
	}
	if macApp == "" {
		return nil
	}
	p := appBundlePath(macApp)
	if p == "" {
		return nil
	}
	return macIconFromPath(p)
}

// macIconFromPath resolves an app bundle's icon, honoring the CFBundleIconFile
// the bundle declares rather than assuming AppIcon.icns.
func macIconFromPath(appPath string) fyne.Resource {
	icon := "AppIcon.icns"
	if info, err := readInfoPlist(appPath); err == nil && info.IconFile != "" {
		icon = info.IconFile
		if !strings.HasSuffix(icon, ".icns") {
			icon += ".icns"
		}
	}
	return appIconResource(filepath.Join(appPath, "Contents", "Resources", icon))
}

// lookupIcon resolves base by stat-ing the common pixmaps and hicolor
// size/scalable app dirs only. It never walks the icon tree: /usr/share/icons
// on a real system can hold hundreds of thousands of files (2.6GB here), so a
// recursive fallback is off the table -- a miss returns nil, no icon.
func lookupIcon(base string) fyne.Resource {
	for _, root := range iconRoots {
		for _, dir := range iconDirs(root) {
			for _, ext := range []string{"png", "svg", "xpm"} {
				if r := resourceFromFile(filepath.Join(dir, base+"."+ext)); r != nil {
					return r
				}
			}
		}
	}
	return nil
}

// iconDirs returns the common on-disk locations for a themed app icon, most
// likely first. Extensions are tried per directory so png wins over svg.
func iconDirs(root string) []string {
	return []string{
		filepath.Join(root, "icons", "hicolor", "scalable", "apps"),
		filepath.Join(root, "icons", "hicolor", "128x128", "apps"),
		filepath.Join(root, "icons", "hicolor", "64x64", "apps"),
		filepath.Join(root, "icons", "hicolor", "48x48", "apps"),
		filepath.Join(root, "icons", "hicolor", "32x32", "apps"),
		filepath.Join(root, "icons", "hicolor", "256x256", "apps"),
		filepath.Join(root, "pixmaps"),
	}
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
