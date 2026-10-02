package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrAttachmentTooLarge = errors.New("attachment exceeds configured size limit")

type SavedFile struct {
	RelativePath string
	Size         int64
	SHA256       string
}

// LocalStore stores attachment bytes outside SQLite. Writes go to a temporary
// file first and are fsync+renamed atomically, mirroring Memos' separation of
// metadata from storage while making desktop crashes less likely to corrupt a
// file being written.
type LocalStore struct {
	root     string
	maxBytes int64
}

func NewLocalStore(root string, maxBytes int64) (*LocalStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("attachment root is required")
	}
	if maxBytes <= 0 {
		maxBytes = 1 << 30 // 1 GiB default per attachment.
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &LocalStore{root: root, maxBytes: maxBytes}, nil
}

func (s *LocalStore) Root() string { return s.root }

func (s *LocalStore) Save(memoUID, attachmentUID, filename string, r io.Reader) (SavedFile, error) {
	if memoUID == "" || attachmentUID == "" {
		return SavedFile{}, errors.New("memo uid and attachment uid are required")
	}
	relative := filepath.Join(memoUID, attachmentUID+safeExtension(filename))
	full, err := ensureInside(s.root, relative)
	if err != nil {
		return SavedFile{}, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return SavedFile{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(full), ".notehub-upload-*")
	if err != nil {
		return SavedFile{}, err
	}
	tmpName := tmp.Name()
	keep := false
	defer func() {
		_ = tmp.Close()
		if !keep {
			_ = os.Remove(tmpName)
		}
	}()

	h := sha256.New()
	limited := io.LimitReader(r, s.maxBytes+1)
	n, err := io.Copy(io.MultiWriter(tmp, h), limited)
	if err != nil {
		return SavedFile{}, err
	}
	if n > s.maxBytes {
		return SavedFile{}, ErrAttachmentTooLarge
	}
	if err := tmp.Sync(); err != nil {
		return SavedFile{}, err
	}
	if err := tmp.Close(); err != nil {
		return SavedFile{}, err
	}
	// UIDs are unique, so an existing path indicates a stale/orphan file. Do
	// not overwrite it silently.
	if _, err := os.Stat(full); err == nil {
		return SavedFile{}, fmt.Errorf("attachment target already exists: %s", filepath.Base(full))
	} else if !os.IsNotExist(err) {
		return SavedFile{}, err
	}
	if err := os.Rename(tmpName, full); err != nil {
		return SavedFile{}, err
	}
	keep = true
	return SavedFile{
		RelativePath: filepath.ToSlash(relative),
		Size:         n,
		SHA256:       hex.EncodeToString(h.Sum(nil)),
	}, nil
}

func (s *LocalStore) Open(relative string) (*os.File, error) {
	full, err := ensureInside(s.root, filepath.FromSlash(relative))
	if err != nil {
		return nil, err
	}
	return os.Open(full)
}

func (s *LocalStore) Path(relative string) (string, error) {
	return ensureInside(s.root, filepath.FromSlash(relative))
}

func (s *LocalStore) Delete(relative string) error {
	full, err := ensureInside(s.root, filepath.FromSlash(relative))
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Best-effort cleanup of now-empty memo directory.
	_ = os.Remove(filepath.Dir(full))
	return nil
}
