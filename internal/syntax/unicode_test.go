package syntax

import "testing"

func TestUnicodeIdentifierCategories(t *testing.T) {
	// Exhaustive classification is checked against the pinned independent
	// Unicode 15 oracle in unicode_snapshot_test.go, not the host version.
	for _, name := range []string{"Ω", "É", "𐐀"} {
		if !IdentifierExported([]byte(name), 0) {
			t.Fatalf("not exported: %s", name)
		}
	}
	for _, name := range []string{"π", "世界", "ǅ", "_"} {
		if IdentifierExported([]byte(name), 0) {
			t.Fatalf("incorrectly exported: %s", name)
		}
	}
}
