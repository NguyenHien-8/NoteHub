package ipc

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/app"
	"github.com/NguyenHien-8/NoteHub/internal/notecontent"
)

func TestProtocolCRUDConflictAndInvalidFrames(t *testing.T) {
	ctx := context.Background()
	b, err := app.OpenBackend(ctx, app.Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	s := New(b, "test")
	defer s.Close()
	call := func(input string) map[string]any {
		t.Helper()
		var out bytes.Buffer
		if err := s.Serve(ctx, strings.NewReader(input+"\n"), &out); err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := json.Unmarshal(out.Bytes(), &value); err != nil {
			t.Fatal(err, out.String())
		}
		return value
	}
	hello := call(`{"id":"1","method":"hello","params":{}}`)
	if hello["id"] != "1" || hello["result"].(map[string]any)["protocol"] != float64(1) {
		t.Fatal(hello)
	}
	created := call(`{"id":"2","method":"memos.create","params":{"content":"Ghi chú #Study"}}`)
	memo := created["result"].(map[string]any)
	uid := memo["uid"].(string)
	if _, leaked := memo["ID"]; leaked {
		t.Fatal("database IDs leaked")
	}
	updated := call(`{"id":"3","method":"memos.update","params":{"uid":"` + uid + `","revision":1,"content":"Changed"}}`)
	if updated["error"] != nil {
		t.Fatal(updated)
	}
	conflict := call(`{"id":"4","method":"memos.update","params":{"uid":"` + uid + `","revision":1,"content":"Lost edit"}}`)
	if conflict["error"].(map[string]any)["code"] != "conflict" {
		t.Fatal(conflict)
	}
	if call(`{"id":"5","method":"unknown"}`)["error"] == nil {
		t.Fatal("unknown method accepted")
	}
	if call(`not json`)["error"] == nil {
		t.Fatal("invalid JSON accepted")
	}
	got := call(`{"id":"6","method":"memos.get","params":{"uid":"` + uid + `"}}`)
	if got["result"].(map[string]any)["content"] != "Changed" {
		t.Fatal("conflict overwrote note")
	}
}

func TestRichTextSearchTagsBackupAndAttachmentIPC(t *testing.T) {
	ctx := context.Background()
	b, err := app.OpenBackend(ctx, app.Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	s := New(b, "test")
	defer s.Close()
	invoke := func(method string, p any) any {
		t.Helper()
		raw, _ := json.Marshal(p)
		v, err := s.dispatch(ctx, method, raw)
		if err != nil {
			t.Fatalf("%s: %v", method, err)
		}
		return v
	}
	rich := notecontent.RichPrefix + `<html><head><style>body {color:#abcdef}</style></head><body><p>Ghi <b>chú</b> #Research/Qt</p></body></html>`
	created := invoke("memos.create", map[string]any{"content": rich}).(memoDTO)
	if len(created.Tags) != 2 || created.Tags[1] != "Research/Qt" {
		t.Fatalf("tags=%v", created.Tags)
	}
	found := invoke("memos.list", map[string]any{"text": "ghi chu"}).(map[string]any)["items"].([]memoDTO)
	if len(found) != 1 {
		t.Fatal("rich text not searchable")
	}
	noise := invoke("memos.list", map[string]any{"text": "abcdef"}).(map[string]any)["items"].([]memoDTO)
	if len(noise) != 0 {
		t.Fatal("CSS indexed as text")
	}
	file := filepath.Join(t.TempDir(), "Tiếng Việt.txt")
	if err := os.WriteFile(file, []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}
	attached := invoke("attachments.add", map[string]any{"uid": created.UID, "paths": []string{file, file}}).(map[string]any)["memo"].(memoDTO)
	if len(attached.Files) != 2 {
		t.Fatal("attachments missing")
	}
	invoke("attachments.reorder", map[string]any{"uid": created.UID, "order": []string{attached.Files[1].UID, attached.Files[0].UID}})
	latest := invoke("memos.get", map[string]any{"uid": created.UID}).(memoDTO)
	if latest.Files[0].UID != attached.Files[1].UID {
		t.Fatal("order not saved")
	}
	archive := filepath.Join(t.TempDir(), "backup.zip")
	invoke("backup.export", map[string]any{"path": archive})
	invoke("memos.delete", map[string]any{"uid": created.UID})
	invoke("backup.import", map[string]any{"path": archive, "policy": "skip"})
	restored := invoke("memos.get", map[string]any{"uid": created.UID}).(memoDTO)
	if restored.Content != rich || len(restored.Files) != 2 {
		t.Fatal("backup lost rich content or files")
	}
	if s.sharing != nil {
		t.Fatal("sharing started without request")
	}
	invoke("sharing.set", map[string]any{"enabled": true, "port": 0})
	grant := invoke("shares.create", map[string]any{"uid": created.UID, "days": 1}).(map[string]any)
	if !strings.HasPrefix(grant["url"].(string), "http://127.0.0.1:") {
		t.Fatal(grant)
	}
	invoke("shares.revoke", map[string]any{"uid": grant["uid"]})
	invoke("sharing.set", map[string]any{"enabled": false})
}

func TestBackendPipeProcessExitsOnParentEOF(t *testing.T) {
	if os.Getenv("NOTEHUB_IPC_TEST_CHILD") == "1" {
		b, err := app.OpenBackend(context.Background(), app.Config{DataDir: os.Getenv("NOTEHUB_IPC_TEST_DATA")})
		if err != nil {
			os.Exit(2)
		}
		s := New(b, "test")
		err = s.Serve(context.Background(), os.Stdin, os.Stdout)
		_ = s.Close()
		_ = b.Close()
		if err != nil {
			os.Exit(3)
		}
		os.Exit(0)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestBackendPipeProcessExitsOnParentEOF$")
	cmd.Env = append(os.Environ(), "NOTEHUB_IPC_TEST_CHILD=1", "NOTEHUB_IPC_TEST_DATA="+t.TempDir())
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err = in.Write([]byte("{\"id\":\"hello\",\"method\":\"hello\"}\n")); err != nil {
		t.Fatal(err)
	}
	var res response
	if err = json.NewDecoder(out).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if res.ID != "hello" || res.Error != nil {
		t.Fatal(res)
	}
	_ = in.Close()
	if err = cmd.Wait(); err != nil {
		t.Fatalf("EOF did not cleanly stop backend: %v", err)
	}
}

func TestProtocolMultipleRecordsAndOversize(t *testing.T) {
	ctx := context.Background()
	b, err := app.OpenBackend(ctx, app.Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	s := New(b, "test")
	defer s.Close()
	var out bytes.Buffer
	input := "{\"id\":\"1\",\"method\":\"hello\"}\n{\"id\":\"2\",\"method\":\"metadata\"}\n"
	if err := s.Serve(ctx, strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	if len(strings.Split(strings.TrimSpace(out.String()), "\n")) != 2 {
		t.Fatal(out.String())
	}
	if err := s.Serve(ctx, strings.NewReader(strings.Repeat("x", (8<<20)+1)), &out); err == nil {
		t.Fatal("unbounded request")
	}
}
