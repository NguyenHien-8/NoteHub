package components

import (
	"image"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

type MemoActions struct {
	Open, Edit, Favorite, Share, Delete func(domain.Memo)
	Tag                                 func(string)
	Attachment                          func(domain.Attachment)
	Reorder                             func(domain.Memo, []int64)
}

func NewMemoCard(m domain.Memo, thumbnails map[int64]image.Image, actions MemoActions) fyne.CanvasObject {
	title, body := memoPreview(m.Content)
	if m.Favorite {
		title = "★ " + title
	}
	heading := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	heading.Truncation = fyne.TextTruncateEllipsis
	preview := widget.NewLabel(body)
	preview.Wrapping = fyne.TextWrapWord
	preview.Truncation = fyne.TextTruncateEllipsis
	if body == "" {
		preview.Hide()
	}
	stamp := widget.NewLabel(m.CreatedAt.In(time.Local).Format("15:04"))
	stamp.Importance = widget.LowImportance
	stamp.SizeName = theme.SizeNameCaptionText
	more := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
	more.Importance = widget.LowImportance
	more.OnTapped = func() {
		if app := fyne.CurrentApp(); app != nil && app.Driver() != nil {
			if c := app.Driver().CanvasForObject(more); c != nil {
				widget.ShowPopUpMenuAtRelativePosition(memoMenu(m, actions), c, fyne.NewPos(0, more.Size().Height), more)
			}
		}
	}

	tags := container.New(layout.NewRowWrapLayoutWithCustomPadding(6, 6))
	for _, tag := range m.Tags {
		tags.Add(tagButton(tag, actions.Tag))
	}
	if len(m.Tags) == 0 {
		tags.Hide()
	}

	files := container.NewVBox()
	imageCount := 0
	for _, attachment := range m.Attachments {
		attachment := attachment
		if strings.HasPrefix(attachment.MIMEType, "image/") {
			imageCount++
			continue
		}
		button := widget.NewButtonWithIcon(boundedText(attachment.Filename, 42)+" · "+fileSize(attachment.Size), theme.MailAttachmentIcon(), func() {
			if actions.Attachment != nil {
				actions.Attachment(attachment)
			}
		})
		button.Importance, button.Alignment = widget.LowImportance, widget.ButtonAlignLeading
		files.Add(button)
	}
	if len(files.Objects) == 0 {
		files.Hide()
	}

	var bodyObject fyne.CanvasObject = preview
	if strings.Contains(body, "\n") || len([]rune(body)) > 80 {
		space := canvas.NewRectangle(color.Transparent)
		space.SetMinSize(fyne.NewSize(0, 52))
		bodyObject = container.NewStack(space, preview)
	}
	open := &tapContent{content: container.NewVBox(heading, bodyObject), tapped: func() {
		if actions.Open != nil {
			actions.Open(m)
		}
	}}
	open.ExtendBaseWidget(open)

	content := []fyne.CanvasObject{container.NewBorder(nil, nil, nil, container.NewHBox(stamp, more), open)}
	if imageCount > 0 {
		var reorder func([]int64)
		if actions.Reorder != nil {
			reorder = func(ids []int64) { actions.Reorder(m, ids) }
		}
		gallery := NewImageGallery(m.Attachments, thumbnails, actions.Attachment, reorder)
		content = append(content, gallery)
	}
	content = append(content, tags, files)
	return Surface(container.New(layout.NewCustomPaddedVBoxLayout(7), content...))
}

func memoMenu(m domain.Memo, actions MemoActions) *fyne.Menu {
	item := func(label string, action func(domain.Memo)) *fyne.MenuItem {
		entry := fyne.NewMenuItem(label, func() {
			if action != nil {
				action(m)
			}
		})
		entry.Disabled = action == nil
		return entry
	}
	favorite := "Favorite"
	if m.Favorite {
		favorite = "Remove favorite"
	}
	return fyne.NewMenu("", item("Edit", actions.Edit), item(favorite, actions.Favorite), item("Share", actions.Share), fyne.NewMenuItemSeparator(), item("Delete", actions.Delete))
}

type tapContent struct {
	widget.BaseWidget
	content fyne.CanvasObject
	tapped  func()
}

func (t *tapContent) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(t.content) }
func (t *tapContent) Tapped(*fyne.PointEvent) {
	if t.tapped != nil {
		t.tapped()
	}
}
