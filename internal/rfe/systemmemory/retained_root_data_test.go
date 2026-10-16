package systemmemory

import "testing"

// One guest address denotes different physical bytes and permissions in each
// exact view. Selection may retain these facts, but cannot transfer authority.
// Ordinary writes are fresh through cached aliases; metadata writes revoke all.
func TestRetainedRootDataIsolationMutationAndBoundedReuse(t *testing.T) {
	domain, root := 0, uint64(10)
	data := make([]byte, 5*4096)
	data[4096], data[8192], data[12288] = 0x19, 0x2b, 0x3d
	remapped := false
	m := New(data, func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) {
		offset, writable := uint64(4096), true
		if root == 20 {
			offset, writable = 8192, false
		}
		if domain == 1 || root == 20 && remapped {
			offset, writable = 12288, true
		}
		return offset + a, writable, nil
	})
	m.RetainRootData = true
	m.Domain = func() int { return domain }
	m.AddressSpace = func() uint64 { return root }
	a := m.NativeContext()
	stamp := m.CodeContext()
	m.NativeRefill(0)
	root = 20
	m.SwitchAddressSpace()
	b := m.NativeContext()
	m.NativeRefill(0)
	domain, root = 1, 10
	c := m.NativeContext()
	m.NativeRefill(0)
	if a == b || a == c || b == c || a.Pages[0].Data == nil || b.Pages[0].Data == nil || c.Pages[0].Data == nil || a.Pages[0].Data[0] != 0x19 || b.Pages[0].Data[0] != 0x2b || c.Pages[0].Data[0] != 0x3d || a.Pages[0].Permissions != 3 || b.Pages[0].Permissions != 1 || c.Pages[0].Permissions != 3 {
		t.Fatal("root/domain data or permission isolation", a.Pages[0], b.Pages[0], c.Pages[0])
	}
	domain = 0
	if m.NativeContext() != a || m.CodeContext() != stamp {
		t.Fatal("unchanged root identity not retained")
	}
	m.BeforeWrite(4096, 1)
	data[4096] = 0x51
	if a.Pages[0].Data == nil || a.Pages[0].Data[0] != 0x51 || b.Pages[0].Data[0] != 0x2b {
		t.Fatal("cached data value or cross-root alias")
	}
	m.MarkPageTable(4 * 4096)
	if a.Pages[0].Data != nil || b.Pages[0].Data != nil || c.Pages[0].Data != nil {
		t.Fatal("new metadata barrier retained old writable aliases")
	}
	m.NativeRefill(0)
	root = 20
	m.NativeRefill(0)
	domain, root = 1, 10
	m.NativeRefill(0)
	m.BeforeWrite(4*4096, 8)
	remapped = true
	if a.Pages[0].Data != nil || b.Pages[0].Data != nil || c.Pages[0].Data != nil {
		t.Fatal("metadata write failed to revoke inactive views")
	}
	domain, root = 0, 20
	m.NativeRefill(0)
	if m.NativeContext() != b || b.Pages[0].Data == nil || b.Pages[0].Data[0] != 0x3d || b.Pages[0].Permissions != 3 {
		t.Fatal("remapping inherited prior permissions or physical backing")
	}
	root = 10
	if m.CodeContext() == stamp {
		t.Fatal("mapping mutation resurrected old code stamp")
	}
	before := m.CodeContext()
	for root = 30; root < 36; root++ {
		m.NativeContext()
		m.NativeRefill(0)
	}
	root = 10
	if m.CodeContext() == before || a.Pages[0].Data != nil || b.Pages[0].Data != nil {
		t.Fatal("bounded slot reuse resurrected cached authority")
	}
}

func TestRetainedRootDataRequiresIdentityAndExcludesDirectWindows(t *testing.T) {
	domain := 0
	m := New(make([]byte, 4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.RetainRootData = true
	m.Domain = func() int { return domain }
	a := m.NativeContext()
	m.NativeRefill(0)
	domain = 1
	m.NativeContext()
	if a.Pages[0].Data != nil {
		t.Fatal("retained data without exact root identity")
	}
	root := uint64(1)
	d, err := NewDirect(4096, func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if !d.DirectAvailable() {
		t.Skip("protected windows unavailable")
	}
	d.RetainRootData = true
	d.AddressSpace = func() uint64 { return root }
	old := d.NativeContext()
	d.NativeRefill(0)
	root = 2
	d.SwitchAddressSpace()
	d.NativeContext()
	if old.Pages[0].Data != nil {
		t.Fatal("protected direct root switch retained old data views")
	}
}
