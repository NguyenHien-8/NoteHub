package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/NguyenHien-8/NoteHub/internal/platform"
	"github.com/NguyenHien-8/NoteHub/internal/repository/sqlite"
	"github.com/NguyenHien-8/NoteHub/internal/service"
	"github.com/NguyenHien-8/NoteHub/internal/storage"
)

type Config struct {
	DataDir            string
	AppName            string
	Version            string
	MaxMemoBytes       int
	MaxAttachmentBytes int64
}

func OpenBackend(ctx context.Context, cfg Config) (*Backend, error) {
	var paths platform.DataPaths
	var err error
	if cfg.DataDir == "" {
		paths, err = platform.ResolveDataPaths(cfg.AppName)
	} else {
		var root string
		root, err = filepath.Abs(cfg.DataDir)
		if err == nil {
			paths = platform.DataPaths{Root: root, Database: filepath.Join(root,"notehub.db"), Attachments: filepath.Join(root,"attachments")}
			err = os.MkdirAll(paths.Attachments,0o755)
		}
	}
	if err != nil {
		return nil, err
	}
	store, err := sqlite.Open(ctx, paths.Database)
	if err != nil {
		return nil, err
	}
	files, err := storage.NewLocalStore(paths.Attachments, cfg.MaxAttachmentBytes)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	opts := service.Options{MaxMemoBytes: cfg.MaxMemoBytes, MaxAttachmentBytes: cfg.MaxAttachmentBytes}
	timeline := service.NewTimelineService(store, store, store)
	backend := &Backend{
		Paths: paths, Store: store,
		Memos:       service.NewMemoService(store, store, store, files, opts),
		Attachments: service.NewAttachmentService(store, store, files),
		Timeline:    timeline,
		Calendar:    service.NewCalendarService(store, timeline),
		Tags:        service.NewTagService(store),
		Search:      service.NewSearchService(store, store, store),
		Shares:      service.NewShareService(store, store, store, store, files),
		Backup:      service.NewBackupService(store, store, store, files, opts, cfg.Version),
	}
	return backend, nil
}
