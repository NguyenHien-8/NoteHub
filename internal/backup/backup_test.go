package backup

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"
)

func TestWriterReaderRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	now := time.Date(2026, 10, 2, 6, 0, 0, 0, time.UTC)
	w := NewWriter(&buf, now)
	if err := w.WriteManifest(Manifest{Format: Format, FormatVersion: FormatVersion, Generator: Generator{Name: "NoteHub", Version: "test"}, ExportTime: FormatTime(now), Counts: Counts{Memos: 1, Attachments: 1}}); err != nil {
		t.Fatal(err)
	}
	uid := "0123456789abcdef0123456789abcdef"
	attUID := "abcdef0123456789abcdef0123456789"
	payload := []byte("hello attachment")
	attPath := AttachmentPath(attUID, "test.txt")
	digest, size, err := w.WriteAttachment(attPath, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	rec := MemoRecord{UID: uid, CreateTime: FormatTime(now), UpdateTime: FormatTime(now), ContentPath: ContentPath(uid), Attachments: []AttachmentRecord{{UID: attUID, Filename: "test.txt", Type: "text/plain", Size: size, SHA256: digest, CreateTime: FormatTime(now), Path: attPath}}}
	if err := w.WriteMemo(rec, []byte("#hello")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	r := bytes.NewReader(buf.Bytes())
	a, err := Read(r, int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Memos) != 1 {
		t.Fatalf("memos=%d", len(a.Memos))
	}
	content, err := a.Content(a.Memos[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "#hello" {
		t.Fatalf("content=%q", content)
	}
	got, err := a.Attachment(a.Memos[0].Attachments[0], 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("attachment mismatch")
	}
}

func TestValidateEntryName(t *testing.T) {
	for _, good := range []string{"manifest.json", "memos/abc.json", "attachments/a/file.png"} {
		if err := ValidateEntryName(good); err != nil {
			t.Fatalf("%q: %v", good, err)
		}
	}
	for _, bad := range []string{"../x", "/abs", `a\\b`, "a/../b", "C:/evil", "a//b"} {
		if err := ValidateEntryName(bad); err == nil {
			t.Fatalf("%q should fail", bad)
		}
	}
}

func TestReaderRejectsTamperedAttachment(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	now := time.Date(2026, 10, 2, 6, 0, 0, 0, time.UTC)
	manifest := Manifest{Format: Format, FormatVersion: FormatVersion, Generator: Generator{Name: "NoteHub", Version: "test"}, ExportTime: FormatTime(now), Counts: Counts{Memos: 1, Attachments: 1}}
	mb, _ := jsonBytes(manifest)
	mw, _ := zw.Create(ManifestEntry)
	_, _ = mw.Write(mb)
	uid := "0123456789abcdef0123456789abcdef"
	auid := "abcdef0123456789abcdef0123456789"
	contentPath := ContentPath(uid)
	cw, _ := zw.Create(contentPath)
	_, _ = cw.Write([]byte("memo"))
	badPayload := []byte("tampered")
	d := sha256.Sum256([]byte("expected"))
	rec := MemoRecord{UID: uid, CreateTime: FormatTime(now), UpdateTime: FormatTime(now), ContentPath: contentPath, Attachments: []AttachmentRecord{{UID: auid, Filename: "x.bin", Type: "application/octet-stream", Size: int64(len(badPayload)), SHA256: hex.EncodeToString(d[:]), CreateTime: FormatTime(now), Path: AttachmentPath(auid, "x.bin")}}}
	rb, _ := jsonBytes(rec)
	rw, _ := zw.Create(RecordPath(uid))
	_, _ = rw.Write(rb)
	aw, _ := zw.Create(AttachmentPath(auid, "x.bin"))
	_, _ = aw.Write(badPayload)
	_ = zw.Close()
	r := bytes.NewReader(buf.Bytes())
	a, err := Read(r, int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Attachment(a.Memos[0].Attachments[0], 1<<20); err == nil {
		t.Fatal("expected checksum failure")
	}
}

func jsonBytes(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	err := enc.Encode(v)
	return b.Bytes(), err
}
