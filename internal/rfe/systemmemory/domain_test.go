package systemmemory

import "testing"

func TestNativeDomainsKeepLinkAuthoritySeparate(t *testing.T) {
	domain := 0
	m := New(make([]byte, 4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.Domain = func() int { return domain }
	user := m.NativeContext()
	user.PublishLink(0x1000, 16, 1, [17]uint8{})
	m.NativeRefill(0)
	domain = 1
	kernel := m.NativeContext()
	if kernel == user || kernel.LookupLink(0x1000) != nil || user.Pages[0].Data != nil {
		t.Fatal("domain reused native link authority or retained former data descriptors")
	}
	kernel.PublishLink(0x2000, 32, 1, [17]uint8{})
	m.NativeRefill(0)
	domain = 0
	if m.NativeContext() != user || user.LookupLink(0x1000) == nil || user.LookupLink(0x2000) != nil || kernel.Pages[0].Data != nil {
		t.Fatal("kernel round trip lost isolated user links or leaked descriptors")
	}
	m.NativeRefill(0)
	m.MarkPageTable(0)
	if user.Pages[0].Data != nil || kernel.Pages[0].Data != nil {
		t.Fatal("physical alias watch failed to revoke every context")
	}
}
