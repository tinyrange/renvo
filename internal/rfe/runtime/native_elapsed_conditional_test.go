//go:build !renvo

package runtime

import "testing"

// Scale 100 deliberately exceeds record count. Conditional exit reconstruction
// must not interpret scalar timing metadata as an SSA alias index.
func TestNativeElapsedConditionalExitKeepsScalarMetadata(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err := n.PrepareLinks(6, 0); err != nil {
		t.Fatal(err)
	}
	ctx := new(MemoryContext)
	b := Builder{MemoryInstructionProgress: true}
	b.Store(2, b.Elapsed(b.Load(1), 100, 0))
	b.Store(0, b.Choose(b.Binary(Equal, b.Load(5), b.Constant(0)), b.Constant(8), b.Constant(0)))
	b.Checkpoint(6, 2)
	b.LoopContinue(b.Binary(Equal, b.Load(0), b.Constant(0)))
	entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(6), 6, 2, ctx)
	if err != nil {
		t.Fatal(err)
	}
	ctx.ClaimLinks(n)
	ctx.PublishLink(0, entry, 2, [17]uint8{})
	for _, loop := range []uint64{0, 1} {
		state := []uint64{0, 500, 777, 888, 0, loop}
		if err := n.RunLinkedSession(state, ctx, 8); err != nil {
			t.Fatal(err)
		}
		wantTime, wantTotal, wantPC := uint64(500), uint64(2), uint64(8)
		if loop != 0 {
			wantTime, wantTotal, wantPC = 1100, 8, 0
		}
		if state[2] != wantTime || ctx.Total != wantTotal || state[0] != wantPC {
			t.Fatal("conditional temporal exit corrupted", state, ctx.Total)
		}
	}
}
