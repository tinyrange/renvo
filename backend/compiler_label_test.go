package main

import "testing"

func TestAsmLabelTableGrowsPastReservedCapacity(t *testing.T) {
	a := renvoAsm{
		labelPos: make([]int32, 0, 2),
	}
	for want := 0; want < 3; want++ {
		if got := renvoAsmNewLabel(&a); got != want {
			t.Fatalf("label %d: got %d", want, got)
		}
	}
	if len(a.labelPos) != 3 {
		t.Fatalf("label table length = %d, want 3", len(a.labelPos))
	}
	for i := 0; i < len(a.labelPos); i++ {
		if got := renvoAsmLabelPosition(&a, i); got != -1 {
			t.Fatalf("new label %d position = %d, want unset sentinel -1", i, got)
		}
	}
	renvoAsmMarkLabel(&a, 1)
	if got := renvoAsmLabelPosition(&a, 1); got != 0 {
		t.Fatalf("label marked at code offset zero has position %d", got)
	}
}
