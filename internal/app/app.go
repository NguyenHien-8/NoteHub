package app

import (
	"github.com/NguyenHien-8/NoteHub/internal/platform"
	"github.com/NguyenHien-8/NoteHub/internal/repository/sqlite"
	"github.com/NguyenHien-8/NoteHub/internal/service"
	"github.com/NguyenHien-8/NoteHub/internal/storage"
)

// Backend is the UI-independent composition root. The Qt desktop frontend talks
// to this object indirectly through internal/ipc, so business logic and data
// access stay in Go while the GUI remains a separate native process.
type Backend struct {
	Paths       platform.DataPaths
	Store       *sqlite.Store
	Files       *storage.LocalStore
	Memos       *service.MemoService
	Attachments *service.AttachmentService
	Timeline    *service.TimelineService
	Calendar    *service.CalendarService
	Tags        *service.TagService
	Search      *service.SearchService
	Shares      *service.ShareService
	Backup      *service.BackupService
}
