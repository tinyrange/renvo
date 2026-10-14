package systemmemory

import "testing"

func TestRootIdentityRoundTripMutationAndBoundedReuse(t *testing.T) {
	root := uint64(1)
	m := New(make([]byte, 4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpace = func() uint64 { return root }
	first := m.CodeContext()
	m.NativeRefill(0)
	root = 2
	m.SwitchAddressSpace()
	second := m.CodeContext()
	if second.Identity == first.Identity || second.Epoch != first.Epoch || m.NativeContext().Pages[0].Data != nil {
		t.Fatal("root switch reused grants or mutated global epoch")
	}
	root = 1
	m.SwitchAddressSpace()
	if m.CodeContext() != first {
		t.Fatal("unchanged root round trip lost identity")
	}
	m.MarkPageTable(0)
	m.BeforeWrite(0, 8)
	if m.CodeContext() == first {
		t.Fatal("page-table mutation preserved stale root proof")
	}
	current := m.CodeContext()
	for root = 3; root < 9; root++ {
		m.SwitchAddressSpace()
		m.CodeContext()
	}
	root = 1
	m.SwitchAddressSpace()
	if m.CodeContext() == current {
		t.Fatal("bounded identity reuse resurrected old stamp")
	}
}
