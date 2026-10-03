package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// HintButton retains standard Button keyboard/focus behavior and gives an
// icon-only control a readable label on hover or keyboard focus.
type HintButton struct {
	widget.Button
	Hint  string
	hints *HintLayer
}

func NewHintButton(label string, icon fyne.Resource, tapped func()) *HintButton {
	b := &HintButton{Button: widget.Button{Text: label, Icon: icon, OnTapped: tapped}, Hint: label}
	b.ExtendBaseWidget(b)
	return b
}

func (b *HintButton) SetHintLayer(hints *HintLayer)     { b.hints = hints }
func (b *HintButton) MouseIn(event *desktop.MouseEvent) { b.Button.MouseIn(event); b.showHint() }
func (b *HintButton) MouseOut()                         { b.hideHint(); b.Button.MouseOut() }
func (b *HintButton) FocusGained()                      { b.Button.FocusGained(); b.showHint() }
func (b *HintButton) FocusLost()                        { b.hideHint(); b.Button.FocusLost() }
func (b *HintButton) Tapped(event *fyne.PointEvent)     { b.hideHint(); b.Button.Tapped(event) }
func (b *HintButton) TypedKey(event *fyne.KeyEvent)     { b.hideHint(); b.Button.TypedKey(event) }

func (b *HintButton) showHint() {
	if b.hints != nil && b.Text == "" && b.Hint != "" {
		b.hints.show(b)
	}
}
func (b *HintButton) hideHint() {
	if b.hints != nil && b.hints.active == b {
		b.hints.hide()
	}
}

// HintLayer lives in the ordinary content tree, so labels never capture the
// mouse or dismiss keyboard focus as a Fyne PopUp overlay would.
type HintLayer struct {
	root       *fyne.Container
	bubble     *fyne.Container
	label      *widget.Label
	background *canvas.Rectangle
	active     *HintButton
}

func WithHints(content fyne.CanvasObject) (fyne.CanvasObject, *HintLayer) {
	h := &HintLayer{label: widget.NewLabel(""), background: canvas.NewRectangle(nil)}
	h.background.CornerRadius = 6
	h.bubble = container.NewStack(h.background, container.NewPadded(h.label))
	h.bubble.Hide()
	h.root = container.NewWithoutLayout(h.bubble)
	return container.NewStack(content, h.root), h
}

func (h *HintLayer) show(button *HintButton) {
	app := fyne.CurrentApp()
	if app == nil || app.Driver().CanvasForObject(button) == nil {
		return
	}
	h.active = button
	h.label.SetText(button.Hint)
	h.background.FillColor = app.Settings().Theme().Color(theme.ColorNameOverlayBackground, app.Settings().ThemeVariant())
	h.background.Refresh()
	size := h.bubble.MinSize()
	position := app.Driver().AbsolutePositionForObject(button).Add(fyne.NewPos(button.Size().Width+6, 0))
	position.X = max(0, min(position.X, h.root.Size().Width-size.Width))
	position.Y = max(0, min(position.Y, h.root.Size().Height-size.Height))
	h.bubble.Resize(size)
	h.bubble.Move(position)
	h.bubble.Show()
}

func (h *HintLayer) hide() { h.active = nil; h.bubble.Hide() }
