//go:build !renvo

package runtime

import "testing"

func TestNativeElapsedExactLinkedPrefixAndLoopIteration(t *testing.T) {
	for _, scale := range []int{1, 100} {
		for _, cyclic := range []bool{false, true} {
			n, err := NewNative(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			if err := n.PrepareLinks(6, 0); err != nil {
				t.Fatal(err)
			}
			ctx := new(MemoryContext)
			b := Builder{MemoryInstructionProgress: true}
			b.Store(2, b.Elapsed(b.Load(1), scale, 0))
			b.Checkpoint(6, 1)
			b.Guard(b.Load(5))
			b.Store(3, b.Elapsed(b.Load(1), scale, 1))
			b.Checkpoint(6, 2)
			b.Store(4, b.Binary(Add, b.Load(4), b.Constant(1)))
			next, again := uint64(8), uint64(0)
			if cyclic {
				next, again = 0, 1
			}
			b.Store(0, b.Constant(next))
			b.Checkpoint(6, 4)
			b.LoopContinue(b.Constant(again))
			ops := b.FinishMemory(6)
			if Validate(ops, 6) == nil {
				t.Fatal("contextual time accepted as pure")
			}
			entry, err := n.CompileCodeVersionedLoop(ops, 6, 4, ctx)
			if err != nil {
				t.Fatal(err)
			}
			ctx.ClaimLinks(n)
			ctx.PublishLink(0, entry, 4, [17]uint8{})
			for _, guard := range []uint64{0, 1} {
				state := []uint64{0, 500, 777, 888, 0, guard}
				if err := n.RunLinkedSession(state, ctx, 16); err != nil {
					t.Fatal(err)
				}
				prefix, retired, iterations := uint64(0), uint64(1), uint64(0)
				if guard == 1 {
					retired, iterations = 4, 1
					if cyclic {
						prefix, retired, iterations = 12, 16, 4
					}
				}
				if state[2] != 500+prefix*uint64(scale) || ctx.Total != retired || state[4] != iterations {
					t.Fatal("wrong dynamic time", scale, cyclic, state, ctx.Total)
				}
				if guard == 0 && (ctx.Status != 2 || state[3] != 888) {
					t.Fatal("failed guard advanced temporal state")
				}
				if guard == 1 && state[3] != 500+(prefix+1)*uint64(scale) {
					t.Fatal("reads incorrectly commoned", state)
				}
			}
			if !cyclic {
				// A second linked body sees retirement from the preceding body as well
				// as its own prefix, without a Go clock synchronization in between.
				c := Builder{MemoryInstructionProgress: true}
				c.Store(2, c.Elapsed(c.Load(1), scale, 0))
				c.Store(0, c.Constant(16))
				c.Checkpoint(6, 1)
				c.LoopContinue(c.Constant(0))
				second, err := n.CompileCodeVersionedLoop(c.FinishMemory(6), 6, 1, ctx)
				if err != nil {
					t.Fatal(err)
				}
				ctx.PublishLink(8, second, 1, [17]uint8{})
				state := []uint64{0, 500, 777, 888, 0, 1}
				if err := n.RunLinkedSession(state, ctx, 16); err != nil {
					t.Fatal(err)
				}
				if ctx.Total != 5 || state[2] != 500+4*uint64(scale) {
					t.Fatal("linked prefix lost", state, ctx.Total)
				}
			}
			n.Close()
		}
	}
}

func TestNativeElapsedLeafStartsFreshAndRejectsBadEncodings(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	var b Builder
	b.Store(0, b.Elapsed(b.Load(1), 100, 3))
	b.Checkpoint(2, 4)
	entry, err := n.CompileMemory(b.FinishMemory(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	state := []uint64{0, 900}
	ctx := &MemoryContext{}
	ctx.Total = 999
	if err := n.CallMemory(entry, state, ctx); err != nil {
		t.Fatal(err)
	}
	if state[0] != 1200 || ctx.Retired != 4 {
		t.Fatal("stale linked count leaked to leaf", state, ctx.Retired)
	}
	for _, scale := range []int{0, -1, 1025} {
		var bad Builder
		bad.Store(0, bad.Elapsed(bad.Load(1), scale, 0))
		if ValidateMemory(bad.FinishMemory(2), 2) == nil {
			t.Fatal("invalid elapsed scale accepted", scale)
		}
	}
}
