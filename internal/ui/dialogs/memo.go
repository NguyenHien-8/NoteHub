package dialogs

import (
	"context"
	"errors"
	"fmt"
	"image"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/ui/components"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

type memoDialogData struct {
	note   *domain.Memo
	thumbs map[int64]image.Image
}

func (m *Manager) loadMemoDialogData(ctx context.Context, id int64) (memoDialogData, error) {
	note, err := m.Backend.Memos.Get(ctx, id)
	if err != nil {
		return memoDialogData{}, err
	}
	data := memoDialogData{note: note, thumbs: make(map[int64]image.Image)}
	if m.preview == nil {
		return data, nil
	}
	for _, attachment := range note.Attachments {
		if ctx.Err() != nil {
			return memoDialogData{}, ctx.Err()
		}
		if strings.HasPrefix(attachment.MIMEType, "image/") {
			if preview := m.preview(ctx, attachment); preview != nil {
				data.thumbs[attachment.ID] = preview
			}
		}
	}
	return data, nil
}

func (m *Manager) ShowMemo(id int64) {
	work.Run(m.Jobs, func(ctx context.Context) (memoDialogData, error) { return m.loadMemoDialogData(ctx, id) }, func(data memoDialogData, err error) {
		if err != nil {
			m.Error(err)
			return
		}
		note := data.note
		text := widget.NewRichTextWithText(note.Content)
		text.Wrapping = fyne.TextWrapWord
		content := container.NewVBox(text)
		if countImages(note.Attachments) > 0 {
			gallery := components.NewImageGallery(note.Attachments, data.thumbs, m.OpenAttachment, nil)
			content.Add(gallery)
		}
		for _, attachment := range note.Attachments {
			attachment := attachment
			if strings.HasPrefix(attachment.MIMEType, "image/") {
				continue
			}
			button := widget.NewButtonWithIcon(attachment.Filename, theme.FileIcon(), func() { m.OpenAttachment(attachment) })
			button.Alignment = widget.ButtonAlignLeading
			button.Importance = widget.LowImportance
			content.Add(button)
		}
		var d *dialog.CustomDialog
		edit := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() { d.Hide(); m.EditMemo(*note) })
		attach := widget.NewButtonWithIcon("Attach files", theme.MailAttachmentIcon(), func() { m.AttachToMemo(id, func() { d.Hide(); m.ShowMemo(id) }) })
		share := widget.NewButtonWithIcon("Share", theme.MailForwardIcon(), func() { m.ShareMemo(*note) })
		body := container.NewBorder(container.NewHBox(edit, attach, share), nil, nil, nil, container.NewVScroll(content))
		d = dialog.NewCustom("Note · "+note.CreatedAt.Local().Format("02 Jan 2006 15:04"), "Close", body, m.Window)
		d.Resize(fitDialogSize(m.Window, fyne.NewSize(700, 520)))
		d.Show()
	})
}

func (m *Manager) EditMemo(note domain.Memo) {
	work.Run(m.Jobs, func(ctx context.Context) (memoDialogData, error) { return m.loadMemoDialogData(ctx, note.ID) }, func(data memoDialogData, err error) {
		if err != nil {
			m.Error(err)
			return
		}
		current := data.note
		thumbs := data.thumbs
		entry := widget.NewMultiLineEntry()
		entry.Wrapping = fyne.TextWrapWord
		entry.SetText(current.Content)
		m.drafts[entry] = current.Content
		status := widget.NewLabel("")
		status.Wrapping = fyne.TextWrapWord
		status.Importance = widget.LowImportance

		var d *dialog.CustomDialog
		var save, reload, cancel *widget.Button
		busy := false
		attachmentBusy := false
		closed := false

		var gallery *components.ImageGallery
		gallery = components.NewImageGallery(current.Attachments, thumbs, m.OpenAttachment, func(ids []int64) {
			if busy || attachmentBusy || closed {
				return
			}
			attachmentBusy = true
			status.SetText("Saving image order…")
			orderedIDs := append([]int64(nil), ids...)
			memoID := current.ID
			work.Run(m.Jobs, func(ctx context.Context) (struct{}, error) {
				return struct{}{}, m.Backend.Attachments.Reorder(ctx, memoID, orderedIDs)
			}, func(_ struct{}, err error) {
				attachmentBusy = false
				if closed {
					return
				}
				if err != nil {
					status.SetText("Image order was not changed.")
					m.Error(err)
					return
				}
				current.Attachments = reorderAttachmentSnapshot(current.Attachments, orderedIDs)
				gallery.SetAttachments(current.Attachments, thumbs)
				status.SetText("Image order saved.")
				if m.Changed != nil {
					m.Changed()
				}
			})
		})
		galleryLabel := widget.NewLabel("Images · drag to reorder · delete to remove")
		galleryLabel.Importance = widget.LowImportance
		galleryScroll := container.NewVScroll(gallery)
		galleryScroll.SetMinSize(fyne.NewSize(0, 150))
		gallerySection := container.NewBorder(galleryLabel, nil, nil, nil, galleryScroll)
		refreshGalleryVisibility := func() {
			if countImages(current.Attachments) == 0 {
				gallerySection.Hide()
			} else {
				gallerySection.Show()
			}
		}
		gallery.SetDelete(func(attachment domain.Attachment) {
			if busy || attachmentBusy || closed {
				return
			}
			dialog.ShowConfirm("Delete image?", "Remove "+attachment.Filename+" from this note now?", func(ok bool) {
				if !ok || busy || attachmentBusy || closed {
					return
				}
				attachmentBusy = true
				status.SetText("Deleting image…")
				work.Run(m.Jobs, func(ctx context.Context) (struct{}, error) {
					return struct{}{}, m.Backend.Attachments.Delete(ctx, attachment.ID)
				}, func(_ struct{}, err error) {
					attachmentBusy = false
					if closed {
						return
					}
					if err != nil {
						status.SetText("Image could not be deleted.")
						m.Error(err)
						return
					}
					current.Attachments = removeAttachment(current.Attachments, attachment.ID)
					delete(thumbs, attachment.ID)
					gallery.SetAttachments(current.Attachments, thumbs)
					refreshGalleryVisibility()
					status.SetText("Image deleted.")
					if m.Changed != nil {
						m.Changed()
					}
				})
			}, m.Window)
		})
		refreshGalleryVisibility()

		cancel = widget.NewButton("Cancel", func() {
			if busy || attachmentBusy || closed {
				return
			}
			if entry.Text == m.drafts[entry] {
				d.Hide()
				return
			}
			dialog.ShowConfirm("Discard edits?", "Your unsaved text changes will be lost. Attachment changes that were already confirmed stay saved.", func(ok bool) {
				if ok {
					d.Hide()
				}
			}, m.Window)
		})
		save = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
			if busy || attachmentBusy || closed {
				return
			}
			busy = true
			save.Disable()
			reload.Disable()
			cancel.Disable()
			entry.Disable()
			content, revision, id := entry.Text, current.Revision, current.ID
			work.Run(m.Jobs, func(ctx context.Context) (*domain.Memo, error) {
				return m.Backend.Memos.Update(ctx, id, revision, content)
			}, func(updated *domain.Memo, err error) {
				if closed {
					if err == nil && m.Changed != nil {
						m.Changed()
					}
					return
				}
				busy = false
				save.Enable()
				reload.Enable()
				cancel.Enable()
				entry.Enable()
				if errors.Is(err, domain.ErrConflict) {
					status.SetText("Ghi chú đã được thay đổi ở nơi khác. Vui lòng tải lại. Your edits are kept here; copy them before reloading.")
					return
				}
				if err != nil {
					m.Error(err)
					return
				}
				d.Hide()
				if m.Changed != nil {
					m.Changed()
				}
			})
		})
		save.Importance = widget.HighImportance
		reload = widget.NewButton("Reload latest", func() {
			if busy || attachmentBusy || closed {
				return
			}
			dialog.ShowConfirm("Reload note?", "Copy any text edits you want to keep before replacing this editor with the latest saved note.", func(ok bool) {
				if !ok || busy || attachmentBusy || closed {
					return
				}
				busy = true
				save.Disable()
				reload.Disable()
				cancel.Disable()
				entry.Disable()
				id := current.ID
				work.Run(m.Jobs, func(ctx context.Context) (memoDialogData, error) { return m.loadMemoDialogData(ctx, id) }, func(latest memoDialogData, err error) {
					if closed {
						return
					}
					busy = false
					save.Enable()
					reload.Enable()
					cancel.Enable()
					entry.Enable()
					if err != nil {
						m.Error(err)
						return
					}
					current = latest.note
					thumbs = latest.thumbs
					entry.SetText(current.Content)
					m.drafts[entry] = current.Content
					gallery.SetAttachments(current.Attachments, thumbs)
					refreshGalleryVisibility()
					status.SetText(fmt.Sprintf("Loaded revision %d", current.Revision))
				})
			}, m.Window)
		})

		center := container.NewBorder(nil, gallerySection, nil, nil, entry)
		body := container.NewBorder(nil, container.NewVBox(status, container.NewHBox(reload, cancel, save)), nil, nil, center)
		d = dialog.NewCustomWithoutButtons("Edit note", body, m.Window)
		d.SetOnClosed(func() { closed = true; delete(m.drafts, entry) })
		d.Resize(fitDialogSize(m.Window, fyne.NewSize(720, 600)))
		d.Show()
		m.Window.Canvas().Focus(entry)
	})
}

func (m *Manager) PickTag(done func(string)) {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Research/FPGA")
	work.Run(m.Jobs, func(ctx context.Context) ([]domain.TagCount, error) { return m.Backend.Tags.List(ctx) }, func(tags []domain.TagCount, err error) {
		if err != nil {
			m.Error(err)
			return
		}
		options := make([]string, 0, len(tags))
		for _, tag := range tags {
			options = append(options, tag.Tag)
		}
		selectTag := widget.NewSelect(options, func(tag string) { entry.SetText(tag) })
		d := dialog.NewForm("Add tag", "Insert tag", "Cancel", []*widget.FormItem{widget.NewFormItem("Existing", selectTag), widget.NewFormItem("Tag", entry)}, func(ok bool) {
			if ok {
				done(entry.Text)
			}
		}, m.Window)
		d.Resize(fitDialogSize(m.Window, fyne.NewSize(420, 240)))
		d.Show()
	})
}

func countImages(attachments []domain.Attachment) int {
	count := 0
	for _, attachment := range attachments {
		if strings.HasPrefix(attachment.MIMEType, "image/") {
			count++
		}
	}
	return count
}

func removeAttachment(attachments []domain.Attachment, id int64) []domain.Attachment {
	out := make([]domain.Attachment, 0, len(attachments))
	for _, attachment := range attachments {
		if attachment.ID != id {
			out = append(out, attachment)
		}
	}
	return out
}

func reorderAttachmentSnapshot(attachments []domain.Attachment, ids []int64) []domain.Attachment {
	byID := make(map[int64]domain.Attachment, len(attachments))
	for _, attachment := range attachments {
		byID[attachment.ID] = attachment
	}
	out := make([]domain.Attachment, 0, len(ids))
	for position, id := range ids {
		attachment, ok := byID[id]
		if !ok {
			return append([]domain.Attachment(nil), attachments...)
		}
		attachment.Position = position
		out = append(out, attachment)
	}
	return out
}

func fitDialogSize(window fyne.Window, desired fyne.Size) fyne.Size {
	if window == nil || window.Canvas() == nil {
		return desired
	}
	available := window.Canvas().Size()
	if available.Width > 80 {
		desired.Width = min(desired.Width, available.Width-40)
	}
	if available.Height > 80 {
		desired.Height = min(desired.Height, available.Height-40)
	}
	return desired
}
