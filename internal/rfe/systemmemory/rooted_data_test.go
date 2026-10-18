package systemmemory

import (
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

// Independent physical frames and asymmetric read/write permissions expose
// accidental cross-domain/root reuse. All mutation barriers must still revoke
// inactive views, including a different root that has not run since the store.
func TestRootedDataRoundTripsIsolatedViewsAndRevokesAllMutations(t *testing.T) {
	root, domain := uint64(1), 0
	frame := [2][2]uint64{{0, 4096}, {8192, 12288}}
	m := New(make([]byte, 5*4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) {
		return frame[domain][root-1] + a, domain == 1 || root == 1, nil
	})
	m.AddressSpacePair = func() (uint64, uint64) { return 1<<48 | root, 1<<52 | 7 }
	m.Domain = func() int { return domain }
	if !m.EnableRootedData() {
		t.Fatal("complete rooted owner rejected")
	}
	var contexts [2][2]*emu.MemoryContext
	for domain = 0; domain < 2; domain++ {
		for root = 1; root <= 2; root++ {
			m.SwitchAddressSpace()
			contexts[domain][root-1] = m.NativeContext()
			m.MarkPageTableEntry(4*4096 + uint64(domain*2+int(root)-1)*8)
			m.NativeRefill(0)
		}
	}
	for repeat := 0; repeat < 3; repeat++ {
		for domain = 0; domain < 2; domain++ {
			for root = 1; root <= 2; root++ {
				m.SelectPrivilege()
				m.SwitchAddressSpace()
				ctx := m.NativeContext()
				page := ctx.PageSlot(0)
				permissions := uint64(3)
				if domain == 0 && root == 2 {
					permissions = 1
				}
				if ctx != contexts[domain][root-1] || page.Data == nil || &page.Data[0] != &m.Data[frame[domain][root-1]] || page.Permissions != permissions {
					t.Fatal("root/domain round trip lost or shared mapping", domain, root, repeat)
				}
			}
		}
	}
	checkRevoked := func(label string) {
		for _, row := range contexts {
			for _, ctx := range row {
				if ctx.PageSlot(0).Data != nil {
					t.Fatal(label, "retained inactive mapping")
				}
			}
		}
	}
	domain, root = 1, 2
	m.BeforeWrite(4*4096, 8) // root 1/domain 0's PTE through another privilege view
	checkRevoked("PTE alias store")
	frame[0][0] = 4096
	domain, root = 0, 1
	m.NativeRefill(0)
	if &m.NativeContext().PageSlot(0).Data[0] != &m.Data[4096] {
		t.Fatal("PTE remap used stale backing")
	}
	for _, mutate := range []func(){m.InvalidateMappings, func() { m.BeforePageTableMetadata(4*4096, 8) }, func() { m.InvalidatePage(0) }, m.ForgetData} {
		for domain = 0; domain < 2; domain++ {
			for root = 1; root <= 2; root++ {
				m.NativeRefill(0)
			}
		}
		mutate()
		checkRevoked("mapping/permission/walker/explicit revocation")
	}
}

func TestRootedDataRequiresExplicitOwnerAndFifthRootRevokesSlots(t *testing.T) {
	root, domain := uint64(1), 0
	m := New(make([]byte, 4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	if m.EnableRootedData() {
		t.Fatal("missing root/domain owner admitted")
	}
	m.AddressSpacePair = func() (uint64, uint64) { return root, root << 40 }
	if m.EnableRootedData() {
		t.Fatal("missing privilege owner admitted")
	}
	m.Domain = func() int { return domain }
	if !m.EnableRootedData() {
		t.Fatal("root/domain owner rejected")
	}
	first := m.NativeContext()
	m.NativeRefill(0)
	stamp := m.CodeContext()
	for root = 2; root <= 5; root++ {
		m.NativeContext()
	}
	if first.PageSlot(0).Data != nil {
		t.Fatal("fifth root resurrected old data slot")
	}
	root = 1
	if m.CodeContext() == stamp {
		t.Fatal("fifth root reused code authority")
	}
	m.NativeRefill(0)
	m.Domain = nil // loss of any required owner callback restores revocation
	m.SelectPrivilege()
	if m.NativeContext().PageSlot(0).Data != nil {
		t.Fatal("lost owner retained permissions")
	}
}
