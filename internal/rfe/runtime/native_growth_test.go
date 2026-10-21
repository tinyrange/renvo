//go:build !renvo

package runtime

import (
	"renvo.dev/internal/rfeabi"
	"testing"
)

func TestLinkedOwnerSurvivesArenaRelocationAndRetiresOnlyDeadEntries(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("supported foreign boundary unavailable")
	}
	n, err := NewNative(4096)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	n.AllowGrowth()
	ram := make([]byte, 4096)
	clock := uint64(1)
	m := &MemoryContext{NativeContext: NativeContext{Clock: &clock}}
	m.Memory = rfeabi.NewOwnedMemory(ram, &clock, rfeabi.NewWriteWatches(len(ram)))
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(1, b.Constant(0))
	b.Checkpoint(2, 1)
	b.LoopContinue(b.Constant(1))
	e, err := n.CompileCodeVersionedLoop(b.FinishMemory(2), 2, 1, m)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(0, e, 2, 1); err != nil {
		t.Fatal(err)
	}
	m.ClaimLinks(n)
	m.PublishLink(0, e, 1, [17]uint8{})
	for pass := 0; pass < 2; pass++ {
		if pass == 1 {
			for j := 0; j < 500; j++ {
				var filler Builder
				filler.Store(0, filler.Binary(Add, filler.Load(0), filler.Constant(uint64(j))))
				dead, err := n.Compile(filler.Finish(2), 2)
				if err != nil {
					t.Fatal(err)
				}
				n.Release(dead)
				if err = n.Call(dead, []uint64{0, 0}); err == nil {
					t.Fatal("retired code resurrected")
				}
			}
		}
		state := []uint64{7, 0}
		if err = n.RunLinkedBackingSession(state, m, 128, ram); err != nil || state[0] != 135 || m.Total != 128 {
			t.Fatal("linked relocation lost code/admission", pass, state, m.Total, err)
		}
	}
}
