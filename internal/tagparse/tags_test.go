package tagparse

import (
	"reflect"
	"testing"
)

func TestExtract(t *testing.T) {
	got := Extract("Hello #FPGA #work/project\n`#skip`\n```\n#skip2\n```\n#tiếngViệt #FPGA")
	want := []string{"FPGA", "work", "work/project", "tiếngViệt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
