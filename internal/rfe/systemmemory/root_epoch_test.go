package systemmemory

import (
	"renvo.dev/internal/rfe/engine"
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

func TestRootSlotReplacementRevokesOnlyEvictedIdentity(t *testing.T) {
	root, domain := uint64(1), 0
	m := New(make([]byte, 4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpacePair = func() (uint64, uint64) { return 7, root }
	m.Domain = func() int { return domain }
	var stamps [2][4]engine.CodeStamp
	var contexts [2][4]*emu.MemoryContext
	for domain = 0; domain < 2; domain++ {
		for i := 0; i < 4; i++ {
			root = uint64(i + 1)
			stamps[domain][i], contexts[domain][i] = m.NativeCodeContext()
			contexts[domain][i].PublishLink(0x1000, 16, 1, [17]uint8{})
		}
	}
	domain, root = 0, 5
	replacement, context := m.NativeCodeContext()
	if context != contexts[0][0] || replacement.Identity != stamps[0][0].Identity || replacement.Epoch <= stamps[0][0].Epoch || context.LookupLink(0x1000) != nil {
		t.Fatal("evicted root retained or resurrected its old publication")
	}
	for domain = 0; domain < 2; domain++ {
		for i := 0; i < 4; i++ {
			if domain == 0 && i == 0 {
				continue
			}
			root = uint64(i + 1)
			stamp, ctx := m.NativeCodeContext()
			if stamp != stamps[domain][i] || ctx != contexts[domain][i] || ctx.LookupLink(0x1000) == nil {
				t.Fatal("unrelated exact root lost code authority on slot replacement", domain, root)
			}
		}
	}
	// Global architectural invalidation must still revoke every saved stamp.
	m.InvalidateMappings()
	for domain = 0; domain < 2; domain++ {
		for i := 0; i < 4; i++ {
			root = uint64(i + 1)
			if m.CodeContext() == stamps[domain][i] {
				t.Fatal("global invalidation preserved old root stamp", domain, root)
			}
		}
	}
}

func TestRootSlotEpochExhaustionDisablesSnapshots(t *testing.T) {
	root := uint64(1)
	m := New(make([]byte, 4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpace = func() uint64 { return root }
	for root = 1; root <= 4; root++ {
		m.NativeCodeContext()
	}
	m.clock = ^uint64(0)
	root = 5
	stamp, ctx := m.NativeCodeContext()
	if stamp.Identity != nil || ctx != nil || m.NativeCodeVersions() || m.CodeContext().Identity != nil {
		t.Fatal("exhausted epoch counter admitted native authority")
	}
}

func TestTableWriteRevokesAllAndOnlyRecordedRoots(t *testing.T) {
	root, domain := uint64(1), 0
	m := New(make([]byte, 4*4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpacePair = func() (uint64, uint64) { return 7, root }
	m.Domain = func() int { return domain }
	type view struct {
		root   uint64
		domain int
		table  uint64
		stamp  engine.CodeStamp
	}
	views := []view{{root: 1, domain: 0, table: 4096}, {root: 2, domain: 0, table: 4096}, {root: 1, domain: 1, table: 4096}, {root: 3, domain: 0, table: 12288}}
	for i := range views {
		root, domain = views[i].root, views[i].domain
		m.MarkPageTable(views[i].table)
		views[i].stamp = m.CodeContext()
		m.NativeRefill(0)
	}
	// Includes a watched interior page, not just the first/last physical page.
	m.BeforeWrite(0, 3*4096)
	for i := range views {
		root, domain = views[i].root, views[i].domain
		if (m.CodeContext() != views[i].stamp) != (i < 3) {
			t.Fatal("shared table owner revocation mismatch", i)
		}
		if m.NativeContext().Pages[0].Data != nil {
			t.Fatal("table mutation retained native data permission", i)
		}
	}
	// An executable alias makes the same physical write global, even if its
	// page-table owner set is smaller. No root may keep a stale code shortcut.
	root, domain = 3, 0
	if _, err := m.CodeVersion(4096); err != nil {
		t.Fatal(err)
	}
	for i := range views {
		root, domain = views[i].root, views[i].domain
		views[i].stamp = m.CodeContext()
	}
	m.BeforeWrite(4096, 1)
	for i := range views {
		root, domain = views[i].root, views[i].domain
		if m.CodeContext() == views[i].stamp {
			t.Fatal("executable alias preserved root proof", i)
		}
	}
}

func TestTableOwnersCoverCrossDomainTLBHits(t *testing.T) {
	root, domain := uint64(1), 1
	m := New(make([]byte, 8192), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpace = func() uint64 { return root }
	m.Domain = func() int { return domain }
	m.MarkPageTable(0) // Shared TLB populated by kernel, before any user view.
	kernel := m.CodeContext()
	domain = 0
	user := m.CodeContext() // User hits the architectural TLB, without a new walk.
	m.BeforeWrite(0, 8)
	if m.CodeContext() == user {
		t.Fatal("new user view did not inherit shared TLB dependencies")
	}
	domain = 1
	if m.CodeContext() == kernel {
		t.Fatal("original kernel table owner survived mutation")
	}
	// A later walk must propagate to another already-cached privilege view too.
	m.MarkPageTable(4096)
	kernel = m.CodeContext()
	domain = 0
	user = m.CodeContext()
	m.BeforeWrite(4096, 8)
	if m.CodeContext() == user {
		t.Fatal("existing user view missed later kernel TLB dependency")
	}
	domain = 1
	if m.CodeContext() == kernel {
		t.Fatal("later kernel table dependency survived mutation")
	}
}

func TestNativeContextRejectsEpochExhaustionDuringSlotSelection(t *testing.T) {
	root := uint64(1)
	m := New(make([]byte, 4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpace = func() uint64 { return root }
	for root = 1; root <= 4; root++ {
		if m.NativeContext() == nil {
			t.Fatal("missing initial context")
		}
	}
	m.clock = ^uint64(0)
	root = 5
	if m.NativeContext() != nil || m.CodeContext().Identity != nil {
		t.Fatal("slot selection exhaustion returned a live native context")
	}
}
