package platform

import "fyne.io/fyne/v2"

const windowSafetyMargin float32 = 16

// InitialWindowSize returns a conservative logical client size for the current
// desktop. Platform implementations may use the OS work area; the result is
// always bounded so content cannot force an oversized first launch.
func InitialWindowSize(scale float32) fyne.Size {
	return initialWindowSize(scale)
}

func fitClientSize(workArea, chrome fyne.Size, scale float32) fyne.Size {
	if scale <= 0 {
		scale = 1
	}
	width := (workArea.Width - windowSafetyMargin - chrome.Width) / scale
	height := (workArea.Height - windowSafetyMargin - chrome.Height) / scale
	width = clampSize(width, 640, 1240)
	height = clampSize(height, 300, 780)
	return fyne.NewSize(width, height)
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

func clampSize(value, low, high float32) float32 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
