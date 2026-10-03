package platform

import "fyne.io/fyne/v2"

const windowSafetyMargin float32 = 8

// InitialWindowSize returns a logical client size fitted to the current desktop
// work area. Platform implementations account for native window chrome and DPI;
// the result keeps a small safety margin but is not capped to an arbitrary
// desktop size, so the first visible frame already fills the usable monitor.
func InitialWindowSize(scale float32) fyne.Size {
	return initialWindowSize(scale)
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
