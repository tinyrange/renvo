package systemmemory

import "testing"

func TestExactTableEntryWatchesKeepUnrelatedEntriesOutOfCodeEpoch(t *testing.T) {
	root, domain := uint64(1), 1
	m := New(make([]byte, 3*4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpace = func() uint64 { return root }
	m.Domain = func() int { return domain }
	m.MarkPageTableEntry(8)
	first := m.CodeContext()
	m.NativeRefill(0)
	ctx := m.NativeContext()
	if ctx.Pages[0].Permissions&4 == 0 {
		t.Fatal("entry filter removed full physical store barrier")
	}
	m.BeforeWrite(16, 8)
	if m.CodeContext() != first || ctx.Pages[0].Data != nil {
		t.Fatal("unrelated entry changed code epoch or retained data grant")
	}
	// Partial writes overlapping either end of a watched entry must invalidate.
	for _, write := range [][2]int{{8, 1}, {15, 1}, {7, 2}, {15, 2}} {
		before := m.CodeContext()
		m.BeforeWrite(uint64(write[0]), write[1])
		if m.CodeContext() == before {
			t.Fatal("partial PTE overlap retained code proof", write)
		}
	}
	domain = 0
	inherited := m.CodeContext()
	m.BeforeWrite(8, 8)
	if m.CodeContext() == inherited {
		t.Fatal("shared TLB entry dependency was not inherited")
	}
	// Legacy whole-page callers always take the conservative path.
	m.MarkPageTable(4096)
	before := m.CodeContext()
	m.BeforeWrite(4096+32, 8)
	if m.CodeContext() == before {
		t.Fatal("legacy page watch narrowed to one entry")
	}
	// Full TLB revocation clears both the dependency set and page barrier.
	m.ClearPageTableWatches()
	m.MarkPageTableEntry(16)
	before = m.CodeContext()
	m.BeforeWrite(8, 8)
	if m.CodeContext() != before {
		t.Fatal("retired exact entry kept its dependency bit")
	}
	m.BeforeWrite(16, 8)
	if m.CodeContext() == before {
		t.Fatal("new exact entry lost its dependency bit")
	}
}

func TestExactTableEntrySharedOwnersAndEpochExhaustion(t *testing.T) {
	root := uint64(1)
	m := New(make([]byte, 4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpace = func() uint64 { return root }
	m.MarkPageTableEntry(8)
	first := m.CodeContext()
	root = 2
	m.MarkPageTableEntry(16)
	second := m.CodeContext()
	m.BeforeWrite(8, 8)
	if m.CodeContext() != second {
		t.Fatal("independent exact root's PTE invalidated")
	}
	root = 1
	if m.CodeContext() == first {
		t.Fatal("inactive exact PTE owner preserved")
	}
	m.clock = ^uint64(0)
	m.BeforeWrite(8, 8)
	if m.CodeContext().Identity != nil || m.NativeContext() != nil {
		t.Fatal("PTE mutation wrapped exhausted epoch")
	}
}
