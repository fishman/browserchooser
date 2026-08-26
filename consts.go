package main

import "time"

const (
	// app metadata
	appID = "dev.fishman.browserchooser"

	// preferences; theme variant int: -1 follow system, 0 light, 1 dark
	prefTheme = "themeVariant"

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

	// stats preference key prefixes
	prefCountPrefix = "count."
	prefLastPrefix  = "last."

	// selection highlight debounce so rapid keystrokes don't jump the cursor
	selDebounce = 20 * time.Millisecond

	// frecency decay half life for recency-weighted ranking
	frecencyHalfLife = 30 * 24 * time.Hour
)

const (
	// list rows shown before the pinned copy/qr slots
	maxRows = 4

	// qr code render side in pixels
	qrSize = 128
)
