package ui

import (
 "image/color"
 "fyne.io/fyne/v2"
 "fyne.io/fyne/v2/theme"
)

type noteTheme struct {mode string}

var favoriteIcon=fyne.NewStaticResource("favorite.svg",[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path d="m12 3 2.8 5.8 6.4.9-4.6 4.5 1.1 6.3L12 17.5l-5.7 3 1.1-6.3-4.6-4.5 6.4-.9z" fill="none" stroke="#F5A623" stroke-width="2" stroke-linejoin="round"/></svg>`))

func NewTheme(mode string) fyne.Theme {return &noteTheme{mode:mode}}
func (t *noteTheme) variant(v fyne.ThemeVariant) fyne.ThemeVariant {
 if t.mode=="Light" {return theme.VariantLight}; if t.mode=="Dark" {return theme.VariantDark}; return v
}
func (t *noteTheme) Color(name fyne.ThemeColorName,v fyne.ThemeVariant) color.Color {
 v=t.variant(v)
 if name==theme.ColorNamePrimary {return color.NRGBA{R:8,G:123,B:255,A:255}}
 if v==theme.VariantLight {
  switch name {
  case theme.ColorNameBackground: return color.NRGBA{R:248,G:250,B:252,A:255}
  case theme.ColorNameForeground: return color.NRGBA{R:15,G:23,B:42,A:255}
  case theme.ColorNameDisabled,theme.ColorNamePlaceHolder: return color.NRGBA{R:100,G:116,B:139,A:255}
  case theme.ColorNameInputBackground,theme.ColorNameOverlayBackground: return color.White
  case theme.ColorNameInputBorder,theme.ColorNameSeparator: return color.NRGBA{R:229,G:234,B:240,A:255}
  case theme.ColorNameSelection,theme.ColorNameHover,theme.ColorNameFocus: return color.NRGBA{R:234,G:243,B:255,A:255}
  case theme.ColorNameButton: return color.NRGBA{R:241,G:245,B:249,A:255}
  }
 }
 return theme.DefaultTheme().Color(name,v)
}
func(t *noteTheme) Font(s fyne.TextStyle) fyne.Resource {return theme.DefaultTheme().Font(s)}
func(t *noteTheme) Icon(n fyne.ThemeIconName) fyne.Resource {return theme.DefaultTheme().Icon(n)}
func(t *noteTheme) Size(n fyne.ThemeSizeName) float32 {
 switch n {
 case theme.SizeNameText: return 15
 case theme.SizeNamePadding: return 8
 case theme.SizeNameInnerPadding: return 12
 case theme.SizeNameInputRadius: return 10
 case theme.SizeNameSelectionRadius: return 10
 }
 return theme.DefaultTheme().Size(n)
}
