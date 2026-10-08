//go:build !renvo

package runtime

import "testing"

func narrowOracle32(kind int, a, b uint64) uint64 {
	x, y := uint32(a), uint32(b)
	switch kind {
	case Add:
		return uint64(x + y)
	case Sub:
		return uint64(x - y)
	case Mul:
		return uint64(x * y)
	case And:
		return uint64(x & y)
	case Or:
		return uint64(x | y)
	case Xor:
		return uint64(x ^ y)
	}
	panic("bad oracle operation")
}

func TestNativeNarrowArithmeticHighBitsAndMultipleUses(t *testing.T) {
	n, err := NewNative(4 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	values := []uint64{0, 1, 0x7fffffff, 0x80000000, 0xffffffff, 0x100000000, 0x8000000000000001, ^uint64(0)}
	for _, kind := range []int{Add, Sub, Mul, And, Or, Xor} {
		for _, immediate := range []bool{false, true} {
			for _, multiple := range []bool{false, true} {
				var b Builder
				a, rhs := b.Load(0), b.Load(1)
				if immediate {
					rhs = b.Constant(0x80000001)
				}
				wide := b.Binary(kind, a, rhs)
				low := b.Binary(And, wide, b.Constant(0xffffffff))
				b.Store(2, low)
				if multiple {
					b.Store(3, wide)
				}
				// Keep enough unrelated values alive to exercise destinations and spills.
				for i := 4; i < 20; i++ {
					b.Store(i, b.Binary(Add, b.Load(i), b.Constant(uint64(i))))
				}
				entry, err := n.Compile(b.Finish(20), 20)
				if err != nil {
					t.Fatal(err)
				}
				for _, a := range values {
					for _, rhs := range values {
						state := make([]uint64, 20)
						state[0], state[1], state[3] = a, rhs, 0x123456789abcdef0
						usedRhs := rhs
						if immediate {
							usedRhs = 0x80000001
						}
						if err := n.Call(entry, state); err != nil {
							t.Fatal(err)
						}
						if state[2] != narrowOracle32(kind, a, usedRhs) {
							t.Fatalf("kind=%d immediate=%v multiple=%v a=%x rhs=%x low=%x", kind, immediate, multiple, a, usedRhs, state[2])
						}
						if multiple && state[3] != operation(kind, a, usedRhs) {
							t.Fatal("full-width second use was narrowed")
						}
						if !multiple && state[3] != 0x123456789abcdef0 {
							t.Fatal("unwritten slot changed")
						}
						for i := 4; i < 20; i++ {
							if state[i] != uint64(i) {
								t.Fatal("spill clobber", i, state[i])
							}
						}
					}
				}
			}
		}
	}
}

func TestNativeNarrowLoopPhiAndPreciseColdMap(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(8, 7); err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.Store(0, b.Binary(And, b.Binary(Add, b.Load(0), b.Load(1)), b.Constant(0xffffffff)))
	b.Store(7, b.Constant(4))
	b.Checkpoint(8, 1)
	b.MemoryLoad(b.Load(2), 8)
	b.Store(7, b.Constant(0))
	b.Checkpoint(8, 2)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(8), 8, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(0, entry, 8, 2); err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 2, [17]uint8{0, 0, 1})
	state := []uint64{0xffffffff, 0x8000000000000002, 8192, 0, 0, 0, 0, 0}
	if err = n.CallLinked(state, m, 64); err != nil {
		t.Fatal(err)
	}
	if state[0] != 1 || state[7] != 4 || m.Total != 1 || m.Status != 1 || m.Address != 8192 || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 {
		t.Fatal("cold map/retirement mismatch", state, m.Total, m.Status, m.Address)
	}
}
