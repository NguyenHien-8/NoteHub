package screens

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
		height := max(float32(38), object.MinSize().Height)
		return container.NewGridWrap(fyne.NewSize(width, height), object)
	}
	field := func(label string, object fyne.CanvasObject) fyne.CanvasObject {
		name := widget.NewLabel(label)
		name.Importance = widget.LowImportance
		return container.NewVBox(name, object)
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
	textSize.Validator = func(value string) error {
		size, err := strconv.ParseFloat(value, 32)
		if err != nil || size < float64(typography.MinTextSize) || size > float64(typography.MaxTextSize) {
			return fmt.Errorf("enter a size from %.0f to %.0f px", typography.MinTextSize, typography.MaxTextSize)
		}
		return nil
	}
	textHint := widget.NewLabel(fmt.Sprintf("Custom size: %.0f–%.0f px. The value follows Windows/macOS/Linux display scaling.", typography.MinTextSize, typography.MaxTextSize))
	textHint.Importance = widget.LowImportance
	fontHint := widget.NewLabel("The font list uses families installed on this computer; NoteHub does not bundle system font files.")
	fontHint.Importance = widget.LowImportance
	fontHint.Wrapping = fyne.TextWrapWord

	applyTypography := func(showError bool) bool {
		if setTypography == nil {
			return true
		}
		size, err := strconv.ParseFloat(textSize.Text, 64)
		if err != nil || size < float64(typography.MinTextSize) || size > float64(typography.MaxTextSize) {
			if showError {
				env.Error(fmt.Errorf("text size must be between %.0f and %.0f px", typography.MinTextSize, typography.MaxTextSize))
			}
			return false
		}
		currentSize = float32(size)
		setTypography(font.Selected, size)
		return true
	}
	font.OnChanged = func(string) {
		if !applyTypography(false) {
			textSize.SetText(formatTextSize(currentSize))
			applyTypography(false)
		}
	}
	textSize.OnSubmitted = func(string) { applyTypography(true) }

	stepTextSize := func(delta float64) {
		size, err := strconv.ParseFloat(textSize.Text, 64)
		if err != nil {
			size = float64(currentSize)
		}
		size = math.Max(float64(typography.MinTextSize), math.Min(float64(typography.MaxTextSize), size+delta))
		textSize.SetText(formatTextSize(float32(size)))
		applyTypography(false)
	}
	decrease := widget.NewButton("−", func() { stepTextSize(-1) })
	increase := widget.NewButton("+", func() { stepTextSize(1) })
	applySize := widget.NewButton("Apply", func() { applyTypography(true) })
	decrease.Importance, increase.Importance, applySize.Importance = widget.LowImportance, widget.LowImportance, widget.LowImportance
	textSizeRow := container.NewHBox(fixedWidth(92, textSize), widget.NewLabel("px"), decrease, increase, applySize)

	typographySettings := container.NewVBox(
		field("Theme", fixedWidth(260, appearance)),
		field("Font", fixedWidth(340, font)),
		fontHint,
		field("Text size", textSizeRow),
		textHint,
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
