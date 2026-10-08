//go:build !renvo

package runtime

import (
	"reflect"
	"testing"
)

// Compare the optimized Go scheduler with individually validated calls, not a
// second copy of its selector. Include every possible remainder and a fault
// after a quantum boundary, so retired/memory counts and partial state matter.
func TestNativeQuantaMatchesSeparateCalls(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	var b Builder
	address := b.Load(1)
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Checkpoint(3, 1)
	b.MemoryStore(address, b.Load(0), 8)
	b.Store(1, b.Binary(Add, address, b.Constant(8)))
	b.Checkpoint(3, 2)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(3), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, start := range []uint64{4096, 8192 - 40*8} {
		for remaining := uint64(1); remaining <= 1025; remaining++ {
			var states [2][]uint64
			var contexts [2]*MemoryContext
			var pages [2]*[4096]byte
			var clocks, epochs [2]uint64
			for i := range states {
				states[i] = []uint64{0, start, 0}
				pages[i] = new([4096]byte)
				clocks[i], epochs[i] = 1, 1
				contexts[i] = &MemoryContext{NativeContext: NativeContext{Clock: &clocks[i]}}
				contexts[i].Fill(1, pages[i], 3, &epochs[i])
				contexts[i].ClaimLinks(n)
				contexts[i].PublishLink(0, entry, 2, [17]uint8{})
				contexts[i].Blocks[0].Reserved = ^uint64(0)
			}
			calls, err := n.RunLinkedQuanta(states[0], contexts[0], 16, remaining, 7)
			if err != nil {
				t.Fatal(err)
			}
			m := contexts[1]
			left := remaining
			var total, memory, exits, iterations uint64
			separate := 0
			for separate < 16 && left != 0 {
				budget := left
				if budget > 64 {
					budget = 64
				}
				if err = n.CallLinked(states[1], m, budget); err != nil {
					t.Fatal(err)
				}
				separate++
				progress := m.Total
				total += progress
				memory += m.MemoryTotal
				exits += m.LoopExits
				iterations += m.LoopIterations
				left -= progress
				if m.Status != 0 || progress == 0 || left < 2 {
					break
				}
			}
			fast := contexts[0]
			if calls != separate || !reflect.DeepEqual(states[0], states[1]) || *pages[0] != *pages[1] || clocks[0] != clocks[1] || epochs[0] != epochs[1] || fast.Total != total || fast.Remaining != left || fast.MemoryTotal != memory || fast.LoopExits != exits || fast.LoopIterations != iterations || fast.Status != m.Status || fast.Address != m.Address || fast.Retired != m.Retired || fast.CodeView != [4]uint64{} || fast.PreparedTargets != 0 {
				t.Fatal("scheduler disagrees with separate calls", start, remaining, calls, separate, states, fast.Total, total, fast.Remaining, left, fast.Status, m.Status)
			}
		}
	}
}

func TestNativeQuantaRejectsMalformedProgressAndClearsBorrow(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.Checkpoint(2, 17)
	b.Guard(b.Constant(0))
	entry, err := n.CompileMemory(b.FinishMemory(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 1, [17]uint8{})
	state := []uint64{91, 0}
	calls, err := n.RunLinkedQuanta(state, m, 16, 129, 7)
	if err == nil || calls != 1 || m.Status != 3 || m.Total != 0 || m.Remaining != 129 || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 || state[0] != 91 {
		t.Fatal("invalid progress accepted or borrowed executable view escaped", calls, err, state, m.Total, m.Remaining, m.Status)
	}
	// The error path must release the arena lock before a close.
	if err = n.Close(); err != nil {
		t.Fatal(err)
	}
}
