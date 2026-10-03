//go:build !windows

package platform

import "fyne.io/fyne/v2"

func initialWindowSize(float32) fyne.Size {
	return fyne.NewSize(1180, 700)
}
