package identity

import "testing"

func TestUIDAndToken(t *testing.T) {
	uid, err := NewUID()
	if err != nil || !ValidUID(uid) {
		t.Fatalf("uid=%q err=%v", uid, err)
	}
	tok, err := NewBearerToken()
	if err != nil {
		t.Fatal(err)
	}
	h1, err := HashToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := HashToken(tok)
	if string(h1) != string(h2) || len(h1) != 32 {
		t.Fatalf("unexpected token hash")
	}
}
