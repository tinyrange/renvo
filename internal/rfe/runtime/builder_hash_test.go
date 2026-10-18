package runtime

import (
	"reflect"
	"testing"
)

func TestBuilderValueNumberingChecksCollisionsAndEveryKeyField(t *testing.T) {
	var b Builder
	var constants []uint64
	// Force a chain crossing the end of the initial table. All candidates share
	// their low 32 immediate bits, exercising full-width key equality as well.
	for high := uint64(0); len(constants) < 7; high++ {
		value := high<<32 | 7
		if builderValueHash(Op{Kind: Const, Imm: value})&15 == 15 {
			constants = append(constants, value)
		}
	}
	var entries []Value
	for _, value := range constants {
		entry := b.Constant(value)
		if int(entry) != len(entries) {
			t.Fatal("distinct colliding constants were merged", value, entry)
		}
		entries = append(entries, entry)
	}
	for i, value := range constants {
		if got := b.Constant(value); got != entries[i] || b.Ops[got].Imm != value {
			t.Fatal("collision lookup lost a full-width key", value, got)
		}
	}
	// Value numbering preserves the whole Op identity, not just its immediate.
	for _, op := range []Op{
		{Kind: Const, A: 1, Imm: constants[0]},
		{Kind: Const, B: 1, Imm: constants[0]},
		{Kind: Add, A: entries[0], B: entries[1]},
		{Kind: Sub, A: entries[0], B: entries[1]},
		{Kind: Add, A: entries[1], B: entries[0]},
	} {
		old := len(b.Ops)
		entry := b.emit(op)
		if int(entry) != old || b.emit(op) != entry {
			t.Fatal("composite expression identity changed", op, entry)
		}
	}
	for i := uint64(0); i < 800; i++ {
		value := i*0x100000001 + 0x8000000000000000
		entry := b.Constant(value)
		if b.Constant(value) != entry || b.Ops[entry].Imm != value {
			t.Fatal("rehash changed an existing value", i)
		}
	}
	if len(b.valueBuckets) > 4*len(b.Ops) || b.valueCount > len(b.Ops) {
		t.Fatal("value index is not bounded by emitted operations")
	}
}

func TestBuilderHashedValueNumberingPreservesTranslatedSemanticsAfterReset(t *testing.T) {
	var b Builder
	var published [][]Op
	for round := 0; round < 3; round++ {
		b.Reset(false)
		x, y := b.Load(0), b.Load(1)
		for i := 0; i < 80; i++ {
			constant := uint64(i+1)*0x100000001 + uint64(round)<<61
			c := b.Constant(constant)
			sum := b.Binary(Add, x, c)
			if b.Binary(Add, x, b.Constant(constant)) != sum {
				t.Fatal("identical translated expressions were not interned")
			}
			x = b.Binary(Xor, sum, b.Binary(Mul, y, b.Constant(uint64(i*2+1))))
			y = b.Binary(Add, y, b.Constant(constant>>16))
			b.Store(0, x)
			if b.Load(0) != x {
				t.Fatal("architectural state forwarding changed")
			}
		}
		b.Store(0, x)
		b.Store(1, y)
		ops := b.Finish(2)
		if err := Validate(ops, 2); err != nil {
			t.Fatal(err)
		}
		published = append(published, ops)
	}
	saved := append([]Op(nil), published[0]...)
	b.Reset(true)
	b.Store(0, b.Constant(0xffffffffffffffff))
	b.Checkpoint(2, 1)
	if err := ValidateMemory(b.FinishMemory(2), 2); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(published[0], saved) {
		t.Fatal("reset or memory lowering mutated published pure IR")
	}
	for round, ops := range published {
		for _, initial := range [][2]uint64{{0, 0}, {1, 2}, {1 << 63, ^uint64(0)}, {0x0123456789abcdef, 0xfedcba9876543210}} {
			want := initial
			for i := 0; i < 80; i++ {
				constant := uint64(i+1)*0x100000001 + uint64(round)<<61
				want[0] = (want[0] + constant) ^ (want[1] * uint64(i*2+1))
				want[1] += constant >> 16
			}
			state := []uint64{initial[0], initial[1]}
			if err := Interpret(ops, state); err != nil || state[0] != want[0] || state[1] != want[1] {
				t.Fatal("translated modular semantics changed", round, initial, state, want, err)
			}
		}
	}
}
