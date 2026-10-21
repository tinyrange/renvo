//go:build !renvo

package runtime

import (
	"encoding/binary"
	"renvo.dev/internal/rfeabi"
	hostruntime "runtime"
	"testing"
	"unsafe"
)

func ownedFixture(t *testing.T) (*Native, *MemoryContext, []byte) {
	t.Helper()
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { n.Close() })
	ram := make([]byte, 4*4096)
	clock := new(uint64)
	*clock = 1
	m := &MemoryContext{NativeContext: NativeContext{Clock: clock}}
	m.Memory = rfeabi.NewOwnedMemory(ram, clock, rfeabi.NewWriteWatches(len(ram)))
	return n, m, ram
}
func TestNativeOwnedMemoryLayoutAndCollidingMappings(t *testing.T) {
	n, m, ram := ownedFixture(t)
	var memory rfeabi.OwnedMemory
	if unsafe.Offsetof(memory.Base) != 0 || unsafe.Offsetof(memory.Mask) != 8 || unsafe.Offsetof(memory.Backing) != 16 || unsafe.Offsetof(memory.Bytes) != 24 || unsafe.Offsetof(memory.Watches) != 32 {
		t.Fatal("owned memory header")
	}
	for i := uint64(0); i < 512; i++ {
		m.Memory.Fill(1+i*64, 0, 3)
	}
	binary.LittleEndian.PutUint64(ram, 0x123456789abcdef0)
	var b Builder
	b.Store(1, b.MemoryLoad(b.Load(0), 8))
	b.Checkpoint(2, 1)
	e, err := n.CompileMemory(b.FinishMemory(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	for i := uint64(0); i < 512; i++ {
		state := []uint64{(1 + i*64) * 4096, 0}
		if err = n.CallMemory(e, state, m); err != nil || m.Status != 0 || state[1] != 0x123456789abcdef0 {
			t.Fatal("colliding native mapping", i, state, m.Status, err)
		}
	}
	m.Memory.Forget(65)
	if err = n.CallMemory(e, []uint64{65 * 4096, 0}, m); err != nil || m.Status != 1 {
		t.Fatal("revoked mapping", m.Status, err)
	}
	if err = n.CallMemory(e, []uint64{129 * 4096, 0}, m); err != nil || m.Status != 0 {
		t.Fatal("revocation evicted neighbor", m.Status, err)
	}
}
func TestNativeOwnedStoresCheckPhysicalBytesOnEveryLoopIteration(t *testing.T) {
	n, m, ram := ownedFixture(t)
	m.Memory.Fill(0, 0, 7)
	m.Memory.Fill(64, 0, 7)
	m.Memory.Watches.Mark(17, 1, 1)
	m.Memory.Watches.Mark(32, 4, 2)
	var b Builder
	b.MemoryStore(b.Load(0), b.Constant(0x3344), 2)
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(2)))
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
	m.PublishLink(0, entry, 1, [17]uint8{0, 1})
	for _, start := range []uint64{0, 64 * 4096} {
		clear(ram)
		state := []uint64{start, 0}
		if err = n.RunLinkedBackingSession(state, m, 128, ram); err != nil {
			t.Fatal(err)
		}
		if m.Status != 1 || m.Total != 8 || state[0] != start+16 || m.Address != start+16 {
			t.Fatal("cached page fact bypassed byte watch", state, m.Status, m.Total, m.Address)
		}
		for j := 0; j < 8; j++ {
			if binary.LittleEndian.Uint16(ram[j*2:]) != 0x3344 {
				t.Fatal("lost store prefix", j)
			}
		}
		if binary.LittleEndian.Uint16(ram[16:]) != 0 {
			t.Fatal("code byte modified")
		}
		hostruntime.GC()
	}
	var one Builder
	one.MemoryStore(one.Load(0), one.Constant(0x7788), 2)
	one.Checkpoint(2, 1)
	leaf, err := n.CompileMemory(one.FinishMemory(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		a      uint64
		status uint64
	}{{18, 0}, {31, 1}, {36, 0}, {4095, 1}} {
		if err = n.CallMemory(leaf, []uint64{test.a, 0}, m); err != nil || m.Status != test.status {
			t.Fatal("leaf byte barrier", test, m.Status, err)
		}
	}
}
