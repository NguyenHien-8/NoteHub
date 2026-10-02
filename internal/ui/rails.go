package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

const (
	defaultLeftRailWidth  float32 = 250
	defaultRightRailWidth float32 = 280
	minLeftRailWidth      float32 = 180
	maxLeftRailWidth      float32 = 420
	minRightRailWidth     float32 = 272
	maxRightRailWidth     float32 = 456
	minCenterWidth        float32 = 400
	railHandleWidth       float32 = 6
)

// Rails is a three-column workspace with independently resizable/collapsible
// side rails. Widths and manual collapse state are persisted in preferences.
// Automatic suppression on narrow windows is intentionally not persisted.
type Rails struct {
	widget.BaseWidget
	left, center, right fyne.CanvasObject
	prefs               fyne.Preferences
	leftWidth           float32
	rightWidth          float32
	leftCollapsed       bool
	rightCollapsed      bool
	preferRightNarrow   bool
	leftHandle          *railHandle
	rightHandle         *railHandle
}

func NewRails(left, center, right fyne.CanvasObject, prefs fyne.Preferences) *Rails {
	r := &Rails{
		left:           left,
		center:         center,
		right:          right,
		prefs:          prefs,
		leftWidth:      defaultLeftRailWidth,
		rightWidth:     defaultRightRailWidth,
		leftCollapsed:  false,
		rightCollapsed: false,
	}
	if prefs != nil {
		r.leftWidth = clamp32(float32(prefs.FloatWithFallback("layout.left.width", float64(defaultLeftRailWidth))), minLeftRailWidth, maxLeftRailWidth)
		r.rightWidth = clamp32(float32(prefs.FloatWithFallback("layout.right.width", float64(defaultRightRailWidth))), minRightRailWidth, maxRightRailWidth)
		r.leftCollapsed = prefs.BoolWithFallback("layout.left.collapsed", false)
		r.rightCollapsed = prefs.BoolWithFallback("layout.right.collapsed", false)
	}
	r.leftHandle = newRailHandle(r, true)
	r.rightHandle = newRailHandle(r, false)
	r.ExtendBaseWidget(r)
	return r
}

func (r *Rails) CreateRenderer() fyne.WidgetRenderer {
	objects := []fyne.CanvasObject{r.left, r.leftHandle, r.center, r.rightHandle, r.right}
	return &railsRenderer{rails: r, objects: objects}
}

func (r *Rails) MinSize() fyne.Size {
	// Deliberately ignore child MinSize values. Note content, filenames or large
	// previews must never force the native desktop window outside the monitor.
	return fyne.NewSize(640, 240)
}

func (r *Rails) LeftCollapsed() bool  { return r.leftCollapsed }
func (r *Rails) RightCollapsed() bool { return r.rightCollapsed }

func (r *Rails) ToggleLeft() {
	if !r.leftCollapsed && !r.left.Visible() {
		// The rail is only hidden by the narrow-window policy. Make it the
		// preferred visible rail without converting that temporary state into a
		// persisted manual collapse.
		r.preferRightNarrow = false
	} else {
		r.leftCollapsed = !r.leftCollapsed
		if !r.leftCollapsed {
			r.preferRightNarrow = false
		}
	}
	if r.prefs != nil {
		r.prefs.SetBool("layout.left.collapsed", r.leftCollapsed)
	}
	r.Refresh()
}

func (r *Rails) ToggleRight() {
	if !r.rightCollapsed && !r.right.Visible() {
		r.preferRightNarrow = true
	} else {
		r.rightCollapsed = !r.rightCollapsed
		if !r.rightCollapsed {
			r.preferRightNarrow = true
		}
	}
	if r.prefs != nil {
		r.prefs.SetBool("layout.right.collapsed", r.rightCollapsed)
	}
	r.Refresh()
}

func (r *Rails) dragLeft(delta float32) {
	if r.leftCollapsed || !r.left.Visible() {
		return
	}
	maxAllowed := r.Size().Width - minCenterWidth - railHandleWidth
	if r.right.Visible() {
		maxAllowed -= r.rightWidth + railHandleWidth
	}
	maxAllowed = min32(maxLeftRailWidth, max32(minLeftRailWidth, maxAllowed))
	r.leftWidth = clamp32(r.leftWidth+delta, minLeftRailWidth, maxAllowed)
	r.Refresh()
}

func (r *Rails) dragRight(delta float32) {
	if r.rightCollapsed || !r.right.Visible() {
		return
	}
	maxAllowed := r.Size().Width - minCenterWidth - railHandleWidth
	if r.left.Visible() {
		maxAllowed -= r.leftWidth + railHandleWidth
	}
	maxAllowed = min32(maxRightRailWidth, max32(minRightRailWidth, maxAllowed))
	// The right divider moves left when the right rail grows.
	r.rightWidth = clamp32(r.rightWidth-delta, minRightRailWidth, maxAllowed)
	r.Refresh()
}

func (r *Rails) persistLeft() {
	if r.prefs != nil {
		r.prefs.SetFloat("layout.left.width", float64(r.leftWidth))
	}
}

func (r *Rails) persistRight() {
	if r.prefs != nil {
		r.prefs.SetFloat("layout.right.width", float64(r.rightWidth))
	}
}

type railsRenderer struct {
	rails   *Rails
	objects []fyne.CanvasObject
}

func (r *railsRenderer) Layout(size fyne.Size) {
	owner := r.rails
	showLeft := !owner.leftCollapsed
	showRight := !owner.rightCollapsed

	needed := minCenterWidth
	if showLeft {
		needed += owner.leftWidth + railHandleWidth
	}
	if showRight {
		needed += owner.rightWidth + railHandleWidth
	}
	if size.Width < needed && showLeft && showRight {
		if owner.preferRightNarrow {
			showLeft = false
		} else {
			showRight = false
		}
	}
	// If one rail still consumes too much room, suppress it automatically. This
	// does not change the user's manual collapse preference.
	if showLeft && size.Width < owner.leftWidth+railHandleWidth+minCenterWidth {
		showLeft = false
	}
	if showRight && size.Width < owner.rightWidth+railHandleWidth+minCenterWidth {
		showRight = false
	}

	x := float32(0)
	setVisible(owner.left, showLeft)
	setVisible(owner.leftHandle, showLeft)
	setVisible(owner.right, showRight)
	setVisible(owner.rightHandle, showRight)

	if showLeft {
		owner.left.Move(fyne.NewPos(x, 0))
		owner.left.Resize(fyne.NewSize(owner.leftWidth, size.Height))
		x += owner.leftWidth
		owner.leftHandle.Move(fyne.NewPos(x, 0))
		owner.leftHandle.Resize(fyne.NewSize(railHandleWidth, size.Height))
		x += railHandleWidth
	}

	rightStart := size.Width
	if showRight {
		rightStart -= owner.rightWidth
		owner.right.Move(fyne.NewPos(rightStart, 0))
		owner.right.Resize(fyne.NewSize(owner.rightWidth, size.Height))
		rightStart -= railHandleWidth
		owner.rightHandle.Move(fyne.NewPos(rightStart, 0))
		owner.rightHandle.Resize(fyne.NewSize(railHandleWidth, size.Height))
	}

	centerWidth := max32(0, rightStart-x)
	owner.center.Move(fyne.NewPos(x, 0))
	owner.center.Resize(fyne.NewSize(centerWidth, size.Height))
}

func (r *railsRenderer) MinSize() fyne.Size           { return r.rails.MinSize() }
func (r *railsRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *railsRenderer) Destroy()                     {}
func (r *railsRenderer) Refresh() {
	// Resizing a rail only changes geometry. Refreshing the full left/center/right
	// widget trees on every mouse-move causes visible flashing on Windows.
	r.Layout(r.rails.Size())
}

func setVisible(object fyne.CanvasObject, visible bool) {
	if visible {
		if !object.Visible() {
			object.Show()
		}
		return
	}
	if object.Visible() {
		object.Hide()
	}
}

type railHandle struct {
	widget.BaseWidget
	owner *Rails
	left  bool
}

func newRailHandle(owner *Rails, left bool) *railHandle {
	h := &railHandle{owner: owner, left: left}
	h.ExtendBaseWidget(h)
	return h
}

func (h *railHandle) CreateRenderer() fyne.WidgetRenderer {
	line := canvas.NewRectangle(color.NRGBA{R: 148, G: 163, B: 184, A: 80})
	return widget.NewSimpleRenderer(line)
}

func (h *railHandle) Cursor() desktop.Cursor { return desktop.HResizeCursor }
func (h *railHandle) Dragged(e *fyne.DragEvent) {
	if h.left {
		h.owner.dragLeft(e.Dragged.DX)
	} else {
		h.owner.dragRight(e.Dragged.DX)
	}
}
func (h *railHandle) DragEnd() {
	if h.left {
		h.owner.persistLeft()
	} else {
		h.owner.persistRight()
	}
}

func clamp32(v, lo, hi float32) float32 { return min32(max32(v, lo), hi) }
func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
