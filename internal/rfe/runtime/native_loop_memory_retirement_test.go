//go:build !renvo

package runtime

import (
	"encoding/binary"
	"testing"
)

func TestNativeLoopRMWRetiresMemoryInstructionsNotAccesses(t *testing.T) {
	for _, owned := range []bool{false, true} {
		for _, cyclic := range []bool{false, true} {
			n, err := NewNative(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			if err = n.PrepareLinks(3, 2); err != nil {
				t.Fatal(err)
			}
			page := new([4096]byte)
			epoch, clock := uint64(1), uint64(1)
			memory := new(MemoryContext)
			memory.Clock = &clock
			memory.Fill(1, page, 3, &epoch)
			b := Builder{MemoryInstructionProgress: true}
			address := b.Load(0)
			// One guest RMW has two checked host accesses but one retirement.
			v := b.MemoryLoad(address, 8)
			b.MemoryStore(address, b.Binary(Add, v, b.Constant(1)), 8)
			b.Checkpoint(3, 1)
			b.Guard(b.Load(1))
			v = b.MemoryLoad(address, 8)
			b.MemoryStore(address, b.Binary(Add, v, b.Constant(2)), 8)
			b.Checkpoint(3, 2)
			endPC := uint64(4)
			if cyclic {
				endPC = 0
			}
			b.Store(2, b.Constant(endPC))
			b.Checkpoint(3, 5)
			continuation := uint64(0)
			if cyclic {
				continuation = 1
			}
			b.LoopContinue(b.Constant(continuation))
			ops := b.FinishMemory(3)
			var entry int
			if owned {
				entry, err = n.CompileCodeVersionedLoop(ops, 3, 5, memory)
			} else {
				entry, err = n.CompileLoop(ops, 3, 5)
			}
			if err != nil {
				t.Fatal(err)
			}
			memory.ClaimLinks(n)
			memory.PublishLink(0, entry, 5, [17]uint8{0, 1, 2, 2, 2, 2})
			for _, proceed := range []uint64{0, 1} {
				binary.LittleEndian.PutUint64(page[:], 10)
				state := []uint64{4096, proceed, 0}
				err = n.RunLinkedSession(state, memory, 15)
				if err != nil {
					t.Fatalf("owned=%t cyclic=%t proceed=%d: %v total=%d memory=%d", owned, cyclic, proceed, err, memory.Total, memory.MemoryTotal)
				}
				total, mem, value, status := uint64(1), uint64(1), uint64(11), uint64(2)
				if proceed != 0 {
					total, mem, value, status = 5, 2, 13, 0
					if cyclic {
						total, mem, value = 15, 6, 19
					}
				}
				if memory.Total != total || memory.MemoryTotal != mem || memory.Status != status || memory.Remaining != 15-total || binary.LittleEndian.Uint64(page[:]) != value {
					t.Fatalf("incorrect precise memory retirement: owned=%t cyclic=%t proceed=%d total=%d memory=%d status=%d value=%d", owned, cyclic, proceed, memory.Total, memory.MemoryTotal, memory.Status, binary.LittleEndian.Uint64(page[:]))
				}
			}
			n.Close()
		}
	}
}
