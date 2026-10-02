package dialogs

import (
	"context"
	"strings"
	"time"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/platform"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

func (m *Manager) Export() {
	if m.backupBusy {
		return
	}
	m.backupBusy = true
	work.Run(m.Jobs, func(ctx context.Context) (string, error) {
		return platform.ChooseBackupDestination(ctx, "NoteHub-"+time.Now().Format("20060102-150405")+".zip")
	}, func(path string, err error) {
		if err != nil {
			m.backupBusy = false
			if !platform.DialogCanceled(err) {
				m.Error(err)
			}
			return
		}
		if !strings.HasSuffix(strings.ToLower(path), ".zip") {
			path += ".zip"
		}
		progress := widget.NewProgressBarInfinite()
		d := dialog.NewCustomWithoutButtons("Exporting backup", container.NewVBox(widget.NewLabel("Copying notes and verifying attachment checksums…"), progress), m.Window)
		d.Show()
		work.Run(m.Jobs, func(ctx context.Context) (struct{}, error) { return struct{}{}, m.Backend.Backup.ExportFile(ctx, path) }, func(_ struct{}, err error) {
			m.backupBusy = false
			progress.Stop()
			d.Hide()
			if err != nil {
				m.Error(err)
				return
			}
			dialog.ShowInformation("Backup exported", path, m.Window)
		})
	})
}

// Export backup dialog.
