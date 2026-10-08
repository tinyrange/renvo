//go:build !renvo

package runtime

import (
	"reflect"
	"testing"
	"unsafe"
)

// Shadow state/context must never detach a guest-visible alias. These cases
// deliberately alias RAM or generation metadata with architectural state and
// expect the conservative original boundary, without detached shadow aliases.
func TestNativeSessionStateAliasesUseCompatibility(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.MemoryStore(b.Constant(4096), b.Constant(37), 8)
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(2, b.Constant(0))
	b.Checkpoint(3, 1)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(3), 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(0, entry, 3, 1); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"ram", "clock", "epoch", "context"} {
		type outcome struct {
			State                               []uint64
			Page                                [4096]byte
			Clock, Epoch, Total, Memory, Status uint64
		}
		var results [2]outcome
		for trial := range results {
			page := new([4096]byte)
			state := []uint64{17, 99, 0}
			clock, epoch := uint64(1), uint64(1)
			m := &MemoryContext{Clock: &clock}
			switch mode {
			case "ram":
				state = unsafe.Slice((*uint64)(unsafe.Pointer(page)), 3)
				copy(state, []uint64{17, 99, 0})
			case "clock":
				m.Clock = &state[0]
			case "context":
				// Alias only integer descriptors, not Go-private pointer fields.
				page = (*[4096]byte)(unsafe.Pointer(&m.Blocks[100]))
			}
			epochPointer := &epoch
			if mode == "epoch" {
				epochPointer = &state[0]
			}
			m.Fill(1, page, 3, epochPointer)
			m.ClaimLinks(n)
			m.PublishLink(0, entry, 1, [17]uint8{0, 1})
			if trial == 0 {
				err = n.CallLinked(state, m, 64)
			} else {
				err = n.RunLinkedSession(state, m, 65536)
			}
			if err != nil {
				t.Fatal(mode, trial, err)
			}
			budget := uint64(64)
			if trial == 1 {
				budget = 65536
			}
			if m.Total != 64 || m.Remaining != budget-64 || m.MemoryTotal != 64 || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 || m.DescriptorBase != 0 {
				t.Fatal(mode, trial, state, m.Total, m.Remaining, m.MemoryTotal)
			}
			results[trial] = outcome{state, *page, *m.Clock, *epochPointer, m.Total, m.MemoryTotal, m.Status}
		}
		if !reflect.DeepEqual(results[0], results[1]) {
			t.Fatal("session detached alias", mode, results[0].State, results[1].State)
		}
	}
}
