package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const defaultTextSize float32 = 14

const brandTextSizeName fyne.ThemeSizeName = "notehub.brand.text"

type noteTheme struct {
	mode       string
	fontFamily string
	textSize   float32
}

var favoriteIcon = fyne.NewStaticResource("favorite.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path d="m12 3 2.8 5.8 6.4.9-4.6 4.5 1.1 6.3L12 17.5l-5.7 3 1.1-6.3-4.6-4.5 6.4-.9z" fill="none" stroke="#F5A623" stroke-width="2" stroke-linejoin="round"/></svg>`))

func NewTheme(mode string) fyne.Theme { return newConfiguredTheme(mode, "System", defaultTextSize) }

func newConfiguredTheme(mode, fontFamily string, textSize float32) fyne.Theme {
	if textSize < 12 {
		textSize = 12
	}
	if textSize > 18 {
		textSize = 18
	}
	if !strings.EqualFold(fontFamily, "Monospace") {
		fontFamily = "System"
	}
	return &noteTheme{mode: mode, fontFamily: fontFamily, textSize: textSize}
}

func (d *Desktop) applyTheme() {
	prefs := d.application.Preferences()
	appearance := prefs.StringWithFallback("appearance", "Light")
	font := prefs.StringWithFallback("font-family", "System")
	size := float32(prefs.FloatWithFallback("font-size", float64(defaultTextSize)))
	d.application.Settings().SetTheme(newConfiguredTheme(appearance, font, size))
}

// SetAppearance must be called on the UI thread, like other widget updates.
func (d *Desktop) SetAppearance(mode string) {
	d.application.Preferences().SetString("appearance", mode)
	d.applyTheme()
}

// SetTypography updates the built-in typeface choice and global UI text size.
// Keeping this inside the theme means dialogs, forms and future screens remain
// visually consistent instead of each widget carrying ad-hoc sizing code.
func (d *Desktop) SetTypography(fontFamily string, textSize float64) {
	prefs := d.application.Preferences()
	prefs.SetString("font-family", fontFamily)
	prefs.SetFloat("font-size", textSize)
	d.applyTheme()
}

func (t *noteTheme) variant(v fyne.ThemeVariant) fyne.ThemeVariant {
	if t.mode == "Light" {
		return theme.VariantLight
	}
	if t.mode == "Dark" {
		return theme.VariantDark
	}
	return v
}
func (t *noteTheme) Color(name fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	v = t.variant(v)
	if name == theme.ColorNamePrimary {
		return color.NRGBA{R: 8, G: 123, B: 255, A: 255}
	}
	if v == theme.VariantDark {
		switch name {
		case theme.ColorNameDisabled, theme.ColorNamePlaceHolder:
			return color.NRGBA{R: 148, G: 163, B: 184, A: 255}
		case theme.ColorNameSeparator, theme.ColorNameInputBorder:
			return color.NRGBA{R: 51, G: 65, B: 85, A: 255}
		}
	}
	if v == theme.VariantLight {
		switch name {
		case theme.ColorNameBackground:
			return color.NRGBA{R: 248, G: 250, B: 252, A: 255}
		case theme.ColorNameForeground:
			return color.NRGBA{R: 15, G: 23, B: 42, A: 255}
		case theme.ColorNameDisabled, theme.ColorNamePlaceHolder:
			return color.NRGBA{R: 100, G: 116, B: 139, A: 255}
		case theme.ColorNameInputBackground, theme.ColorNameOverlayBackground:
			return color.White
		case theme.ColorNameInputBorder, theme.ColorNameSeparator:
			return color.NRGBA{R: 229, G: 234, B: 240, A: 255}
		case theme.ColorNameSelection, theme.ColorNameHover, theme.ColorNameFocus:
			return color.NRGBA{R: 234, G: 243, B: 255, A: 255}
		case theme.ColorNameButton:
			return color.NRGBA{R: 241, G: 245, B: 249, A: 255}
		}
	}
	return theme.DefaultTheme().Color(name, v)
}
func (t *noteTheme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Symbol {
		return theme.DefaultTheme().Font(style)
	}
	if strings.EqualFold(t.fontFamily, "Monospace") {
		return theme.DefaultTextMonospaceFont()
	}
	return theme.DefaultTheme().Font(style)
}
func (t *noteTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }
func (t *noteTheme) Size(n fyne.ThemeSizeName) float32 {
	if n == brandTextSizeName {
		// Keep the NoteHub wordmark at the original visual emphasis even when
		// the rest of the application is made more compact.
		return theme.DefaultTheme().Size(theme.SizeNameHeadingText)
	}
	switch n {
	case theme.SizeNameText:
		return t.textSize
	case theme.SizeNameCaptionText:
		return max(10, t.textSize-2)
	case theme.SizeNameSubHeadingText:
		return t.textSize + 2
	case theme.SizeNameHeadingText:
		return t.textSize + 5
	case theme.SizeNameInlineIcon:
		return t.textSize + 4
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameScrollBar:
		return 10
	case theme.SizeNameScrollBarSmall:
		return 3
	case theme.SizeNameInputRadius:
		return 10
	case theme.SizeNameSelectionRadius:
		return 10
	}
	return theme.DefaultTheme().Size(n)
}
