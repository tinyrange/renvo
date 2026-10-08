//go:build !renvo

package runtime

import (
	"renvo.dev/internal/backendcompiled"
	"testing"
)

func TestNativeLoopZeroProgressSideExitRejected(t *testing.T) {
	n, err := NewNative(4096)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	var b Builder
	b.RegionGuard(b.Load(0)) // would make dispatch spin without consuming its budget
	b.Store(1, b.Constant(0))
	b.Checkpoint(2, 2)
	b.LoopContinue(b.Constant(1))
	ops := b.FinishMemory(2)
	if _, err = n.CompileLoop(ops, 2, 2); err == nil {
		t.Fatal("zero-progress side exit accepted")
	}
	records := make([]int, 0, len(ops)*4)
	for _, o := range ops {
		records = append(records, o.Kind, int(o.A), int(o.B), int(o.Imm))
	}
	for _, arm := range []bool{false, true} {
		if _, ok := backendcompiled.RenvoEmitLoopBlock(records, 2, 2, arm); ok {
			t.Fatal("raw native emitter accepted zero-progress side exit", arm)
		}
	}
}
