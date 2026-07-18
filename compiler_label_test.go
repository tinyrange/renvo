package main

import "testing"

func TestAsmLabelTableGrowsPastReservedCapacity(t *testing.T) {
	a := rtgAsm{
		labelPos: make([]int, 0, 2),
		labelSet: make([]bool, 0, 2),
	}
	for want := 0; want < 3; want++ {
		if got := rtgAsmNewLabel(&a); got != want {
			t.Fatalf("label %d: got %d", want, got)
		}
	}
	if len(a.labelPos) != 3 || len(a.labelSet) != 3 {
		t.Fatalf("label tables did not grow together: positions=%d set=%d", len(a.labelPos), len(a.labelSet))
	}
}
