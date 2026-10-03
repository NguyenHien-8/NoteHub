package backup

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	maxEntries        = 100_000
	maxManifestBytes  = 1 << 20
	maxRecordBytes    = 4 << 20
	maxContentBytes   = 8 << 20
	maxExpansionRatio = 64
	maxExpansionSlack = 64 << 20
)

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var mediaTypePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9!#$&^_.+-]{0,126}/[A-Za-z0-9][A-Za-z0-9!#$&^_.+-]{0,126}$`)
var reservedWindowsNames = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
	"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {}, "COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
	"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {}, "LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
}

// ValidateEntryName is adapted from Memos' export reader rules. It rejects
// traversal and cross-platform-dangerous names rather than trying to repair an
// untrusted archive.
func ValidateEntryName(name string) error {
	if name == "" {
		return errors.New("entry name is empty")
	}
	if !utf8.ValidString(name) {
		return fmt.Errorf("entry name %q is not valid UTF-8", name)
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("entry name %q is not a regular relative file", name)
	}
	for _, r := range name {
		if r < 0x20 || r == '\\' || r == ':' || r == 0x7f {
			return fmt.Errorf("entry name %q contains a forbidden character", name)
		}
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("entry name %q contains an invalid path segment", name)
		}
	}
	if path.Clean(name) != name {
		return fmt.Errorf("entry name %q is not canonical", name)
	}
	return nil
}

func SafeFilename(filename string) string {
	var b strings.Builder
	for _, r := range filename {
		switch {
		case r == '/' || r == '\\' || r == ':' || r < 0x20 || r == 0x7f || r == utf8.RuneError:
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	safe := strings.Trim(b.String(), " .")
	if safe == "" {
		return "file"
	}
	stem := safe
	if i := strings.IndexByte(stem, '.'); i >= 0 {
		stem = stem[:i]
	}
	if _, reserved := reservedWindowsNames[strings.ToUpper(stem)]; reserved {
		return "_" + safe
	}
	return safe
}

type Archive struct {
	Manifest Manifest
	Memos    []MemoRecord
	entries  map[string]*zip.File
}

func Read(r io.ReaderAt, size int64) (*Archive, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("not a ZIP file: %w", err)
	}
	if len(zr.File) > maxEntries {
		return nil, fmt.Errorf("archive has more than %d entries", maxEntries)
	}
	a := &Archive{entries: make(map[string]*zip.File, len(zr.File))}
	var compressed, uncompressed uint64
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") && f.UncompressedSize64 == 0 {
			continue
		}
		compressed += f.CompressedSize64
		uncompressed += f.UncompressedSize64
		if uncompressed > maxExpansionRatio*compressed+maxExpansionSlack {
			return nil, fmt.Errorf("archive expansion ratio is unsafe")
		}
		if err := ValidateEntryName(f.Name); err != nil {
			return nil, err
		}
		if f.Method != zip.Store && f.Method != zip.Deflate {
			return nil, fmt.Errorf("entry %q uses unsupported compression method", f.Name)
		}
		if f.Flags&0x1 != 0 {
			return nil, fmt.Errorf("entry %q is encrypted", f.Name)
		}
		if _, dup := a.entries[f.Name]; dup {
			return nil, fmt.Errorf("duplicate entry %q", f.Name)
		}
		a.entries[f.Name] = f
	}
	manifestBytes, err := a.readEntry(ManifestEntry, maxManifestBytes)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(manifestBytes, &a.Manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest JSON: %w", err)
	}
	if err := validateManifest(a.Manifest); err != nil {
		return nil, err
	}
	if err := a.readMemos(); err != nil {
		return nil, err
	}
	return a, nil
}

func validateManifest(m Manifest) error {
	if m.Format != Format {
		return fmt.Errorf("unsupported format %q", m.Format)
	}
	major, _, err := parseFormatVersion(m.FormatVersion)
	if err != nil {
		return err
	}
	if major != 1 {
		return fmt.Errorf("unsupported format version %s", m.FormatVersion)
	}
	if m.Generator.Name == "" || m.Generator.Version == "" {
		return errors.New("manifest generator is required")
	}
	if _, err := ParseTime(m.ExportTime); err != nil {
		return fmt.Errorf("invalid exportTime: %w", err)
	}
	if m.Counts.Memos < 0 || m.Counts.Attachments < 0 {
		return errors.New("manifest counts cannot be negative")
	}
	return nil
}

func (a *Archive) readMemos() error {
	records := make(map[string]MemoRecord)
	contents := make(map[string]struct{})
	for _, name := range slices.Sorted(maps.Keys(a.entries)) {
		rest, ok := strings.CutPrefix(name, MemosDir)
		if !ok || strings.Contains(rest, "/") {
			continue
		}
		if uid, ok := strings.CutSuffix(rest, ".md"); ok {
			contents[uid] = struct{}{}
			continue
		}
		uid, ok := strings.CutSuffix(rest, ".json")
		if !ok {
			continue
		}
		b, err := a.readEntry(name, maxRecordBytes)
		if err != nil {
			return err
		}
		var rec MemoRecord
		if err := json.Unmarshal(b, &rec); err != nil {
			return fmt.Errorf("%s is not valid JSON: %w", name, err)
		}
		if rec.UID != uid {
			return fmt.Errorf("%s uid does not match filename", name)
		}
		if err := validateMemoRecord(rec); err != nil {
			return err
		}
		if _, ok := a.entries[rec.ContentPath]; !ok {
			return fmt.Errorf("memo %s content is missing", rec.UID)
		}
		for _, at := range rec.Attachments {
			if _, ok := a.entries[at.Path]; !ok {
				return fmt.Errorf("memo %s attachment %s is missing", rec.UID, at.UID)
			}
		}
		if _, dup := records[uid]; dup {
			return fmt.Errorf("duplicate memo uid %s", uid)
		}
		records[uid] = rec
	}
	for uid := range contents {
		if _, ok := records[uid]; !ok {
			return fmt.Errorf("content for memo %s has no record", uid)
		}
	}
	if len(records) != a.Manifest.Counts.Memos {
		return fmt.Errorf("manifest memo count %d does not match archive count %d", a.Manifest.Counts.Memos, len(records))
	}
	attachCount := 0
	for _, r := range records {
		attachCount += len(r.Attachments)
	}
	if attachCount != a.Manifest.Counts.Attachments {
		return fmt.Errorf("manifest attachment count %d does not match archive count %d", a.Manifest.Counts.Attachments, attachCount)
	}
	a.Memos = make([]MemoRecord, 0, len(records))
	for _, r := range records {
		a.Memos = append(a.Memos, r)
	}
	slices.SortFunc(a.Memos, func(x, y MemoRecord) int {
		xT, _ := ParseTime(x.CreateTime)
		yT, _ := ParseTime(y.CreateTime)
		if xT.Before(yT) {
			return -1
		}
		if xT.After(yT) {
			return 1
		}
		return strings.Compare(x.UID, y.UID)
	})
	return nil
}

func validateMemoRecord(rec MemoRecord) error {
	if err := validateUID(rec.UID); err != nil {
		return err
	}
	if _, err := ParseTime(rec.CreateTime); err != nil {
		return fmt.Errorf("memo %s invalid createTime", rec.UID)
	}
	if _, err := ParseTime(rec.UpdateTime); err != nil {
		return fmt.Errorf("memo %s invalid updateTime", rec.UID)
	}
	if rec.ContentPath != ContentPath(rec.UID) {
		return fmt.Errorf("memo %s has unexpected contentPath", rec.UID)
	}
	seen := make(map[string]struct{})
	for _, at := range rec.Attachments {
		if err := validateUID(at.UID); err != nil {
			return fmt.Errorf("memo %s attachment: %w", rec.UID, err)
		}
		if _, dup := seen[at.UID]; dup {
			return fmt.Errorf("memo %s lists attachment %s twice", rec.UID, at.UID)
		}
		seen[at.UID] = struct{}{}
		// DetectContentType and TypeByExtension legitimately include parameters
		// such as charset=utf-8. Accept them in existing backups, while rejecting
		// malformed MIME values and header injection.
		mediaType, _, typeErr := mime.ParseMediaType(at.Type)
		if at.Filename == "" || typeErr != nil || strings.ContainsAny(at.Type, "\r\n") || !mediaTypePattern.MatchString(mediaType) || at.Size < 0 || !sha256Pattern.MatchString(at.SHA256) {
			return fmt.Errorf("memo %s has invalid attachment metadata", rec.UID)
		}
		if _, err := ParseTime(at.CreateTime); err != nil {
			return fmt.Errorf("memo %s attachment %s invalid createTime", rec.UID, at.UID)
		}
		if at.Path != AttachmentPath(at.UID, at.Filename) {
			return fmt.Errorf("memo %s attachment %s has unexpected path", rec.UID, at.UID)
		}
		if err := ValidateEntryName(at.Path); err != nil {
			return err
		}
	}
	return nil
}

func (a *Archive) Content(rec MemoRecord) ([]byte, error) {
	return a.readEntry(rec.ContentPath, maxContentBytes)
}

// OpenAttachment streams an attachment entry after checking its declared size.
// The caller must compare the digest produced while storing the stream with
// rec.SHA256; this avoids holding large media files in memory.
func (a *Archive) OpenAttachment(rec AttachmentRecord, maxBytes int64) (io.ReadCloser, error) {
	if maxBytes <= 0 {
		maxBytes = 1 << 30
	}
	if rec.Size > maxBytes {
		return nil, fmt.Errorf("attachment %s exceeds configured limit", rec.UID)
	}
	f, ok := a.entries[rec.Path]
	if !ok {
		return nil, fmt.Errorf("attachment %s missing", rec.UID)
	}
	if f.UncompressedSize64 != uint64(rec.Size) {
		return nil, fmt.Errorf("attachment %s size mismatch", rec.UID)
	}
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (a *Archive) Attachment(rec AttachmentRecord, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = 1 << 30
	}
	if rec.Size > maxBytes {
		return nil, fmt.Errorf("attachment %s exceeds configured limit", rec.UID)
	}
	f, ok := a.entries[rec.Path]
	if !ok {
		return nil, fmt.Errorf("attachment %s missing", rec.UID)
	}
	if f.UncompressedSize64 != uint64(rec.Size) {
		return nil, fmt.Errorf("attachment %s size mismatch", rec.UID)
	}
	b, err := a.readEntry(rec.Path, maxBytes)
	if err != nil {
		return nil, err
	}
	d := sha256.Sum256(b)
	if hex.EncodeToString(d[:]) != rec.SHA256 {
		return nil, fmt.Errorf("attachment %s SHA-256 mismatch", rec.UID)
	}
	return b, nil
}

func (a *Archive) readEntry(name string, limit int64) ([]byte, error) {
	f, ok := a.entries[name]
	if !ok {
		return nil, fmt.Errorf("entry %q is missing", name)
	}
	if f.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("entry %q is larger than %d bytes", name, limit)
	}
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var b bytes.Buffer
	if f.UncompressedSize64 < uint64(^uint(0)>>1) {
		b.Grow(int(f.UncompressedSize64))
	}
	n, err := io.Copy(&b, io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if n > limit {
		return nil, fmt.Errorf("entry %q exceeds limit", name)
	}
	return b.Bytes(), nil
}
