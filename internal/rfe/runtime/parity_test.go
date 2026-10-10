//go:build !renvo

package runtime

import (
	"math/bits"
	"testing"
)

func TestNativeByteParity(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	var b Builder
	b.Store(1, b.ByteParity(b.Load(0)))
	ops := b.Finish(2)
	entry, err := n.Compile(ops, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	var loop Builder
	v := loop.Binary(Add, loop.Load(0), loop.Constant(1))
	loop.Store(0, v)
	loop.Store(1, loop.ByteParity(v))
	loop.Checkpoint(3, 1)
	loop.LoopContinue(loop.Constant(1))
	le, err := n.CompileLoop(loop.FinishMemory(3), 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, upper := range []uint64{0, 0x8000000000000000, 0xffffffffffffff00, 0x123456789abcde00} {
		for low := uint64(0); low < 256; low++ {
			value := upper | low
			want := uint64(bits.OnesCount8(uint8(value)) & 1)
			state := []uint64{value, 99}
			if err = Interpret(ops, state); err != nil {
				t.Fatal(err)
			}
			if state[1] != want {
				t.Fatalf("IR parity %x: %v want %d", value, state, want)
			}
			state[1] = 99
			if err = n.Call(entry, state); err != nil {
				t.Fatal(err)
			}
			if state[1] != want {
				t.Fatalf("native parity %x: %v want %d", value, state, want)
			}
			var constant Builder
			folded := constant.ByteParity(constant.Constant(value))
			if constant.Ops[folded].Kind != Const || constant.Ops[folded].Imm != want {
				t.Fatal("constant parity", value)
			}
			m := new(MemoryContext)
			m.ClaimLinks(n)
			m.PublishLink(0, le, 1, [17]uint8{})
			ls := []uint64{value, 99, 0}
			if err = n.CallLinked(ls, m, 3); err != nil {
				t.Fatal(err)
			}
			if ls[0] != value+3 || ls[1] != uint64(bits.OnesCount8(uint8(value+3))&1) || m.Total != 3 {
				t.Fatal("loop parity", value, ls, m.Total)
			}
		}
	}
}
