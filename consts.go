package main

import (
	"runtime"
	"time"
)

const (
	// app metadata
	appID    = "dev.fishman.browserchooser"
	winTitle = "Browser Chooser"

	// window: fixed 7 slots (input bar + 5 rows + qr bar)
	winW = 260
	rowH = 44
	qrH  = 30
	winH = 44 + 5*rowH + qrH

	// message ids
	msgTitle       = "title"
	msgPlaceholder = "placeholder"
	msgCopyLink    = "copyLink"
	msgShowQR      = "showQR"
	msgHostTop     = "hostTop"

	// selection highlight debounce so rapid keystrokes don't jump the cursor
	selDebounce = 20 * time.Millisecond

	// frecency decay half life for recency-weighted ranking
	frecencyHalfLife = 30 * 24 * time.Hour

	// minimum host opens before a per-host browser preference is trusted
	minHostUses = 3
)

const (
	// list rows shown before the pinned copy/qr slots
	maxRows = 4

	// qr code render side in pixels
	qrSize = 128
)

// windowHeight requests a little more than the content's exact minimum height
// only where window decorations eat into it (the macOS title bar), so the
// bottom copy row and QR button are never clipped there. Linux keeps the exact
// fit; SetFixedSize locks the size.
func windowHeight() float32 {
	if runtime.GOOS == "darwin" {
		return float32(winH + rowH)
	}
	return float32(winH)
}
