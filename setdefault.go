package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

//go:embed dev.fishman.browserchooser.desktop
var desktopFile []byte

const desktopID = "dev.fishman.browserchooser.desktop"

// setDefault registers browserchooser as the default handler for http(s).
// On Linux that installs a desktop file and calls xdg-mime; on macOS it asks
// LaunchServices to route http/https to this app's bundle id.
func setDefault() error {
	switch runtime.GOOS {
	case "linux":
		return setDefaultLinux()
	case "darwin":
		return setDefaultDarwin()
	default:
		return fmt.Errorf("--set-default is not supported on %s", runtime.GOOS)
	}
}

func setDefaultLinux() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".local", "share", "applications")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	dst := filepath.Join(dir, desktopID)
	if err := os.WriteFile(dst, desktopFile, 0o644); err != nil {
		return err
	}
	for _, m := range []string{"text/html", "x-scheme-handler/http", "x-scheme-handler/https", "x-scheme-handler/ftp"} {
		if out, err := exec.Command("xdg-mime", "default", desktopID, m).CombinedOutput(); err != nil {
			return fmt.Errorf("xdg-mime %s: %v: %s", m, err, out)
		}
	}
	return nil
}

// setDefaultDarwin tells LaunchServices to open http/https links in this app.
// There is no shell builtin for it, so it runs a tiny Swift one-liner against
// CoreServices. Requires Xcode command line tools (/usr/bin/swift).
func setDefaultDarwin() error {
	script := "import CoreServices\n" +
		"for scheme in [\"http\", \"https\"] {\n" +
		"    LSSetDefaultHandlerForURLScheme(scheme as CFString, \"" + appID + "\" as CFString)\n" +
		"}\n"
	f, err := os.CreateTemp("", "setdefault-*.swift")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(script); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if out, err := exec.Command("/usr/bin/swift", f.Name()).CombinedOutput(); err != nil {
		return fmt.Errorf("swift: %v: %s", err, out)
	}
	return nil
}
