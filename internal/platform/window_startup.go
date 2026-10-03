package platform

import "sync"

// PrepareWindowForShow configures the native desktop window's initial state
// before Fyne creates its GLFW window. The returned function must be called
// immediately after Window.Show so the process-wide GLFW creation hint does
// not leak into any later windows.
//
// On Windows this requests a maximized window at creation time. This is
// intentionally different from maximizing an already-visible HWND: the first
// visible frame is already fitted to the Windows work area, eliminating the
// normal-window -> maximized flash during startup.
func PrepareWindowForShow() func() {
	return prepareWindowForShow()
}

// prepareMaximizedCreationHint makes a process-wide creation hint scoped to a
// single window creation. Keeping the restore idempotent makes it safe to use
// with defer even if startup code returns early in the future.
func prepareMaximizedCreationHint(set func(bool)) func() {
	set(true)
	var once sync.Once
	return func() {
		once.Do(func() { set(false) })
	}
}
