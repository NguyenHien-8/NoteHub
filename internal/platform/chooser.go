package platform

import (
	"context"
	"errors"

	"github.com/ncruces/zenity"
)

var ErrDialogCanceled = zenity.ErrCanceled

func ChooseAttachments(ctx context.Context) ([]string, error) {
	return zenity.SelectFileMultiple(zenity.Title("Attach files to NoteHub"), zenity.Context(ctx))
}

func ChooseBackup(ctx context.Context) (string, error) {
	return zenity.SelectFile(zenity.Title("Import NoteHub backup"), zenity.FileFilter{Name: "NoteHub backup", Patterns: []string{"*.zip"}}, zenity.Context(ctx))
}

// Export never overwrites an existing backup; the service enforces this too.
func ChooseBackupDestination(ctx context.Context, filename string) (string, error) {
	return zenity.SelectFileSave(zenity.Title("Export to a new backup file"), zenity.Filename(filename), zenity.FileFilter{Name: "ZIP backup", Patterns: []string{"*.zip"}}, zenity.Context(ctx))
}

func DialogCanceled(err error) bool {
	return errors.Is(err, ErrDialogCanceled) || errors.Is(err, context.Canceled)
}
