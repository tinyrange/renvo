package runtime

import (
	"math/rand"
	"testing"
)

func TestValueNumberingStateVersions(t *testing.T) {
	var b Builder
	a, other := b.Load(0), b.Load(1)
	first := b.Binary(Add, a, other)
	duplicate := b.Binary(Add, b.Load(0), b.Load(1))
	if first != duplicate {
		t.Fatal("duplicate expression was not shared")
	}
	b.Store(2, first)
	b.Store(0, b.Binary(Add, a, b.Constant(1)))
	updated := b.Binary(Add, b.Load(0), b.Load(1))
	if updated == first {
		t.Fatal("expression reused across a changed operand")
	}
	b.Store(3, updated)
	// The first expression must retain the OLD value of slot zero, even though
	// the builder commits the new value of slot zero before slots two and three.
	ops := b.Finish(4)
	rng := rand.New(rand.NewSource(91))
	for i := 0; i < 1000; i++ {
		x, y := rng.Uint64(), rng.Uint64()
		state := []uint64{x, y, 0, 0}
		if err := Interpret(ops, state); err != nil {
			t.Fatal(err)
		}
		if state[0] != x+1 || state[1] != y || state[2] != x+y || state[3] != x+1+y {
			t.Fatalf("state versioning: %x", state)
		}
	}
}
