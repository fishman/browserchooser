//go:build darwin

package main

/*
void roundWindow(void *win, double radius);
*/
import "C"

import (
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

// windowInset is the transparent margin kept inside the rounded window corners
// so the masksToBounds clip never cuts the input's border. The window grows by
// 2*inset to keep the content size constant.
func windowInset() float32 { return float32(macCornerRadius) }

// roundCorners makes the borderless picker window transparent with rounded
// corners and a matching shadow. Applied natively just after the window shows.
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
					C.roundWindow(unsafe.Pointer(mc.NSWindow), C.double(macCornerRadius))
				}
			})
		})
	})
}
