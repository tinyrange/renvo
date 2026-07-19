package unsafe

import "testing"

func TestUnsafeHostShim(t *testing.T) {
	var v int32
	if Sizeof(v) != 4 {
		t.Fatalf("Sizeof(int32) = %d", Sizeof(v))
	}
	if Alignof(v) == 0 {
		t.Fatalf("Alignof(int32) = 0")
	}
	if Offsetof(7) != 7 {
		t.Fatalf("Offsetof shim failed")
	}
}
