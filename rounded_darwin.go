//go:build darwin

package main

/*
void roundWindow(void *win);
*/
import "C"

import (
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

// roundCorners makes the borderless picker window transparent with rounded
// corners and a matching shadow. Fyne's splash window is a plain square
// NSWindow, so the rounding is applied natively just after the window shows.
// No-op when the window does not expose native access.
func roundCorners(w fyne.Window) {
	nw, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	time.AfterFunc(80*time.Millisecond, func() {
		fyne.Do(func() {
			nw.RunNative(func(ctx any) {
				if mc, ok := ctx.(driver.MacWindowContext); ok {
					C.roundWindow(unsafe.Pointer(mc.NSWindow))
				}
			})
		})
	})
}
