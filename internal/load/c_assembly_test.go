package load

import (
	"strconv"
	"testing"
)

func TestAssemblySourceIndexWordBoundaries(t *testing.T) {
	maximum := int(^uint(0) >> 1)
	for _, index := range []int{0, 1, 9, 10, 99, 100, -1, -10, maximum, -maximum - 1} {
		if got, want := decimalAssemblyIndex(index), strconv.Itoa(index); got != want {
			t.Fatalf("source index %d: got %q, want %q", index, got, want)
		}
	}
}
