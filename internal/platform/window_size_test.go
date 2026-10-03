package platform

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestWindowClientSizeFitsScaledWorkArea(t *testing.T) {
	for _, tc := range []struct {
		name   string
		work   fyne.Size
		scale  float32
		chrome fyne.Size
	}{
		{"1366x768 at150", fyne.NewSize(1366, 728), 1.5, fyne.NewSize(24, 58)},
		{"1920x1080 at130", fyne.NewSize(1920, 1028), 1.3, fyne.NewSize(21, 50)},
		{"2560x1440 at200", fyne.NewSize(2560, 1360), 2, fyne.NewSize(32, 78)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			size := fitClientSize(tc.work, tc.chrome, tc.scale)
			if size.Width*tc.scale+tc.chrome.Width > tc.work.Width-windowSafetyMargin || size.Height*tc.scale+tc.chrome.Height > tc.work.Height-windowSafetyMargin {
				t.Fatalf("logical size %v at scale %v overflows work area %v with chrome %v", size, tc.scale, tc.work, tc.chrome)
			}
			if size.Width < 640 || size.Height < 300 {
				t.Fatalf("unusable initial size: %v", size)
			}
			expectedWidth := (tc.work.Width - windowSafetyMargin - tc.chrome.Width) / tc.scale
			expectedHeight := (tc.work.Height - windowSafetyMargin - tc.chrome.Height) / tc.scale
			if size.Width != expectedWidth || size.Height != expectedHeight {
				t.Fatalf("first frame should fill the usable work area: got %v want %.2fx%.2f", size, expectedWidth, expectedHeight)
			}
		})
	}
}

func TestWindowBoundsCenterOnNegativeOriginMonitor(t *testing.T) {
	x, y := centeredWindowPosition(-1920, -120, 0, 920, 1400, 800)
	if x != -1660 || y != 0 {
		t.Fatalf("got (%d,%d), expected (-1660,0)", x, y)
	}
	x, y = centeredWindowPosition(-1920, -120, 0, 920, 2000, 1100)
	if x != -1920 || y != -120 {
		t.Fatalf("oversized window lost monitor origin: (%d,%d)", x, y)
	}
}

func TestWindowClientSizeHandlesInvalidScale(t *testing.T) {
	if got := fitClientSize(fyne.NewSize(900, 700), fyne.NewSize(16, 40), 0); got != fyne.NewSize(876, 652) {
		t.Fatalf("invalid scale fallback: %v", got)
	}
}
