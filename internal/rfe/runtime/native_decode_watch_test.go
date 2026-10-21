//go:build !renvo

package runtime

import (
	"encoding/binary"
	"testing"
	"unsafe"
)

func TestNativeDecodedOnlyStoresVersionPhysicalAliasesAndFailClosed(t *testing.T) {
	for _, loop := range []bool{false, true} {
		for _, size := range []int{1, 2, 4, 8, 16} {
			n, m, ram := ownedFixture(t)
			if unsafe.Offsetof(m.Memory.DecodeEpochs) != 56 {
				t.Fatal("decode epoch ABI")
			}
			versions := make([]uint64, len(ram)/4096)
			m.Memory.SetDecodeEpochs(versions)
			m.Memory.Fill(0, 0, 7)
			m.Memory.Fill(64, 0, 7)
			m.Memory.Watches.Mark(0, len(ram), 4)
			var b Builder
			value := b.Constant(0x123456789abcdef0)
			if size == 16 {
				b.MemoryPairStore(b.Load(0), value, value)
			} else {
				b.MemoryStore(b.Load(0), value, size)
			}
			b.Store(0, b.Binary(Add, b.Load(0), b.Constant(uint64(size))))
			b.Store(1, b.Constant(0))
			b.Checkpoint(2, 1)
			var entry int
			var err error
			if loop {
				b.LoopContinue(b.Constant(1))
				entry, err = n.CompileCodeVersionedLoop(b.FinishMemory(2), 2, 1, m)
			} else {
				entry, err = n.CompileMemory(b.FinishMemory(2), 2)
			}
			if err != nil {
				t.Fatal(err)
			}
			if loop {
				if err = n.PrepareLinks(2, 1); err != nil {
					t.Fatal(err)
				}
				if err = n.PrepareTargetLink(0, entry, 2, 1); err != nil {
					t.Fatal(err)
				}
				m.ClaimLinks(n)
				m.PublishLink(0, entry, 1, [17]uint8{0, 1})
			}
			run := func(address uint64) ([]uint64, error) {
				state := []uint64{address, 0}
				var e error
				if loop {
					e = n.RunLinkedBackingSession(state, m, 4, ram)
				} else {
					e = n.CallMemory(entry, state, m)
				}
				return state, e
			}
			for _, alias := range []uint64{0, 64 * 4096} {
				clear(ram)
				before := *m.Clock
				state, err := run(alias)
				count := uint64(1)
				if loop {
					count = 4
				}
				if err != nil || m.Status != 0 || state[0] != alias+count*uint64(size) || versions[0] <= before || versions[0] > *m.Clock || *m.Clock < before+count || *m.Clock > before+2*count {
					t.Fatal("decode-only write was not native/coherent", loop, size, state, m.Status, *m.Clock, versions, err)
				}
				if ram[0] != 0xf0 {
					t.Fatal("lost physical store")
				}
			}
			// Live code, PTEs, unknown bits and missing version owners always exit
			// before any guest byte or epoch mutation, including cached loop facts.
			for _, flag := range []byte{1, 2, 8} {
				m.Memory.Watches.Mark(0, size, flag)
				clear(ram)
				before := *m.Clock
				state, err := run(64 * 4096)
				if err != nil || m.Status != 1 || state[0] != 64*4096 || *m.Clock != before || binary.LittleEndian.Uint64(ram) != 0 {
					t.Fatal("protected watch bypass", loop, size, flag, state, m.Status, err)
				}
				m.Memory.Watches.Unmark(0, size, flag)
			}
			*m.Clock = ^uint64(0)
			clear(ram)
			before := versions[0]
			state, err := run(0)
			if err != nil || m.Status != 1 || state[0] != 0 || *m.Clock != ^uint64(0) || versions[0] != before || ram[0] != 0 {
				t.Fatal("epoch overflow committed store", loop, size, state, m.Status, err)
			}
			n.Close()
		}
	}
}
