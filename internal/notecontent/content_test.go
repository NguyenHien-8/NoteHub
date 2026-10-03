package notecontent

import "testing"

func TestRichTextProjectionSkipsCSSAndPreservesUnicode(t *testing.T) {
	content := RichPrefix + `<html><head><style>#ffffff {color:red}</style></head><body><p>Ghi <b>chú</b> #Study</p><pre>#Code</pre></body></html>`
	if got := Plain(content); got != "Ghi chú #Study\n#Code" {
		t.Fatalf("plain=%q", got)
	}
	if got := TagSource(content); got != "Ghi chú #Study\n" {
		t.Fatalf("tags=%q", got)
	}
	if Plain("**legacy** #Tag") != "**legacy** #Tag" {
		t.Fatal("legacy rewritten")
	}
}
