package runtime

import (
	"reflect"
	"testing"
)

func TestTranslationScratchPreservesPublishedBlocks(t *testing.T) {
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(7)))
	first := b.Finish(2)
	saved := append([]Op(nil), first...)
	for round := 0; round < 128; round++ {
		b.Reset(round%2 != 0)
		b.Store(1, b.Binary(Xor, b.Load(0), b.Constant(uint64(round))))
		var next []Op
		if round%2 != 0 {
			b.Checkpoint(2, 1)
			next = b.FinishMemory(2)
			if err := ValidateMemory(next, 2); err != nil {
				t.Fatal(err)
			}
		} else {
			next = b.Finish(2)
			if err := Validate(next, 2); err != nil {
				t.Fatal(err)
			}
		}
		if !reflect.DeepEqual(first, saved) {
			t.Fatal("published operations alias translation scratch")
		}
		state := []uint64{11, 0}
		if err := Interpret(first, state); err != nil || state[0] != 18 || state[1] != 0 {
			t.Fatal("old block changed", state, err)
		}
		if round%2 == 0 {
			state = []uint64{11, 0}
			if err := Interpret(next, state); err != nil || state[0] != 11 || state[1] != (11^uint64(round)) {
				t.Fatal("stale value numbering", round, state, err)
			}
		}
	}
}
