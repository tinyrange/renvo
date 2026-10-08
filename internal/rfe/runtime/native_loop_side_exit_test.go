//go:build !renvo

package runtime

import "testing"

func TestNativeLoopSideExitContinuesInDispatcher(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(4, 3); err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(7)))
	b.Store(3, b.Constant(4))
	b.Checkpoint(4, 1)
	b.RegionGuard(b.Load(1))
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(99)))
	b.Store(3, b.Constant(0))
	b.Checkpoint(4, 2)
	b.LoopContinue(b.Constant(1))
	region, err := n.CompileLoop(b.FinishMemory(4), 4, 2)
	if err != nil {
		t.Fatal(err)
	}
	var successor Builder
	successor.Store(2, successor.Binary(Add, successor.Load(0), successor.Constant(11)))
	successor.Store(3, successor.Constant(8))
	leaf, err := n.Compile(successor.Finish(4), 4)
	if err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, region, 2, [17]uint8{})
	m.PublishLink(4, leaf, 1, [17]uint8{})
	state := []uint64{3, 0, 0, 0}
	if err = n.CallLinked(state, m, 4); err != nil {
		t.Fatal(err)
	}
	if state[0] != 10 || state[2] != 21 || state[3] != 8 || m.Total != 2 || m.Remaining != 2 || m.Status != 0 || m.LoopExits != 1 || m.LoopIterations != 0 {
		t.Fatal("side exit returned before native successor or replayed prefix", state, m.Total, m.Remaining, m.Status, m.LoopExits, m.LoopIterations)
	}
}
