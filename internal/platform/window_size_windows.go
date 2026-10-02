//go:build windows

package platform

import (
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
)

const (
	spiGetWorkArea = 0x0030
	smCXSizeFrame  = 32
	smCYSizeFrame  = 33
	smCYCaption    = 4
	smCXPadBorder  = 92
)

type winRect struct {
	Left, Top, Right, Bottom int32
}

func initialWindowSize(scale float32) fyne.Size {
	user32 := syscall.NewLazyDLL("user32.dll")
	spi := user32.NewProc("SystemParametersInfoW")
	metrics := user32.NewProc("GetSystemMetrics")
	var rect winRect
	if result, _, _ := spi.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&rect)), 0); result == 0 || rect.Right <= rect.Left || rect.Bottom <= rect.Top {
		return fyne.NewSize(1180, 700)
	}
	metric := func(id uintptr) float32 {
		value, _, _ := metrics.Call(id)
		return float32(int32(value))
	}
	frameX := metric(smCXSizeFrame) + metric(smCXPadBorder)
	frameY := metric(smCYSizeFrame) + metric(smCXPadBorder)
	chrome := fyne.NewSize(frameX*2, metric(smCYCaption)+frameY*2)
	work := fyne.NewSize(float32(rect.Right-rect.Left), float32(rect.Bottom-rect.Top))
	return fitClientSize(work, chrome, scale)
}
