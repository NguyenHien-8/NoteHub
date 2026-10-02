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
	s := &surface{content: container.New(layout.NewCustomPaddedLayout(12, 12, 12, 12), content)}
	s.ExtendBaseWidget(s)
	return s
}

type surface struct {
	widget.BaseWidget
	content fyne.CanvasObject
}

func (s *surface) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.White)
	bg.CornerRadius, bg.StrokeWidth = 12, 1
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
	red, green, blue, _ := background.RGBA()
	// A forced Light/Dark theme can intentionally ignore the system variant.
	// Use its resolved color so appearance settings also update the card surface.
	if (red+green+blue)/3 < 0x8000 {
		r.bg.FillColor = background
		r.bg.StrokeColor = color.NRGBA{R: 55, G: 65, B: 81, A: 255}
	} else {
		r.bg.FillColor = color.White
		r.bg.StrokeColor = color.NRGBA{R: 229, G: 234, B: 240, A: 255}
	}
	r.bg.Refresh()
	r.owner.content.Refresh()
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
