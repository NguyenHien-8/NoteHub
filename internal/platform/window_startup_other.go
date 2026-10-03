//go:build !windows

package platform

func maximizeNativeWindow(uintptr) bool { return false }
