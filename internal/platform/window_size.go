package platform

import (
	"math"

	"fyne.io/fyne/v2"
)

const windowSafetyMargin float32 = 8

// InitialWindowSize returns a logical client size fitted to the current desktop
// work area. The argument is Fyne's user scale preference, not Canvas.Scale():
// a newly-created Fyne canvas reports 1.0 until its native window exists, which
// is too early to represent Windows display DPI reliably.
func InitialWindowSize(userScale float32) fyne.Size {
	return initialWindowSize(userScale)
}

// combinedScale mirrors Fyne's Windows scale combination closely enough for
// pre-show fallback sizing: system content scale multiplied by the optional
// user scale, rounded to one decimal place. On Windows this is only a hidden
// fallback geometry; the GLFW maximized-at-creation hint owns the first visible
// work-area fit.
func combinedScale(userScale, systemScale float32) float32 {
	if userScale <= 0 {
		userScale = 1
	}
	if systemScale <= 0 {
		systemScale = 1
	}
	raw := userScale * systemScale
	return float32(math.Round(float64(raw*10))) / 10
}

func fitClientSize(workArea, chrome fyne.Size, scale float32) fyne.Size {
	if scale <= 0 {
		scale = 1
	}
	width := (workArea.Width - windowSafetyMargin - chrome.Width) / scale
	height := (workArea.Height - windowSafetyMargin - chrome.Height) / scale
	// Do not cap the upper bound. A capped client first appears as a small
	// centered window and then visibly jumps when Windows maximizes it. Sizing
	// directly to the monitor work area makes the very first painted frame fit
	// the screen, while the small safety margin keeps borders inside the work
	// area at fractional DPI scales.
	return fyne.NewSize(max32(width, 640), max32(height, 300))
}

func centeredWindowPosition(left, top, right, bottom, width, height int) (int, int) {
	workWidth, workHeight := right-left, bottom-top
	x, y := left, top
	if width < workWidth {
		x = left + (workWidth-width)/2
	}
	if height < workHeight {
		y = top + (workHeight-height)/2
	}
	return x, y
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
