package app

import (
	"github.com/NguyenHien-8/NoteHub/internal/platform"
	"github.com/NguyenHien-8/NoteHub/internal/repository/sqlite"
	"github.com/NguyenHien-8/NoteHub/internal/service"
)

// Backend is the UI-independent composition root. Fyne (or any future GUI)
// should call this layer instead of opening SQLite directly.
type Backend struct {
	Paths       platform.DataPaths
	Store       *sqlite.Store
	Memos       *service.MemoService
	Attachments *service.AttachmentService
	Timeline    *service.TimelineService
	Calendar    *service.CalendarService
	Tags        *service.TagService
	Search      *service.SearchService
	Shares      *service.ShareService
	Backup      *service.BackupService
}
