package platform

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

// MaximizeWindow requests the normal OS maximized state after Show has created
// the native window. Windows chooses the monitor work area, preserving its
// taskbar, title bar and DPI behavior. Other drivers keep the fitted fallback.
func MaximizeWindow(window fyne.Window) bool {
	return maximizeWindow(window, maximizeNativeWindow)
}

func maximizeWindow(window fyne.Window, maximize func(uintptr) bool) bool {
	native, ok := window.(driver.NativeWindow)
	if !ok {
		return false
	}
	maximized := false
	native.RunNative(func(context any) {
		if windows, ok := context.(driver.WindowsWindowContext); ok && windows.HWND != 0 {
			maximized = maximize(windows.HWND)
		}
	})
	return maximized
}
