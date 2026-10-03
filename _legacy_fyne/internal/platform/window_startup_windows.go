//go:build windows

package platform

import "github.com/go-gl/glfw/v3.3/glfw"

func prepareWindowForShow() func() {
	return prepareMaximizedCreationHint(func(maximized bool) {
		if maximized {
			// Fyne creates desktop windows hidden first. Asking GLFW to create the
			// window maximized means the first frame that becomes visible already
			// uses Windows' maximized work-area bounds (taskbar, frame and DPI
			// included), instead of showing a normal frame and maximizing it later.
			glfw.WindowHint(glfw.Maximized, glfw.True)
			return
		}
		// Window hints are process-wide. Reset immediately after Fyne has created
		// the NoteHub window so any future native window keeps its normal default.
		glfw.WindowHint(glfw.Maximized, glfw.False)
	})
}
