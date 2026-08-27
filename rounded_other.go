//go:build !darwin

package main

import "fyne.io/fyne/v2"

func roundCorners(fyne.Window) {}

// windowInset is 0 on non-rounded windows: no transparent margin.
func windowInset() float32 { return 0 }
