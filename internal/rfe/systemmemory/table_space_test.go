package systemmemory

import "testing"

func TestTableWatchesFollowBoundedRootProofLifetime(t *testing.T) {
	space := uint64(1)
	m := New(make([]byte, 7*4096), func() bool { return true }, func(a uint64, exec bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpace = func() uint64 { return space }
	m.MarkPageTable(0)
	m.MarkPageTable(4096) // Shared with root 2; retiring root 1 alone is insufficient.
	first := m.CodeContext()
	for root := uint64(2); root <= 5; root++ {
		space = root
		m.SwitchAddressSpace()
		m.MarkPageTable(root * 4096)
		if root == 2 {
			m.MarkPageTable(4096)
		}
		m.CodeContext()
	}
	before := m.CodeContext()
	m.BeforeWrite(0, 1)
	if m.CodeContext() != before {
		t.Fatal("evicted root kept retired table barrier")
	}
	m.NativeRefill(4096)
	if m.NativeContext().Pages[1].Permissions&4 == 0 {
		t.Fatal("shared live root lost table protection")
	}
	space = 6
	m.SwitchAddressSpace()
	m.MarkPageTable(6 * 4096)
	before = m.CodeContext()
	m.BeforeWrite(4096, 1)
	if m.CodeContext() != before {
		t.Fatal("last evicted owner kept retired table barrier")
	}
	space = 1
	m.SwitchAddressSpace()
	if m.CodeContext() == first {
		t.Fatal("root identity reuse resurrected old native proof")
	}
}

func TestRootWatchRetirementPreservesOtherDomainAndBoundsMetadata(t *testing.T) {
	space, domain := uint64(1), 0
	m := New(make([]byte, 4096), func() bool { return true }, func(a uint64, exec bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpace = func() uint64 { return space }
	m.Domain = func() int { return domain }
	m.MarkPageTable(0)
	domain = 1
	m.MarkPageTable(0)
	domain = 0
	for root := uint64(2); root < 1000; root++ {
		space = root
		m.SwitchAddressSpace()
		m.MarkPageTable(0)
	}
	// One global entry per physical frame, not one entry per reused root.
	if len(m.tablePages) > len(m.watched) {
		t.Fatal("root churn grew watch metadata without bound", len(m.tablePages))
	}
	domain, space = 1, 1
	m.NativeRefill(0)
	if m.NativeContext().Pages[0].Permissions&4 == 0 {
		t.Fatal("another domain's live table proof lost its barrier")
	}
}
