package backup

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Writer struct {
	zip      *zip.Writer
	seen     map[string]struct{}
	modified time.Time
	manifest bool
}

func NewWriter(w io.Writer, modified time.Time) *Writer {
	return &Writer{zip: zip.NewWriter(w), seen: make(map[string]struct{}), modified: modified.UTC().Truncate(time.Second)}
}

func (w *Writer) WriteManifest(m Manifest) error {
	if w.manifest {
		return fmt.Errorf("manifest already written")
	}
	if err := w.writeJSON(ManifestEntry, m); err != nil {
		return err
	}
	w.manifest = true
	return nil
}

func (w *Writer) WriteAttachment(name string, r io.Reader) (string, int64, error) {
	if !w.manifest {
		return "", 0, fmt.Errorf("manifest must be written first")
	}
	entry, err := w.create(name, zip.Deflate)
	if err != nil {
		return "", 0, err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(entry, h), r)
	if err != nil {
		return "", n, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func (w *Writer) WriteMemo(rec MemoRecord, content []byte) error {
	if !w.manifest {
		return fmt.Errorf("manifest must be written first")
	}
	cw, err := w.create(rec.ContentPath, zip.Deflate)
	if err != nil {
		return err
	}
	if _, err := cw.Write(content); err != nil {
		return err
	}
	return w.writeJSON(RecordPath(rec.UID), rec)
}

func (w *Writer) Close() error {
	if !w.manifest {
		return fmt.Errorf("manifest was not written")
	}
	return w.zip.Close()
}

func (w *Writer) writeJSON(name string, value any) error {
	entry, err := w.create(name, zip.Deflate)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(entry)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func (w *Writer) create(name string, method uint16) (io.Writer, error) {
	if err := ValidateEntryName(name); err != nil {
		return nil, err
	}
	if _, dup := w.seen[name]; dup {
		return nil, fmt.Errorf("duplicate archive entry %q", name)
	}
	w.seen[name] = struct{}{}
	return w.zip.CreateHeader(&zip.FileHeader{Name: name, Method: method, Modified: w.modified})
}
