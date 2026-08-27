//go:build darwin

package main

/*
void installURLReceiver(void (*cb)(const char *));
*/
import "C"

import (
	"fyne.io/fyne/v2"
)

var urlCallback func(string)

//export onIncomingURL
func onIncomingURL(s *C.char) {
	if s == nil || urlCallback == nil {
		return
	}
	url := C.GoString(s)
	fyne.Do(func() { urlCallback(url) })
}

// installURLReceiver registers a callback fired on the main thread when macOS
// opens this app for an http(s) link (as the default browser). No-op unless a
// native NSApplication delegate is present.
func installURLReceiver(fn func(string)) {
	urlCallback = fn
	C.installURLReceiver(C.onIncomingURL)
}
