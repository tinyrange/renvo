//go:build !renvo

package runtime

import (
	"encoding/binary"
	"testing"
)

// A readonly invariant has no hot allocation. Using it as another slot's
// next-iteration input must read its value, not reinterpret its slot number as
// a Const immediate. The second iteration independently proves the transfer.
func TestNativeLoopUnallocatedInvariantPhiSource(t *testing.T) {
	const words = 4
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(words, 3); err != nil {
		t.Fatal(err)
	}
	var b Builder
	address, invariant := b.Load(0), b.Load(1)
	b.Store(2, b.MemoryLoad(address, 8))
	b.Store(0, b.Load(2))
	b.Checkpoint(words, 1)
	b.Store(0, invariant)
	b.Checkpoint(words, 2)
	b.Store(3, b.Constant(0))
	b.Checkpoint(words, 3)
	b.LoopContinue(b.Constant(1))
	ops := b.FinishMemory(words)
	for _, codeVersioned := range []bool{false, true} {
		t.Run(map[bool]string{false: "generic", true: "code-versioned"}[codeVersioned], func(t *testing.T) {
			page := new([4096]byte)
			binary.LittleEndian.PutUint64(page[:], 9)
			binary.LittleEndian.PutUint64(page[8:], 19)
			epoch := uint64(1)
			memory := new(MemoryContext)
			memory.Fill(1, page, 1, &epoch)
			var entry int
			if codeVersioned {
				entry, err = n.CompileCodeVersionedLoop(ops, words, 3, memory)
			} else {
				entry, err = n.CompileLoop(ops, words, 3)
			}
			if err != nil {
				t.Fatal(err)
			}
			memory.ClaimLinks(n)
			memory.PublishLink(0, entry, 3, [17]uint8{})
			state := []uint64{4096, 4104, 99, 0}
			if err = n.CallLinked(state, memory, 6); err != nil {
				t.Fatal(err)
			}
			if memory.Status != 0 || memory.Total != 6 || state[0] != 4104 || state[1] != 4104 || state[2] != 19 || state[3] != 0 {
				t.Fatalf("unallocated invariant phi source: state=%v progress=%d status=%d address=%d", state, memory.Total, memory.Status, memory.Address)
			}
		})
	}
}
