package dialogs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/share"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

func (m *Manager) ShareMemo(note domain.Memo) {
	expiry := widget.NewSelect([]string{"Never expires", "1 day", "7 days", "30 days"}, nil)
	expiry.SetSelected("7 days")
	info := widget.NewLabel("Tokens are shown only once. Enable the local share server in Settings to use a link on this computer.")
	info.Wrapping = fyne.TextWrapWord
	links := container.NewVBox()
	grants := container.NewVBox()
	closed := false
	var refresh func()
	var create *widget.Button
	create = widget.NewButton("Create share", func() {
		var until *time.Time
		days := map[string]int{"1 day": 1, "7 days": 7, "30 days": 30}[expiry.Selected]
		if days > 0 {
			t := time.Now().AddDate(0, 0, days)
			until = &t
		}
		create.Disable()
		work.Run(m.Jobs, func(ctx context.Context) (*domain.ShareGrant, error) {
			return m.Backend.Shares.Create(ctx, note.ID, until)
		}, func(grant *domain.ShareGrant, err error) {
			if closed {
				return
			}
			create.Enable()
			if err != nil {
				m.Error(err)
				return
			}
			token := grant.Token
			value := token
			label := "New token (copy now)"
			if base := m.SharingURL(); base != "" {
				value, _ = share.URL(base, token)
				label = "Local share link (copy now)"
			}
			entry := widget.NewEntry()
			entry.SetText(value)
			copy := widget.NewButton("Copy", func() { m.Window.Clipboard().SetContent(value) })
			links.Objects = []fyne.CanvasObject{widget.NewLabel(label), entry, copy}
			links.Refresh()
			m.Changed()
			refresh()
		})
	})
	create.Importance = widget.HighImportance
	refresh = func() {
		work.Run(m.Jobs, func(ctx context.Context) ([]domain.Share, error) { return m.Backend.Shares.List(ctx, note.ID) }, func(items []domain.Share, err error) {
			if closed {
				return
			}
			if err != nil {
				m.Error(err)
				return
			}
			grants.RemoveAll()
			if len(items) == 0 {
				grants.Add(widget.NewLabel("No shares yet"))
			}
			for _, item := range items {
				expires := "Never expires"
				if item.ExpiresAt != nil {
					expires = item.ExpiresAt.Local().Format("02 Jan 2006 15:04")
					if !time.Now().Before(*item.ExpiresAt) {
						expires = "Expired · " + expires
					}
				}
				label := widget.NewLabel(fmt.Sprintf("%s… · %s", item.UID[:min(len(item.UID), 8)], expires))
				label.Truncation = fyne.TextTruncateEllipsis
				revoke := widget.NewButton("Revoke", func() {
					work.Run(m.Jobs, func(ctx context.Context) (struct{}, error) { return struct{}{}, m.Backend.Shares.Revoke(ctx, item.UID) }, func(_ struct{}, err error) {
						if closed {
							return
						}
						if err != nil {
							m.Error(err)
							return
						}
						links.RemoveAll()
						m.Changed()
						refresh()
					})
				})
				grants.Add(container.NewBorder(nil, nil, nil, revoke, label))
			}
		})
	}
	content := container.NewBorder(container.NewVBox(info, container.NewBorder(nil, nil, nil, create, expiry), links), nil, nil, nil, container.NewVScroll(grants))
	d := dialog.NewCustom("Share note", "Close", content, m.Window)
	d.Resize(fyne.NewSize(660, 500))
	d.SetOnClosed(func() { closed = true; links.RemoveAll() })
	d.Show()
	refresh()
}

// Human-readable import failures deliberately omit token or file payloads.
func failureSummary(failures []string) string { return strings.Join(failures, "\n") }

// Share link/token dialog.
