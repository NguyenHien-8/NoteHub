package platform

import (
	"reflect"
	"testing"
)

func TestPrepareMaximizedCreationHintIsScopedToOneWindow(t *testing.T) {
	var states []bool
	restore := prepareMaximizedCreationHint(func(maximized bool) {
		states = append(states, maximized)
	})

	if !reflect.DeepEqual(states, []bool{true}) {
		t.Fatalf("creation hint not enabled before show: %v", states)
	}

	restore()
	restore() // idempotent: a defer + explicit cleanup must not flip it twice.
	if !reflect.DeepEqual(states, []bool{true, false}) {
		t.Fatalf("creation hint not restored exactly once: %v", states)
	}
}
