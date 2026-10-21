//go:build !renvo

package runtime

import (
	"encoding/binary"
	"renvo.dev/internal/rfeabi"
	"testing"
)

func TestNativeWideBodyPrivateFrameAndPreciseFaultPrefix(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("wide foreign sessions unavailable on this host")
	}
	const count = 7000
	const output = 65536
	n, err := NewNative(32 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	ram := make([]byte, 32*4096)
	clock := uint64(1)
	m := &MemoryContext{NativeContext: NativeContext{Clock: &clock}}
	m.Memory = rfeabi.NewOwnedMemory(ram, &clock, rfeabi.NewWriteWatches(len(ram)))
	for p := 0; p < len(ram)/4096; p++ {
		m.Memory.Fill(uint64(p), uint64(p*4096), 3)
	}
	b := Builder{MemoryInstructionProgress: true}
	values := make([]Value, count)
	for i := range values {
		binary.LittleEndian.PutUint64(ram[i*8:], uint64(i*17+3))
		b.Store(1, b.Constant(uint64(i*4)))
		b.Checkpoint(2, i)
		values[i] = b.MemoryLoad(b.Constant(uint64(i*8)), 8)
	}
	for i, v := range values {
		b.Store(1, b.Constant(uint64((count+i)*4)))
		b.Checkpoint(2, count+i)
		b.MemoryStore(b.Constant(output+uint64(i*8)), v, 8)
	}
	b.Store(1, b.Constant(4))
	b.Checkpoint(2, count*2)
	b.LoopContinue(b.Constant(0))
	ops := b.FinishMemory(2)
	if len(ops) <= 2048 {
		t.Fatal("test did not exceed old record limit")
	}
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	entry, err := n.CompileCodeVersionedLoop(ops, 2, count*2, m)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(0, entry, 2, count*2); err != nil {
		t.Fatal(err)
	}
	m.ClaimLinks(n)
	m.PublishLink(0, entry, count*2, [17]uint8{})
	state := []uint64{0, 0}
	if err = n.RunLinkedBackingSession(state, m, count*2, ram); err != nil {
		t.Fatal(err)
	}
	if m.Total != count*2 || m.MemoryTotal != count*2 || m.Status != 0 || state[1] != 4 {
		t.Fatal("wide completion mismatch", m.Total, m.MemoryTotal, m.Status, state)
	}
	for i := range values {
		if binary.LittleEndian.Uint64(ram[output+i*8:]) != uint64(i*17+3) {
			t.Fatal("private-frame spill lost live value", i)
		}
	}
	clear(ram[output:])
	m.Memory.Forget(5)
	state[1] = 0
	if err = n.RunLinkedBackingSession(state, m, count*2, ram); err != nil {
		t.Fatal(err)
	}
	if m.Total != 5*512 || m.MemoryTotal != 5*512 || m.Status != 1 || m.Address != 5*4096 || state[1] != 5*512*4 {
		t.Fatal("wide fault prefix mismatch", m.Total, m.MemoryTotal, m.Status, m.Address, state)
	}
	for _, v := range ram[output:] {
		if v != 0 {
			t.Fatal("fault executed later output store")
		}
	}
}
