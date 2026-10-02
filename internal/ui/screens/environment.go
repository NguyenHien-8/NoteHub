package screens

import (
 "context"
 "errors"

 "fyne.io/fyne/v2"
 "fyne.io/fyne/v2/dialog"
 "github.com/NguyenHien-8/NoteHub/internal/app"
 "github.com/NguyenHien-8/NoteHub/internal/ui/components"
 "github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

// Environment supplies services and navigation; screens never own persistence.
type Environment struct {
 Backend *app.Backend
 Window fyne.Window
 Jobs *work.Runner
 Previews *Previews
 Actions components.MemoActions
 Changed func()
 Tag func(string)
 Memo func(int64)
 PickFiles func(func([]string))
 PickTag func(func(string))
}

func (e *Environment) Error(err error) {
 if err!=nil && !errors.Is(err,context.Canceled) {dialog.ShowError(err,e.Window)}
}
