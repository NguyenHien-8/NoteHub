package screens

import (
	"context"
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/platform"
	"github.com/NguyenHien-8/NoteHub/internal/ui/components"
	"github.com/NguyenHien-8/NoteHub/internal/ui/dialogs"
	"github.com/NguyenHien-8/NoteHub/internal/ui/typography"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

func NewSettings(env *Environment, application fyne.App, manager *dialogs.Manager, version string, setAppearance func(string), setTypography func(string, float64)) fyne.CanvasObject {
	section := func(title string, objects ...fyne.CanvasObject) fyne.CanvasObject {
		return components.Surface(container.NewVBox(append([]fyne.CanvasObject{widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})}, objects...)...))
	}
	fixedWidth := func(width float32, object fyne.CanvasObject) fyne.CanvasObject {
		height := max(float32(40), object.MinSize().Height)
		return container.NewGridWrap(fyne.NewSize(width, height), object)
	}
	settingRow := func(label string, object fyne.CanvasObject) fyne.CanvasObject {
		name := widget.NewLabel(label)
		name.Importance = widget.LowImportance
		// Border keeps the label pinned to the left edge and the field pinned to
		// the right edge without relying on a Form's label sizing heuristics.
		return container.NewBorder(nil, nil, name, object, layout.NewSpacer())
	}
	control := func(width float32, object fyne.CanvasObject) fyne.CanvasObject {
		return fixedWidth(width, components.ControlSurface(object))
	}

	appearance := widget.NewSelect([]string{"System", "Light", "Dark"}, nil)
	appearance.SetSelected(application.Preferences().StringWithFallback("appearance", "Light"))
	appearance.OnChanged = func(mode string) {
		if setAppearance != nil {
			setAppearance(mode)
		}
	}

	fontOptions := typography.AvailableFamilies()
	currentFont := application.Preferences().StringWithFallback("font-family", "System")
	if !typography.IsKnownFamily(currentFont) {
		currentFont = "System"
	}
	if !containsString(fontOptions, currentFont) {
		fontOptions = append(fontOptions, currentFont)
	}
	font := widget.NewSelect(fontOptions, nil)
	font.SetSelected(currentFont)

	currentSize := typography.ClampTextSize(float32(application.Preferences().FloatWithFallback("font-size", float64(typography.DefaultTextSize))))
	textSize := widget.NewEntry()
	textSize.SetText(formatTextSize(currentSize))
	textSize.SetPlaceHolder(formatTextSize(typography.DefaultTextSize))

	// Typography changes are intentionally live. Theme and font selects apply
	// immediately, while text size applies as soon as the entry contains a
	// complete valid value. Intermediate typing such as "" or "1" is ignored
	// instead of flashing an error dialog or rebuilding the theme unnecessarily.
	applyFont := func() {
		if setTypography != nil {
			setTypography(font.Selected, float64(currentSize))
		}
	}
	font.OnChanged = func(string) { applyFont() }
	textSize.OnChanged = func(value string) {
		size, err := strconv.ParseFloat(value, 64)
		if err != nil || size < float64(typography.MinTextSize) || size > float64(typography.MaxTextSize) {
			return
		}
		if currentSize == float32(size) {
			return
		}
		currentSize = float32(size)
		if setTypography != nil {
			setTypography(font.Selected, size)
		}
	}
	textSize.OnSubmitted = func(value string) {
		size, err := strconv.ParseFloat(value, 64)
		if err != nil || size < float64(typography.MinTextSize) || size > float64(typography.MaxTextSize) {
			textSize.SetText(formatTextSize(currentSize))
		}
	}

	const fieldWidth float32 = 320
	textSizeField := container.NewBorder(nil, nil, nil, widget.NewLabel("px"), textSize)
	typographySettings := container.NewVBox(
		settingRow("Theme", control(fieldWidth, appearance)),
		settingRow("Font", control(fieldWidth, font)),
		settingRow("Text size", control(fieldWidth, textSizeField)),
	)

	dataPath := widget.NewLabel(env.Backend.Paths.Root)
	dataPath.Wrapping = fyne.TextWrapBreak
	open := widget.NewButton("Open data folder", func() {
		work.Run(env.Jobs, func(context.Context) (struct{}, error) {
			return struct{}{}, platform.OpenExternal(env.Backend.Paths.Root)
		}, func(_ struct{}, err error) { env.Error(err) })
	})
	port := widget.NewEntry()
	port.SetText(strconv.Itoa(application.Preferences().IntWithFallback("share-port", 8787)))
	info := widget.NewLabel("Sharing is off. Local links work only on this computer.")
	info.Wrapping = fyne.TextWrapWord
	var toggle *widget.Button
	toggle = widget.NewButton("Enable local share server", func() {
		n, err := strconv.Atoi(port.Text)
		if err != nil || n < 1 || n > 65535 {
			env.Error(fmt.Errorf("enter a port between 1 and 65535"))
			return
		}
		enabled := manager.SharingURL() == ""
		toggle.Disable()
		port.Disable()
		work.Run(env.Jobs, func(context.Context) (struct{}, error) { return struct{}{}, manager.SetSharing(enabled, n) }, func(_ struct{}, err error) {
			toggle.Enable()
			if err != nil {
				port.Enable()
				env.Error(err)
				return
			}
			if url := manager.SharingURL(); url != "" {
				toggle.SetText("Disable local share server")
				info.SetText("Listening at " + url + " · this computer only")
				application.Preferences().SetInt("share-port", n)
			} else {
				toggle.SetText("Enable local share server")
				info.SetText("Sharing is off. No HTTP port is open.")
				port.Enable()
			}
		})
	})
	content := container.NewVBox(
		widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		section("Appearance & typography", typographySettings), section("Data", dataPath, open),
		section("Backup", widget.NewLabel("Save a portable ZIP with your notes, favorites and attachments."), container.NewHBox(widget.NewButton("Export Backup", manager.Export), widget.NewButton("Import Backup", manager.Import))),
		section("Sharing", widget.NewForm(widget.NewFormItem("Port", port)), toggle, info),
		section("About", widget.NewLabel("NoteHub "+version), widget.NewLabel("A quiet place for your notes. Stored locally on your computer.")),
	)
	return container.NewVScroll(content)
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func formatTextSize(size float32) string {
	return strconv.FormatFloat(float64(size), 'f', -1, 32)
}

// Application settings screen.
