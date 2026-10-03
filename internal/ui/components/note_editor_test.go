package components

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestNoteEditorFormatsSelectionAndKeepsDraftOnBusy(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	ed := NewNoteEditor(nil)
	ed.SetText("Xin chào thế giới")
	ed.Entry.TypedShortcut(&fyne.ShortcutSelectAll{})
	ed.Format("bold", "")
	if ed.Entry.Text != "**Xin chào thế giới**" {
		t.Fatalf("format selection: %q", ed.Entry.Text)
	}
	ed.SetBusy(true)
	ed.Format("italic", "")
	if ed.Entry.Text != "**Xin chào thế giới**" || !ed.Entry.Disabled() {
		t.Fatal("toolbar modified a busy draft")
	}
	ed.SetBusy(false)
	ed.Entry.TypedShortcut(&fyne.ShortcutSelectAll{})
	ed.Format("clear", "")
	if ed.Entry.Text != "Xin chào thế giới" {
		t.Fatalf("clear formatting: %q", ed.Entry.Text)
	}
}

func TestNoteEditorLineActionsUseUnicodeOffsets(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	ed := NewNoteEditor(nil)
	ed.SetText("Đầu\nMột\nHai")
	ed.Entry.CursorRow, ed.Entry.CursorColumn = 1, 2
	ed.Entry.Refresh()
	ed.Format("bullet", "")
	if ed.Entry.Text != "Đầu\n- Một\nHai" {
		t.Fatalf("list targeted wrong line: %q", ed.Entry.Text)
	}
	ed.Entry.TypedShortcut(&fyne.ShortcutSelectAll{})
	ed.Format("number", "")
	if ed.Entry.Text != "1. Đầu\n2. Một\n3. Hai" {
		t.Fatalf("list conversion: %q", ed.Entry.Text)
	}
	ed.Entry.TypedShortcut(&fyne.ShortcutSelectAll{})
	ed.Format("indent", "")
	if !strings.HasPrefix(ed.Entry.Text, "    1. Đầu\n    2.") {
		t.Fatalf("indent: %q", ed.Entry.Text)
	}
	ed.Entry.TypedShortcut(&fyne.ShortcutSelectAll{})
	ed.Format("outdent", "")
	if ed.Entry.Text != "1. Đầu\n2. Một\n3. Hai" {
		t.Fatalf("outdent: %q", ed.Entry.Text)
	}
}

func TestMarkdownStylesAndLiteralCode(t *testing.T) {
	runs := parseNoteMarkdown("**bold _both_** ~~gone~~ <u>under</u> <span style=\"font-size:24px;color:#123456;background-color:#ffff00\">marked</span>\n\n`<u>literal</u>`\n\n```html\n<b>literal block</b>\n```\n\n![external](https://example.invalid/a.png)")
	found := map[string]markdownStyle{}
	for _, run := range runs {
		found[strings.TrimSpace(run.text)] = run.style
	}
	if !found["both"].text.Bold || !found["both"].text.Italic || !found["gone"].strike || !found["under"].underline {
		t.Fatalf("lost nested or extended styles: %#v", found)
	}
	if found["marked"].size != 24 || found["marked"].foreground == nil || found["marked"].background == nil {
		t.Fatal("lost span styles")
	}
	plain := MarkdownPlainText("`<u>literal</u>`\n\n```\n**literal block**\n```\n\n![external](https://example.invalid/a.png)")
	if !strings.Contains(plain, "<u>literal</u>") || !strings.Contains(plain, "**literal block**") || !strings.Contains(plain, "external") {
		t.Fatalf("code/image fallback lost content: %q", plain)
	}
}

func TestMarkdownImportRejectsBinaryAndOversizedInput(t *testing.T) {
	for _, input := range []string{"bad\x00text", string([]byte{0xff}), strings.Repeat("x", maxMarkdownBytes+1)} {
		if _, err := ReadMarkdown(strings.NewReader(input)); err == nil {
			t.Fatal("accepted invalid Markdown input")
		}
	}
	got, err := ReadMarkdown(strings.NewReader("\ufeff# Title\r\n\r\nText"))
	if err != nil || got != "# Title\n\nText" {
		t.Fatalf("normalization: %q, %v", got, err)
	}
}

func TestNoteEditorFormattingWithoutSelectionArmsStyleAtCursor(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	ed := NewNoteEditor(nil)
	ed.SetText("abc")
	ed.Entry.CursorRow, ed.Entry.CursorColumn = 0, 3
	ed.Entry.Refresh()
	ed.Format("bold", "")
	if ed.Entry.Text != "abc****" {
		t.Fatalf("bold markers = %q", ed.Entry.Text)
	}
	if ed.Entry.CursorTextOffset() != 5 {
		t.Fatalf("cursor should be between bold markers, got %d", ed.Entry.CursorTextOffset())
	}
}
