package link

import "testing"

func TestC11DirectiveLineBoundaries(t *testing.T) {
	for _, source := range []string{"// renvo:c11", "// renvo:c11\n", "// renvo:c11\r", "package main\r\n// renvo:c11\r\n", "\n\r\n// renvo:c11"} {
		if !coreTextHasC11Directive([]byte(source)) {
			t.Fatalf("missing directive: %q", source)
		}
	}
	for _, source := range []string{"", "\r\n", "// renvo:c1", " // renvo:c11", "// renvo:c11 ", "x// renvo:c11", "// renvo:c11x\n", "// renvo:c\n11"} {
		if coreTextHasC11Directive([]byte(source)) {
			t.Fatalf("unexpected directive: %q", source)
		}
	}
}
