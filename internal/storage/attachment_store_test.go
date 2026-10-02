package storage

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestLocalStoreSaveOpenDelete(t *testing.T) {
	s, err := NewLocalStore(t.TempDir(), 1024)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Save("0123456789abcdef0123456789abcdef", "abcdef0123456789abcdef0123456789", "a.txt", bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Size != 5 || len(got.SHA256) != 64 {
		t.Fatalf("saved=%+v", got)
	}
	f, err := s.Open(got.RelativePath)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(f)
	_ = f.Close()
	if string(b) != "hello" {
		t.Fatalf("got %q", b)
	}
	if err := s.Delete(got.RelativePath); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(got.RelativePath); err == nil {
		t.Fatal("expected file to be deleted")
	}
}

func TestLocalStoreLimit(t *testing.T) {
	s, err := NewLocalStore(t.TempDir(), 4)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Save("0123456789abcdef0123456789abcdef", "abcdef0123456789abcdef0123456789", "a.bin", bytes.NewBufferString("12345"))
	if !errors.Is(err, ErrAttachmentTooLarge) {
		t.Fatalf("err=%v", err)
	}
}

func TestEnsureInside(t *testing.T) {
	root := t.TempDir()
	if _, err := ensureInside(root, "../escape"); err == nil {
		t.Fatal("expected traversal rejection")
	}
}
