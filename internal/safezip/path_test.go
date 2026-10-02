package safezip

import "testing"

func TestValidateName(t *testing.T) {
	good := []string{"manifest.json", "attachments/abc/file.png"}
	bad := []string{"../x", "/abs", `a\\b`, "a/../b"}
	for _, v := range good {
		if err := ValidateName(v); err != nil {
			t.Fatalf("%q should pass: %v", v, err)
		}
	}
	for _, v := range bad {
		if err := ValidateName(v); err == nil {
			t.Fatalf("%q should fail", v)
		}
	}
}
