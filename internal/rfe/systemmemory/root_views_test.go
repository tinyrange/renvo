package systemmemory

import "testing"

func TestRootViewsKeepIsolatedLinksAndRevokeData(t *testing.T) {
	root := uint64(1)
	m := New(make([]byte, 8192), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpace = func() uint64 { return root }
	a := m.NativeContext()
	a.PublishLink(0x1000, 16, 1, [17]uint8{})
	m.NativeRefill(0)
	root = 2
	m.SwitchAddressSpace()
	b := m.NativeContext()
	if a == b || b.LookupLink(0x1000) != nil || a.Pages[0].Data != nil {
		t.Fatal("new root inherited links or old data permissions")
	}
	b.PublishLink(0x2000, 32, 1, [17]uint8{})
	m.NativeRefill(4096)
	root = 1
	m.SwitchAddressSpace()
	if m.NativeContext() != a || a.LookupLink(0x1000) == nil || a.LookupLink(0x2000) != nil || b.Pages[1].Data != nil {
		t.Fatal("round trip lost root links or retained another root's data")
	}
	m.MarkPageTable(0)
	m.BeforeWrite(0, 8)
	if a.Pages[0].Data != nil || b.Pages[1].Data != nil {
		t.Fatal("physical table alias failed to revoke all views")
	}
}
