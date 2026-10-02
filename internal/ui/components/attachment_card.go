package components

import (
	"image"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func attachmentPreview(thumbnail image.Image, hasFile bool) fyne.CanvasObject {
	if thumbnail != nil {
		preview := canvas.NewImageFromImage(thumbnail)
		preview.FillMode = canvas.ImageFillContain
		return container.NewGridWrap(fyne.NewSize(150, 100), preview)
	}
	icon := theme.DocumentIcon()
	if hasFile {
		icon = theme.FileIcon()
	}
	bg := canvas.NewRectangle(color.NRGBA{R: 234, G: 243, B: 255, A: 255})
	bg.CornerRadius = 10
	image := canvas.NewImageFromResource(icon)
	image.FillMode = canvas.ImageFillContain
	return container.NewGridWrap(fyne.NewSize(80, 90), container.NewStack(bg, container.New(layout.NewCustomPaddedLayout(24, 24, 20, 20), image)))
}

func NewAttachmentCard(a domain.Attachment, thumbnail image.Image, open, reveal, remove, goMemo func()) fyne.CanvasObject {
	name := widget.NewLabelWithStyle(boundedText(a.Filename, 70), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	name.Truncation = fyne.TextTruncateEllipsis
	description := a.MIMEType + " · " + fileSize(a.Size)
	if strings.EqualFold(a.MIMEType, "application/pdf") {
		description = "PDF document · " + fileSize(a.Size)
	}
	metadata := widget.NewLabel(description)
	metadata.Truncation = fyne.TextTruncateEllipsis
	metadata.Importance = widget.LowImportance
	actions := container.New(layout.NewRowWrapLayoutWithCustomPadding(4, 4))
	for _, action := range []struct {
		name     string
		icon     fyne.Resource
		callback func()
	}{
		{"Open", theme.FolderOpenIcon(), open}, {"Reveal", theme.FolderIcon(), reveal},
		{"Go to Memo", theme.DocumentIcon(), goMemo}, {"Delete", theme.DeleteIcon(), remove},
	} {
		button := widget.NewButtonWithIcon(action.name, action.icon, action.callback)
		button.Importance = widget.LowImportance
		if action.callback == nil {
			button.Disable()
		}
		actions.Add(button)
	}
	content := container.New(layout.NewCustomPaddedVBoxLayout(6), name, metadata, actions)
	return Surface(container.NewBorder(nil, nil, attachmentPreview(thumbnail, true), nil, content))
}
