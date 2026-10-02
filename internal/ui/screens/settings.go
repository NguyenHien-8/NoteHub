package screens

import (
	"context"
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/platform"
	"github.com/NguyenHien-8/NoteHub/internal/ui/components"
	"github.com/NguyenHien-8/NoteHub/internal/ui/dialogs"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

func NewSettings(env *Environment, application fyne.App, manager *dialogs.Manager, version string, setAppearance func(string), setTypography func(string, float64)) fyne.CanvasObject {
	section := func(title string, objects ...fyne.CanvasObject) fyne.CanvasObject {
		return components.Surface(container.NewVBox(append([]fyne.CanvasObject{widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})}, objects...)...))
	}
	appearance := widget.NewSelect([]string{"System", "Light", "Dark"}, func(mode string) {
		if setAppearance != nil {
			setAppearance(mode)
		}
	})
	appearance.SetSelected(application.Preferences().StringWithFallback("appearance", "Light"))
	font := widget.NewSelect([]string{"System", "Monospace"}, nil)
	font.SetSelected(application.Preferences().StringWithFallback("font-family", "System"))
	sizes := []string{"12", "13", "14", "15", "16", "17", "18"}
	textSize := widget.NewSelect(sizes, nil)
	currentSize := int(application.Preferences().FloatWithFallback("font-size", 14))
	if currentSize < 12 || currentSize > 18 {
		currentSize = 14
	}
	textSize.SetSelected(strconv.Itoa(currentSize))
	applyTypography := func() {
		if setTypography == nil {
			return
		}
		size, err := strconv.ParseFloat(textSize.Selected, 64)
		if err != nil {
			return
		}
		setTypography(font.Selected, size)
	}
	font.OnChanged = func(string) { applyTypography() }
	textSize.OnChanged = func(string) { applyTypography() }
	typography := widget.NewForm(
		widget.NewFormItem("Theme", appearance),
		widget.NewFormItem("Font", font),
		widget.NewFormItem("Text size", textSize),
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
		section("Appearance & typography", typography), section("Data", dataPath, open),
		section("Backup", widget.NewLabel("Save a portable ZIP with your notes, favorites and attachments."), container.NewHBox(widget.NewButton("Export Backup", manager.Export), widget.NewButton("Import Backup", manager.Import))),
		section("Sharing", widget.NewForm(widget.NewFormItem("Port", port)), toggle, info),
		section("About", widget.NewLabel("NoteHub "+version), widget.NewLabel("A quiet place for your notes. Stored locally on your computer.")),
	)
	return container.NewVScroll(content)
}

// Application settings screen.
