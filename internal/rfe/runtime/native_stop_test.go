//go:build !renvo

package runtime

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNativeStopReplacesInstructionQuantumAndPreservesProgress(t *testing.T) {
	n, m, ram := ownedFixture(t)
	var stop uint32
	m.Stop = &stop
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(1, b.Constant(0))
	b.Checkpoint(2, 1)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(2), 2, 1, m)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(0, entry, 2, 1); err != nil {
		t.Fatal(err)
	}
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 1, [17]uint8{})
	state := []uint64{7, 0}
	if err = n.RunLinkedBackingSession(state, m, 100000, ram); err != nil || state[0] != 100007 || m.Total != 100000 {
		t.Fatal("old instruction quantum survived", state, m.Total, err)
	}
	timer := time.AfterFunc(2*time.Millisecond, func() { atomic.StoreUint32(&stop, 1) })
	defer timer.Stop()
	before := state[0]
	if err = n.RunLinkedBackingSession(state, m, ^uint64(0), ram); err != nil || state[0]-before != m.Total || atomic.LoadUint32(&stop) != 1 {
		t.Fatal("asynchronous native stop lost progress", state, m.Total, err)
	}
	before = state[0]
	if err = n.RunLinkedBackingSession(state, m, ^uint64(0), ram); err != nil || state[0] != before || m.Total != 0 {
		t.Fatal("already-pending event executed guest work", state, m.Total, err)
	}
}
