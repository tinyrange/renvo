package systemmemory

import (
	"renvo.dev/internal/rfe/engine"
	"testing"
)

func TestPageInvalidationScopesExecuteProofsButRevokesAllData(t *testing.T) {
	domain := 0
	root, high := uint64(11), uint64(22)
	m := New(make([]byte, 65536), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a & 65535, true, nil })
	m.EnablePageInvalidation()
	m.Domain = func() int { return domain }
	m.AddressSpacePair = func() (uint64, uint64) { return root, high }
	var before [2][3]engine.CodeStamp
	// Both roots have all three privilege views. The ordered upper root tag is
	// the only distinction; views 0/1 observed page1, view2 only page2.
	for r := 0; r < 2; r++ {
		high = uint64(22 + r)
		for d := 0; d < 3; d++ {
			domain = d
			if _, err := m.CodeVersion(uint64(1+d/2) * 4096); err != nil {
				t.Fatal(err)
			}
			before[r][d] = m.CodeContext()
			m.NativeRefill(3 * 4096)
		}
	}
	root, high, domain = 11, 22, 0
	m.InvalidatePage(4 * 4096) // never execute checked in either root
	for r := 0; r < 2; r++ {
		high = uint64(22 + r)
		for d := 0; d < 3; d++ {
			domain = d
			if got := m.CodeContext(); got != before[r][d] {
				t.Fatal("data-only eviction revoked code", r, d)
			}
		}
	}
	if m.filled != 0 {
		t.Fatal("data descriptors survived single-page invalidation")
	}
	root, high, domain = 11, 22, 0
	m.InvalidatePage(4096)
	for r := 0; r < 2; r++ {
		high = uint64(22 + r)
		for d := 0; d < 3; d++ {
			domain = d
			got := m.CodeContext()
			if changed := got != before[r][d]; changed != (r == 0 && d < 2) {
				t.Fatal("wrong execute-proof invalidation scope", r, d, changed)
			}
		}
	}
}

func TestPageInvalidationRetainsFilterAcrossEpochsAndTableWatches(t *testing.T) {
	root := uint64(7)
	m := New(make([]byte, 65536), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a & 65535, true, nil })
	m.EnablePageInvalidation()
	m.AddressSpace = func() uint64 { return root }
	if _, err := m.CodeVersion(4096); err != nil {
		t.Fatal(err)
	}
	m.MarkPageTableEntry(8192)
	m.BeforeWrite(8192, 8) // PTE update: architectural TLB may still retain old code.
	before := m.CodeContext()
	m.InvalidatePage(4096)
	if m.CodeContext() == before {
		t.Fatal("PTE write erased execute-page dependency")
	}
	before = m.CodeContext()
	m.BeforeWrite(8192, 8)
	if m.CodeContext() == before {
		t.Fatal("INVLPG dropped surviving table watch")
	}
	// Repeat through a global epoch change, not just a table-owner mutation.
	m.InvalidateMappings()
	before = m.CodeContext()
	m.InvalidatePage(4096)
	if m.CodeContext() == before {
		t.Fatal("global epoch change erased execute dependency")
	}
	// A deliberate filter collision is permitted to revoke, never to grant.
	before = m.CodeContext()
	m.InvalidatePage(4096 << 12)
	if m.CodeContext() == before {
		t.Fatal("filter collision lost a possible execute page")
	}
}

func TestPageInvalidationRootSlotReuseAndFallback(t *testing.T) {
	root := uint64(1)
	m := New(make([]byte, 65536), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a & 65535, true, nil })
	m.EnablePageInvalidation()
	m.AddressSpace = func() uint64 { return root }
	if _, err := m.CodeVersion(4096); err != nil {
		t.Fatal(err)
	}
	old := m.CodeContext()
	for root = 2; root <= 5; root++ {
		m.CodeContext()
	}
	root = 5
	fresh := m.CodeContext()
	if fresh == old {
		t.Fatal("slot reused old authority")
	}
	m.InvalidatePage(4096)
	if m.CodeContext() != fresh {
		t.Fatal("replaced root inherited former execute-page filter")
	}
	// An active-state change is not permission to skip revocation of inactive
	// cached roots. IRQ-shadow transitions can temporarily disable native entry.
	m.Active = func() bool { return false }
	if _, err := m.CodeVersion(4096); err == nil {
		t.Fatal("inactive execute admitted")
	}
	m.InvalidatePage(8192)
	m.Active = func() bool { return true }
	if m.CodeContext() != fresh {
		t.Fatal("unobserved page unexpectedly revoked")
	}
	if _, err := m.CodeVersion(4096); err != nil {
		t.Fatal(err)
	}
	m.Active = func() bool { return false }
	m.InvalidatePage(4096)
	m.Active = func() bool { return true }
	if m.CodeContext() == fresh {
		t.Fatal("inactive cached execute authority survived")
	}
	// Unscoped clients cannot use the per-root filter at all.
	m.AddressSpace = nil
	before := m.CodeContext()
	m.InvalidatePage(0xf000)
	if m.CodeContext() == before {
		t.Fatal("legacy client failed to revoke globally")
	}
}

func TestPageInvalidationEnableRevokesPreexistingProofs(t *testing.T) {
	root := uint64(1)
	m := New(make([]byte, 8192), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpace = func() uint64 { return root }
	if _, err := m.CodeVersion(4096); err != nil {
		t.Fatal(err)
	}
	old := m.CodeContext()
	m.EnablePageInvalidation()
	if m.CodeContext() == old {
		t.Fatal("opt-in inherited untracked authority")
	}
	fresh := m.CodeContext()
	m.EnablePageInvalidation()
	if m.CodeContext() != fresh {
		t.Fatal("repeated opt-in revoked tracked authority")
	}
	if _, err := m.CodeVersion(4096); err != nil {
		t.Fatal(err)
	}
	fresh = m.CodeContext()
	m.InvalidatePage(4096)
	if m.CodeContext() == fresh {
		t.Fatal("newly tracked proof survived page eviction")
	}
}
