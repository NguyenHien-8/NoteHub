package dialogs

import (
	"context"

	"fyne.io/fyne/v2/dialog"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

func (m *Manager) DeleteMemo(note domain.Memo) {
	dialog.ShowConfirm("Delete note?", "This note and its local attachments will be deleted.", func(ok bool) {
		if !ok {
			return
		}
		work.Run(m.Jobs, func(ctx context.Context) (struct{}, error) { return struct{}{}, m.Backend.Memos.Delete(ctx, note.ID) }, func(_ struct{}, err error) {
			if err != nil {
				m.Error(err)
				return
			}
			m.Changed()
		})
	}, m.Window)
}

func (m *Manager) Favorite(note domain.Memo) {
	work.Run(m.Jobs, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, m.Backend.Memos.SetFavorite(ctx, note.ID, !note.Favorite)
	}, func(_ struct{}, err error) {
		if err != nil {
			m.Error(err)
			return
		}
		m.Changed()
	})
}

// Reusable confirmation dialog.
