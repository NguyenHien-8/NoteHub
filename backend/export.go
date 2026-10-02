package backend

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"time"
)

const exportFormat = "timeline-notes-export"
const exportVersion = "1.0"

type exportManifest struct {
	Format      string    `json:"format"`
	Version     string    `json:"version"`
	ExportedAt  time.Time `json:"exportedAt"`
	MemoCount   int       `json:"memoCount"`
	AttachCount int       `json:"attachmentCount"`
}

type exportAttachment struct {
	UID         string    `json:"uid"`
	Filename    string    `json:"filename"`
	MIMEType    string    `json:"mimeType"`
	Size        int64     `json:"size"`
	SHA256      string    `json:"sha256"`
	CreatedAt   time.Time `json:"createdAt"`
	ArchivePath string    `json:"archivePath"`
}

type exportMemo struct {
	UID         string             `json:"uid"`
	Content     string             `json:"content"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
	Tags        []string           `json:"tags"`
	Attachments []exportAttachment `json:"attachments,omitempty"`
}

func (s *Service) Export(ctx context.Context, zipPath string) error {
	var memos []Memo
	for offset := 0; ; offset += 500 {
		page, err := s.ListTimeline(ctx, TimelineQuery{Limit: 500, Offset: offset})
		if err != nil {
			return err
		}
		memos = append(memos, page...)
		if len(page) < 500 {
			break
		}
	}
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(zipPath)
		}
	}()
	zw := zip.NewWriter(f)
	attachCount := 0
	records := make([]exportMemo, 0, len(memos))
	for _, m := range memos {
		rec := exportMemo{UID: m.UID, Content: m.Content, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt, Tags: m.Tags}
		for _, a := range m.Attachments {
			archivePath := path.Join("attachments", m.UID, a.UID+path.Ext(a.Filename))
			w, err := zw.Create(archivePath)
			if err != nil {
				_ = zw.Close()
				return err
			}
			if err := s.copyAttachmentTo(ctx, a, w); err != nil {
				_ = zw.Close()
				return err
			}
			rec.Attachments = append(rec.Attachments, exportAttachment{UID: a.UID, Filename: a.Filename, MIMEType: a.MIMEType, Size: a.Size, SHA256: a.SHA256, CreatedAt: a.CreatedAt, ArchivePath: archivePath})
			attachCount++
		}
		records = append(records, rec)
	}
	manifest := exportManifest{Format: exportFormat, Version: exportVersion, ExportedAt: time.Now().UTC(), MemoCount: len(records), AttachCount: attachCount}
	if err := writeJSONZip(zw, "manifest.json", manifest); err != nil {
		_ = zw.Close()
		return err
	}
	if err := writeJSONZip(zw, "memos.json", records); err != nil {
		_ = zw.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	ok = true
	return nil
}

func writeJSONZip(zw *zip.Writer, name string, v any) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	return nil
}

func readAllLimit(r io.Reader, limit int64) ([]byte, error) {
	lr := io.LimitReader(r, limit+1)
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("entry exceeds %d bytes", limit)
	}
	return b, nil
}
