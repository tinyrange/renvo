//go:build !renvo

package runtime

import "testing"

// Successful checks dominate only the identical immutable value. A later
// load and failed guard must keep their own precise architectural prefix.
func TestNativeDominatingGuardDoesNotProveNewMemoryValues(t *testing.T) {
	var b Builder
	v := b.MemoryLoad(b.Load(0), 8)
	c := b.Binary(Equal, v, b.Constant(17))
	b.Checkpoint(3, 1)
	b.Guard(c)
	b.MemoryStore(b.Load(0), b.Constant(23), 8)
	b.Checkpoint(3, 2)
	b.Guard(c)
	v2 := b.MemoryLoad(b.Load(0), 8)
	b.Checkpoint(3, 3)
	b.Guard(b.Binary(Equal, v2, b.Constant(17)))
	b.Store(1, b.Constant(99))
	ops := b.FinishMemory(3)
	page := new([4096]byte)
	page[0] = 17
	epoch := uint64(1)
	m := new(MemoryContext)
	m.Clock = &epoch
	m.Fill(1, page, 3, &epoch)
	state := []uint64{4096, 11, 0}
	n, e := NewNative(1 << 20)
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	entry, e := n.CompileMemory(ops, 3)
	if e != nil {
		t.Fatal(e)
	}
	if e := n.CallMemory(entry, state, m); e != nil {
		t.Fatal(e)
	}
	if m.Status != 2 || m.Retired != 3 || state[1] != 11 || page[0] != 23 {
		t.Fatal("guard/value prefix", m.Status, m.Retired, state, page[0])
	}
}
