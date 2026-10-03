package platform

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/test"
)

type nativeStartupWindow struct {
	fyne.Window
	context any
}

func (w nativeStartupWindow) RunNative(f func(any)) { f(w.context) }

func TestMaximizeRequiresLiveWindowsHandle(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	w := a.NewWindow("startup")
	for _, tc := range []struct {
		name string
		window fyne.Window
		want bool
	}{
		{"headless", w, false},
		{"other platform", nativeStartupWindow{w, driver.UnknownContext{}}, false},
		{"not shown", nativeStartupWindow{w, driver.WindowsWindowContext{}}, false},
		{"shown", nativeStartupWindow{w, driver.WindowsWindowContext{HWND: 42}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			got := maximizeWindow(tc.window, func(handle uintptr) bool {
				called = true
				if handle != 42 { t.Fatalf("wrong native window: %d", handle) }
				return true
			})
			if got != tc.want || called != tc.want {
				t.Fatalf("maximize=%v native call=%v, want %v", got, called, tc.want)
			}
		})
	}
}
