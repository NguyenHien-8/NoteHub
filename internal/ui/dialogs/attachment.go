package dialogs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2/dialog"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/platform"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

func (m *Manager) PickFiles(done func([]string)) {
	work.Run(m.Jobs, platform.ChooseAttachments, func(paths []string, err error) {
		if platform.DialogCanceled(err) {
			return
		}
		if err != nil {
			m.Error(err)
			return
		}
		done(paths)
	})
}

func (m *Manager) OpenAttachment(a domain.Attachment) {
	work.Run(m.Jobs, func(ctx context.Context) (struct{}, error) {
		path, err := m.Backend.Attachments.Path(ctx, a.ID)
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, platform.OpenExternal(path)
	}, func(_ struct{}, err error) { m.Error(err) })
}

func (m *Manager) AttachToMemo(id int64, done func()) {
	m.PickFiles(func(paths []string) {
		work.Run(m.Jobs, func(ctx context.Context) ([]string, error) {
			var failures []string
			for _, path := range paths {
				if _, err := m.Backend.Attachments.AddFromFile(ctx, id, path); err != nil {
					failures = append(failures, fmt.Sprintf("%s: %v", filepath.Base(path), err))
				}
			}
			return failures, nil
		}, func(failures []string, err error) {
			m.Error(err)
			m.Changed()
			if done != nil {
				done()
			}
			if len(failures) > 0 {
				dialog.ShowInformation("Some files could not be attached", strings.Join(failures, "\n"), m.Window)
			}
		})
	})
}

// Attachment selection/details dialog.
