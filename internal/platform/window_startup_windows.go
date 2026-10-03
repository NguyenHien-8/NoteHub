//go:build windows

package platform

import "syscall"

var showWindowAsync = syscall.NewLazyDLL("user32.dll").NewProc("ShowWindowAsync")

func maximizeNativeWindow(handle uintptr) bool {
	const swMaximize = 3
	// Queue on the native window's owning thread; do not block Fyne's event loop.
	result, _, _ := showWindowAsync.Call(handle, swMaximize)
	return result != 0
}
