package systemmemory

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

func TestFastRefillNoWalkAndExactRootContextOwnership(t *testing.T) {
	root, high, domain, active := uint64(7), uint64(9), 0, true
	m := New(make([]byte, 8192), func() bool { return active }, func(uint64, bool) (uint64, bool, error) {
		t.Fatal("fast refill invoked walking resolver")
		return 0, false, nil
	})
	m.AddressSpacePair = func() (uint64, uint64) { return root, high }
	m.Domain = func() int { return domain }
	calls := 0
	m.CachedMapping = func(a uint64, write bool) (uint64, bool, bool) { calls++; return a, true, true }
	stamp, ctx := m.NativeCodeContext()
	ctx.Status, ctx.Address = 1, 16
	copied := *ctx
	var wrongClock uint64
	other := &emu.MemoryContext{}
	for _, bad := range []*emu.MemoryContext{&copied, other} {
		if m.NativeFastRefill(bad) {
			t.Fatal("copied/foreign context admitted")
		}
	}
	ctx.Clock = &wrongClock
	if m.NativeFastRefill(ctx) {
		t.Fatal("foreign clock admitted")
	}
	ctx.Clock = &m.clock
	slots, next, clock := m.spaces, m.spaceNext, m.clock
	root = 8
	if m.NativeFastRefill(ctx) {
		t.Fatal("unobserved root admitted")
	}
	root = 7
	high = 10
	if m.NativeFastRefill(ctx) {
		t.Fatal("upper root component ignored")
	}
	high = 9
	domain = 1
	if m.NativeFastRefill(ctx) {
		t.Fatal("foreign domain admitted")
	}
	domain = 0
	active = false
	if m.NativeFastRefill(ctx) {
		t.Fatal("inactive CPU admitted")
	}
	active = true
	if calls != 0 || m.spaceNext != next || m.clock != clock {
		t.Fatal("rejection mutated root state or queried mappings")
	}
	for d := range slots {
		for s := range slots[d] {
			if slots[d][s].valid != m.spaces[d][s].valid || slots[d][s].epoch != m.spaces[d][s].epoch {
				t.Fatal("rejection created/changed root slot")
			}
		}
	}
	if !m.NativeFastRefill(ctx) || calls != 1 || ctx.PageSlot(0).Data == nil || ctx.PageSlot(0).Permissions != 3 || m.CodeContext() != stamp {
		t.Fatal("existing cached RAM refill failed")
	}
	m.ForgetData()
	if ctx.PageSlot(0).Data != nil {
		t.Fatal("fast descriptor escaped ordinary revocation")
	}
}

func TestFastRefillReadOnlyWatchedWriteAndExtentGuards(t *testing.T) {
	m := New(make([]byte, 8192+8), func() bool { return true }, func(uint64, bool) (uint64, bool, error) { t.Fatal("walk"); return 0, false, nil })
	_, ctx := m.NativeCodeContext()
	writable, present := false, true
	offset := uint64(0)
	m.CachedMapping = func(a uint64, write bool) (uint64, bool, bool) { return offset | a&4095, writable, present }
	ctx.Status, ctx.Address = 1, 8
	if !m.NativeFastRefill(ctx) || ctx.PageSlot(0).Permissions != 1 {
		t.Fatal("cached read-only read rejected or writable")
	}
	m.ForgetData()
	ctx.RefillAccess = 1
	if m.NativeFastRefill(ctx) || ctx.PageSlot(0).Data != nil {
		t.Fatal("read-only write admitted")
	}
	writable = true
	m.MarkPageTable(0)
	if m.NativeFastRefill(ctx) {
		t.Fatal("page-table store barrier bypassed")
	}
	ctx.RefillAccess = 0
	if !m.NativeFastRefill(ctx) || ctx.PageSlot(0).Permissions != 7 {
		t.Fatal("watched read lost barrier metadata")
	}
	m.ForgetData()
	ctx.RefillAccess = 1
	m.watched[0] = 1
	if m.NativeFastRefill(ctx) {
		t.Fatal("executable alias store admitted")
	}
	m.watched[0] = 0
	for _, a := range []uint64{4081, 4095} {
		ctx.Address = a
		if m.NativeFastRefill(ctx) {
			t.Fatal("near-boundary retry admitted", a)
		}
	}
	ctx.Address = 8
	present = false
	if m.NativeFastRefill(ctx) {
		t.Fatal("TLB miss admitted")
	}
	present = true
	offset = 8192
	if m.NativeFastRefill(ctx) {
		t.Fatal("partial final RAM page admitted")
	}
	offset = 12288
	if m.NativeFastRefill(ctx) {
		t.Fatal("device/non-RAM offset admitted")
	}
	offset = 0
	ctx.Status = 2
	if m.NativeFastRefill(ctx) {
		t.Fatal("nonmemory guard admitted")
	}
}
