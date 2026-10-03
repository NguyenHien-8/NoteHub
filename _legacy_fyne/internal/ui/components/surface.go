package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var primaryBlue = color.NRGBA{R: 8, G: 123, B: 255, A: 255}

// Surface wraps content in a rounded card and follows the current theme.
func Surface(content fyne.CanvasObject) fyne.CanvasObject {
	s := &surface{content: container.New(layout.NewCustomPaddedLayout(10, 10, 10, 10), content)}
	s.ExtendBaseWidget(s)
	return s
}

type surface struct {
	widget.BaseWidget
	content fyne.CanvasObject
}

func (s *surface) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.White)
	bg.CornerRadius, bg.StrokeWidth = 10, 1
	r := &surfaceRenderer{owner: s, bg: bg, objects: []fyne.CanvasObject{bg, s.content}}
	r.Refresh()
	return r
}

type surfaceRenderer struct {
	owner   *surface
	bg      *canvas.Rectangle
	objects []fyne.CanvasObject
}

func (r *surfaceRenderer) Layout(size fyne.Size)        { r.bg.Resize(size); r.owner.content.Resize(size) }
func (r *surfaceRenderer) MinSize() fyne.Size           { return r.owner.content.MinSize() }
func (r *surfaceRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *surfaceRenderer) Destroy()                     {}
func (r *surfaceRenderer) Refresh() {
	variant := fyne.CurrentApp().Settings().ThemeVariant()
	background := r.owner.Theme().Color(theme.ColorNameInputBackground, variant)
	if resolvedDark(background) {
		r.bg.FillColor = background
		r.bg.StrokeColor = color.NRGBA{R: 55, G: 65, B: 81, A: 255}
	} else {
		r.bg.FillColor = color.White
		r.bg.StrokeColor = color.NRGBA{R: 229, G: 234, B: 240, A: 255}
	}
	r.bg.Refresh()
	r.owner.content.Refresh()
}

type panelTone uint8

const (
	panelSide panelTone = iota
	panelWorkspace
)

// SidePanel provides a soft activity/sidebar surface similar to a modern code
// editor. WorkspacePanel uses a calmer center surface so the three regions stay
// visually distinct without permanent divider lines.
func SidePanel(content fyne.CanvasObject) fyne.CanvasObject {
	return newChromePanel(content, panelSide, 10)
}

func WorkspacePanel(content fyne.CanvasObject) fyne.CanvasObject {
	return newChromePanel(content, panelWorkspace, 12)
}

type chromePanel struct {
	widget.BaseWidget
	content fyne.CanvasObject
	tone    panelTone
}

func newChromePanel(content fyne.CanvasObject, tone panelTone, padding float32) *chromePanel {
	p := &chromePanel{content: container.New(layout.NewCustomPaddedLayout(padding, padding, padding, padding), content), tone: tone}
	p.ExtendBaseWidget(p)
	return p
}

func (p *chromePanel) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(color.Transparent)
	background.CornerRadius = 12
	background.StrokeWidth = 1
	r := &chromePanelRenderer{owner: p, background: background, objects: []fyne.CanvasObject{background, p.content}}
	r.Refresh()
	return r
}

type chromePanelRenderer struct {
	owner      *chromePanel
	background *canvas.Rectangle
	objects    []fyne.CanvasObject
}

func (r *chromePanelRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
	r.owner.content.Resize(size)
}
func (r *chromePanelRenderer) MinSize() fyne.Size           { return r.owner.content.MinSize() }
func (r *chromePanelRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *chromePanelRenderer) Destroy()                     {}
func (r *chromePanelRenderer) Refresh() {
	variant := fyne.CurrentApp().Settings().ThemeVariant()
	resolved := r.owner.Theme().Color(theme.ColorNameBackground, variant)
	dark := resolvedDark(resolved)
	if dark {
		r.background.StrokeColor = color.NRGBA{R: 51, G: 65, B: 85, A: 230}
		if r.owner.tone == panelSide {
			r.background.FillColor = color.NRGBA{R: 24, G: 33, B: 45, A: 255}
		} else {
			r.background.FillColor = color.NRGBA{R: 15, G: 23, B: 32, A: 255}
		}
	} else {
		r.background.StrokeColor = color.NRGBA{R: 220, G: 228, B: 238, A: 255}
		if r.owner.tone == panelSide {
			r.background.FillColor = color.NRGBA{R: 242, G: 246, B: 251, A: 255}
		} else {
			r.background.FillColor = color.White
		}
	}
	r.background.Refresh()
	r.owner.content.Refresh()
}

func resolvedDark(c color.Color) bool {
	red, green, blue, _ := c.RGBA()
	return (red+green+blue)/3 < 0x8000
}

// scopedTheme keeps standard Fyne button focus, keyboard and disabled behavior
// while giving selected navigation and tag pills the reference colors.
type scopedTheme struct {
	background color.Color
	foreground color.Color
	compact    bool
}

func (t scopedTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return t.background
	case theme.ColorNameForegroundOnPrimary:
		return t.foreground
	}
	return fyne.CurrentApp().Settings().Theme().Color(name, variant)
}
func (t scopedTheme) Font(style fyne.TextStyle) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Font(style)
}
func (t scopedTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(name)
}
func (t scopedTheme) Size(name fyne.ThemeSizeName) float32 {
	if t.compact && name == theme.SizeNamePadding {
		return 2
	}
	if t.compact && name == theme.SizeNameInnerPadding {
		return 4
	}
	return fyne.CurrentApp().Settings().Theme().Size(name)
}

func softButton(button *widget.Button, bg, fg color.Color) fyne.CanvasObject {
	button.Importance = widget.HighImportance
	return container.NewThemeOverride(button, scopedTheme{background: bg, foreground: fg})
}

func tagButton(tag string, onTag func(string)) fyne.CanvasObject {
	button := widget.NewButton(boundedText("#"+trimTag(tag), 32), func() {
		if onTag != nil {
			onTag(trimTag(tag))
		}
	})
	return softButton(button, tagColor(tag), color.NRGBA{R: 51, G: 65, B: 85, A: 255})
}

func trimTag(tag string) string {
	if len(tag) > 0 && tag[0] == '#' {
		return tag[1:]
	}
	return tag
}

// ControlSurface gives compact form controls a visible, theme-aware field
// background while preserving the native Select/Entry interaction, focus,
// keyboard navigation and hover behavior. The child controls use a transparent
// input background so one clean rounded field is rendered instead of stacked
// nested rectangles.
func ControlSurface(content fyne.CanvasObject) fyne.CanvasObject {
	themed := container.NewThemeOverride(content, controlSurfaceTheme{})
	s := &controlSurface{content: container.New(layout.NewCustomPaddedLayout(2, 2, 4, 4), themed)}
	s.ExtendBaseWidget(s)
	return s
}

type controlSurface struct {
	widget.BaseWidget
	content fyne.CanvasObject
}

func (s *controlSurface) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(color.Transparent)
	background.CornerRadius = 9
	background.StrokeWidth = 1
	r := &controlSurfaceRenderer{owner: s, background: background, objects: []fyne.CanvasObject{background, s.content}}
	r.Refresh()
	return r
}

type controlSurfaceRenderer struct {
	owner      *controlSurface
	background *canvas.Rectangle
	objects    []fyne.CanvasObject
}

func (r *controlSurfaceRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
	r.owner.content.Resize(size)
}
func (r *controlSurfaceRenderer) MinSize() fyne.Size           { return r.owner.content.MinSize() }
func (r *controlSurfaceRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *controlSurfaceRenderer) Destroy()                     {}
func (r *controlSurfaceRenderer) Refresh() {
	variant := fyne.CurrentApp().Settings().ThemeVariant()
	th := r.owner.Theme()
	r.background.FillColor = th.Color(theme.ColorNameButton, variant)
	r.background.StrokeColor = th.Color(theme.ColorNameInputBorder, variant)
	r.background.Refresh()
	r.owner.content.Refresh()
}

// controlSurfaceTheme removes a child input's own flat background/border so
// ControlSurface can provide one consistent field chrome around Select, Entry,
// or a compound control such as "12 px". All other theme values delegate to
// the active NoteHub theme and therefore still react immediately to Light/Dark,
// font-family and text-size changes.
type controlSurfaceTheme struct{}

func (controlSurfaceTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameInputBackground, theme.ColorNameInputBorder:
		return color.Transparent
	default:
		return fyne.CurrentApp().Settings().Theme().Color(name, variant)
	}
}
func (controlSurfaceTheme) Font(style fyne.TextStyle) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Font(style)
}
func (controlSurfaceTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(name)
}
func (controlSurfaceTheme) Size(name fyne.ThemeSizeName) float32 {
	return fyne.CurrentApp().Settings().Theme().Size(name)
}
