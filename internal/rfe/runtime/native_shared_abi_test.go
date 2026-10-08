//go:build !renvo

package runtime

import (
	"encoding/binary"
	"reflect"
	"renvo.dev/internal/backendcompiled"
	"runtime"
	"testing"
)

// Shared pure spills must not overwrite the context slot consumed by a later
// memory/loop body. The same compiled image must also work through standalone
// compatibility entries, and budget exits must leave complete architecture.
func TestNativeSharedABIMixedEntriesSpillsAndBudgets(t *testing.T) {
	const words = 28
	n, err := NewNative(2 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(words, 27); err != nil {
		t.Fatal(err)
	}
	var p Builder
	inputs := make([]Value, 24)
	for i := range inputs {
		inputs[i] = p.Load(i)
	}
	for i, v := range inputs {
		p.Store(i, p.Binary(Add, v, p.Constant(uint64(i+1))))
	}
	p.Store(27, p.Constant(4))
	pure, err := n.Compile(p.Finish(words), words)
	if err != nil {
		t.Fatal(err)
	}
	var mem Builder
	live := make([]Value, 24)
	for i := range live {
		live[i] = mem.Binary(Mul, mem.Load(i), mem.Constant(uint64(i*2+3)))
	}
	mem.MemoryStore(mem.Constant(4096), mem.Load(0), 8)
	mem.Store(27, mem.Constant(8))
	mem.Checkpoint(words, 1)
	sum := mem.MemoryLoad(mem.Constant(4096), 8)
	for _, v := range live {
		sum = mem.Binary(Xor, sum, v)
	}
	mem.Store(24, sum)
	mem.Checkpoint(words, 2)
	mem.MemoryLoad(mem.Constant(4096), 1)
	mem.Checkpoint(words, 3)
	memory, err := n.CompileMemory(mem.FinishMemory(words), words)
	if err != nil {
		t.Fatal(err)
	}
	var loop Builder
	left := loop.Binary(Sub, loop.Load(25), loop.Constant(1))
	loop.Store(25, left)
	loop.Store(24, loop.Binary(Add, loop.Load(24), loop.Constant(7)))
	cond := loop.Binary(Equal, loop.Binary(Equal, left, loop.Constant(0)), loop.Constant(0))
	loop.Store(27, loop.Choose(cond, loop.Constant(8), loop.Constant(12)))
	loop.Checkpoint(words, 2)
	loop.LoopContinue(cond)
	region, err := n.CompileLoop(loop.FinishMemory(words), words, 2)
	if err != nil {
		t.Fatal(err)
	}
	// Keep a genuinely legacy framed entry in this chain, not a new wrapper.
	var last Builder
	last.Store(24, last.Binary(Add, last.Load(24), last.Constant(9)))
	last.Store(27, last.Constant(16))
	ops := last.Finish(words)
	records := make([]int, 0, len(ops)*4)
	for _, op := range ops {
		records = append(records, op.Kind, int(op.A), int(op.B), int(op.Imm))
	}
	code, ok := backendcompiled.RenvoEmitPureBlock(records, words, runtime.GOARCH == "arm64")
	if !ok {
		t.Fatal("legacy emission")
	}
	legacy, err := n.arena.InstallBlock(code, words)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range [][3]int{{pure, words, 1}, {memory, words, 3}, {region, words, 2}, {legacy, words, 1}} {
		if err = n.AdmitLink(entry[0], entry[1], entry[2]); err != nil {
			t.Fatal(err)
		}
	}
	// Exercise both generic admissions and every prepared family continuation,
	// including the legacy framed leaf which clobbers the state-base scratch.
	for _, prepared := range []bool{false, true} {
		if prepared {
			for _, target := range [][3]int{{0, pure, 1}, {4, memory, 3}, {8, region, 2}, {12, legacy, 1}} {
				if err = n.PrepareTargetLink(uint64(target[0]), target[1], words, target[2]); err != nil {
					t.Fatal(err)
				}
			}
		}
		for trial := uint64(0); trial < 8; trial++ {
			for _, budget := range []uint64{1, 2, 4, 5, 9, 10, 11, 64} {
				state := make([]uint64, words)
				want := make([]uint64, words)
				for i := 0; i < 24; i++ {
					state[i] = trial*101 + uint64(i*17)
					want[i] = state[i] + uint64(i+1)
				}
				state[25], state[26] = 3, 999
				want[25], want[26], want[27] = 3, 999, 4
				expectedTotal, expectedMemory, iterations := uint64(1), uint64(0), uint64(0)
				if budget >= 4 {
					want[24] = want[0]
					for i := 0; i < 24; i++ {
						want[24] ^= want[i] * uint64(i*2+3)
					}
					want[27] = 8
					expectedTotal, expectedMemory = 4, 3
					iterations = (budget - 4) / 2
					if iterations > 3 {
						iterations = 3
					}
					want[24] += iterations * 7
					want[25] -= iterations
					expectedTotal += iterations * 2
					if iterations == 3 {
						want[27] = 12
					}
					if budget >= 11 {
						want[24] += 9
						want[27] = 16
						expectedTotal++
					}
				}
				page := new([4096]byte)
				clock, epoch := uint64(17), uint64(17)
				m := &MemoryContext{Clock: &clock}
				m.Fill(1, page, 3, &epoch)
				m.ClaimLinks(n)
				m.PublishLink(0, pure, 1, [17]uint8{})
				m.PublishLink(4, memory, 3, [17]uint8{0, 1, 2, 3})
				m.PublishLink(8, region, 2, [17]uint8{})
				m.PublishLink(12, legacy, 1, [17]uint8{})
				if err = n.CallLinked(state, m, budget); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(state, want) || m.Total != expectedTotal || m.MemoryTotal != expectedMemory || m.Remaining != budget-expectedTotal || m.LoopIterations != iterations || m.Status != 0 || m.CodeView != [4]uint64{} {
					t.Fatal("mixed shared/legacy ABI", trial, budget, state, want, m.Total, m.MemoryTotal, m.Remaining, m.Status)
				}
				if budget >= 4 {
					if binary.LittleEndian.Uint64(page[:]) != want[0] || clock != 18 || epoch != 18 {
						t.Fatal("shared context corrupted", budget, page[:8], clock, epoch)
					}
				} else if clock != 17 || epoch != 17 {
					t.Fatal("unexecuted memory changed", budget, clock, epoch)
				}
				if budget == 64 {
					direct := make([]uint64, words)
					for i := 0; i < 24; i++ {
						direct[i] = trial*101 + uint64(i*17)
					}
					direct[25], direct[26] = 3, 999
					directPage := new([4096]byte)
					directClock, directEpoch := uint64(17), uint64(17)
					dm := &MemoryContext{Clock: &directClock}
					dm.Fill(1, directPage, 3, &directEpoch)
					dm.ClaimLinks(n)
					dm.PublishLink(8, region, 2, [17]uint8{})
					dm.PublishLink(12, legacy, 1, [17]uint8{})
					if err = n.Call(pure, direct); err != nil {
						t.Fatal(err)
					}
					if err = n.CallMemory(memory, direct, dm); err != nil {
						t.Fatal(err)
					}
					if err = n.CallLinked(direct, dm, 64); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(direct, want) || *directPage != *page || directClock != clock || directEpoch != epoch {
						t.Fatal("standalone/shared mismatch", direct, want)
					}
				}
			}
		}
	}
}

func TestNativeSharedABILargeScratchThenMemory(t *testing.T) {
	n, err := NewNative(2 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 1); err != nil {
		t.Fatal(err)
	}
	var p Builder
	v := p.Load(0)
	values := make([]Value, 800)
	for i := range values {
		v = p.Binary(Add, v, p.Constant(1))
		values[i] = v
	}
	sum := p.Constant(0)
	for i := len(values) - 1; i >= 0; i-- {
		sum = p.Binary(Add, sum, values[i])
	}
	p.Store(2, sum)
	p.Store(1, p.Constant(4))
	pure, err := n.Compile(p.Finish(3), 3)
	if err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.MemoryStore(b.Constant(4096), b.Load(2), 8)
	b.Checkpoint(3, 1)
	b.Store(0, b.MemoryLoad(b.Constant(4096), 8))
	b.Store(1, b.Constant(8))
	b.Checkpoint(3, 2)
	memory, err := n.CompileMemory(b.FinishMemory(3), 3)
	if err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, pure, 1, [17]uint8{})
	m.PublishLink(4, memory, 2, [17]uint8{0, 1, 2})
	page := new([4096]byte)
	clock, epoch := uint64(1), uint64(1)
	m.Clock = &clock
	m.Fill(1, page, 3, &epoch)
	for input := uint64(0); input < 17; input++ {
		state := []uint64{input, 0, 999}
		want := 800*input + 800*801/2
		if err = n.CallLinked(state, m, 3); err != nil || state[0] != want || state[2] != want || state[1] != 8 || m.Total != 3 || m.MemoryTotal != 2 || binary.LittleEndian.Uint64(page[:]) != want || clock != input+2 || epoch != clock {
			t.Fatal("large shared scratch/context", err, state, want, m.Total, m.MemoryTotal, clock, epoch)
		}
	}
}

// The original leaf ABI allowed R11/X11 to be scratch. A compatibility entry
// that clears it must not poison the state base inherited by its shared successor.
func TestNativeSharedABILegacyStateBaseClobber(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	code := []byte{0x48, 0xc7, 0x40, 0x08, 4, 0, 0, 0, 0x4d, 0x31, 0xdb, 0xc3}
	if runtime.GOARCH == "arm64" {
		code = nil
		for _, op := range []uint32{0xd2800082, 0xf9000402, 0xaa1f03eb, 0xd65f03c0} {
			code = binary.LittleEndian.AppendUint32(code, op)
		}
	}
	legacy, err := n.arena.InstallBlock(code, 2)
	if err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(7)))
	b.Store(1, b.Constant(8))
	shared, err := n.Compile(b.Finish(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, legacy, 1, [17]uint8{})
	m.PublishLink(4, shared, 1, [17]uint8{})
	state := []uint64{11, 0}
	if err = n.CallLinked(state, m, 2); err != nil || state[0] != 18 || state[1] != 8 || m.Total != 2 {
		t.Fatal("legacy scratch clobber leaked", err, state, m.Total)
	}
}
