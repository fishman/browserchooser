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

// setDefault registers browserchooser as the default handler for http(s),
// ftp, and text/html on Linux, installing the desktop file first if needed.
func setDefault() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("--set-default is only supported on Linux")
	}
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
