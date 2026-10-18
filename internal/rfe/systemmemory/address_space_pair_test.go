package systemmemory

import (
	"renvo.dev/internal/rfe/engine"
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

func TestRootPairUsesBothExactComponentsAndPreservesDomainLinks(t *testing.T) {
	roots := [][2]uint64{{1, 2}, {2, 1}, {1, 1<<48 | 2}, {1<<48 | 1, 2}}
	root, domain := roots[0], 0
	m := New(make([]byte, 4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpace = func() uint64 { return 99 }
	m.AddressSpacePair = func() (uint64, uint64) { return root[0], root[1] }
	m.Domain = func() int { return domain }
	var contexts [2][4]*emu.MemoryContext
	var stamps [2][4]engine.CodeStamp
	for domain = 0; domain < 2; domain++ {
		for i := range roots {
			root = roots[i]
			// Also exercise callback changes without SwitchAddressSpace: direct
			// architectural register setup must not reuse another root's grants.
			stamp, ctx := m.CodeContext(), m.NativeContext()
			for d := 0; d <= domain; d++ {
				for j := range contexts[d] {
					if contexts[d][j] == ctx || stamps[d][j].Identity == stamp.Identity {
						t.Fatal("distinct exact roots or domains shared an identity", domain, i, d, j)
					}
				}
			}
			contexts[domain][i], stamps[domain][i] = ctx, stamp
			ctx.PublishLink(0x1000+uint64(i)*4, (i+1)*16, 1, [17]uint8{})
			m.NativeRefill(0)
			if ctx.Pages[0].Data == nil {
				t.Fatal("data mapping did not warm")
			}
		}
	}
	for domain = 0; domain < 2; domain++ {
		for i := range roots {
			root = roots[i]
			ctx := m.NativeContext()
			if ctx != contexts[domain][i] || m.CodeContext() != stamps[domain][i] || ctx.LookupLink(0x1000+uint64(i)*4) == nil {
				t.Fatal("unchanged root/domain round trip lost executable links", domain, i)
			}
			for d := range contexts {
				for j := range contexts[d] {
					if contexts[d][j].Pages[0].Data != nil {
						t.Fatal("root callback change retained data permissions", d, j)
					}
				}
			}
		}
	}
	domain = 0
	m.MarkPageTable(0)
	m.BeforeWrite(0, 8)
	for domain = 0; domain < 2; domain++ {
		for i := range roots {
			root = roots[i]
			changed := m.CodeContext() != stamps[domain][i]
			if changed != (i == len(roots)-1) {
				t.Fatal("page-table write did not follow its exact root owner", domain, i, changed)
			}
		}
	}
}

func TestRootPairFifthSlotReleasesOnlyRetiredTableOwners(t *testing.T) {
	root, domain := uint64(1), 0
	m := New(make([]byte, 3*4096), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.AddressSpacePair = func() (uint64, uint64) { return 7, root }
	m.Domain = func() int { return domain }
	m.MarkPageTable(0)
	m.MarkPageTable(4096)
	m.MarkPageTable(8192)
	first := m.CodeContext()
	domain = 1
	m.MarkPageTable(8192)
	domain = 0
	for root = 2; root <= 5; root++ {
		m.CodeContext()
		if root == 2 {
			m.MarkPageTable(4096)
		}
	}
	root = 5
	before := m.CodeContext()
	m.BeforeWrite(0, 1)
	if m.CodeContext() != before {
		t.Fatal("fifth paired root retained the evicted root's private watch")
	}
	// A write in another live root must revoke THAT owner's proof, without
	// revoking root 5, which never walked the table.
	for _, offset := range []uint64{4096, 8192} {
		ownerRoot, ownerDomain := uint64(2), 0
		if offset == 8192 {
			ownerRoot, ownerDomain = 1, 1
		}
		root, domain = ownerRoot, ownerDomain
		ownerBefore := m.CodeContext()
		root, domain = 5, 0
		before = m.CodeContext()
		m.BeforeWrite(offset, 1)
		if m.CodeContext() != before {
			t.Fatal("unrelated paired root revoked", offset)
		}
		root, domain = ownerRoot, ownerDomain
		if m.CodeContext() == ownerBefore {
			t.Fatal("root replacement released another root/domain's table watch", offset)
		}
	}
	domain = 0
	root = 6
	before = m.CodeContext()
	m.BeforeWrite(4096, 1)
	if m.CodeContext() != before {
		t.Fatal("last paired-root owner kept its retired watch")
	}
	root = 1
	if m.CodeContext() == first {
		t.Fatal("reused paired-root slot resurrected an old proof")
	}
	if len(m.tablePages) > len(m.watched) {
		t.Fatal("paired root churn exceeded bounded table metadata")
	}
}

func TestNativeSnapshotObservesOneExactDomainRootAndFreshReplacementEpoch(t *testing.T) {
	root, domain := uint64(1), 0
	domains, roots, active := 0, 0, 0
	enabled := true
	m := New(make([]byte, 4096), func() bool { active++; return enabled }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.Domain = func() int { domains++; return domain }
	m.AddressSpacePair = func() (uint64, uint64) { roots++; return 1<<48 | 7, root }
	first, firstContext := m.NativeCodeContext()
	if domains != 1 || roots != 1 || active != 1 || first.Identity == nil || firstContext == nil || firstContext.Clock == nil {
		t.Fatal("snapshot repeated or omitted architectural observation", domains, roots, active)
	}
	for domain = 0; domain < 3; domain++ {
		for root = 1; root <= 6; root++ {
			domains, roots, active = 0, 0, 0
			stamp, context := m.NativeCodeContext()
			if domains != 1 || roots != 1 || active != 1 {
				t.Fatal("snapshot is not a single observation")
			}
			if stamp != m.CodeContext() || context != m.NativeContext() || context.Clock == nil {
				t.Fatal("snapshot and ordinary queries disagree", domain, root)
			}
			if domain == 0 && root == 5 && (stamp == first || stamp.Epoch <= first.Epoch) {
				t.Fatal("fifth root returned epoch from before slot replacement")
			}
		}
	}
	domain = 0
	enabled = false
	if stamp, context := m.NativeCodeContext(); stamp.Identity != nil || context != nil {
		t.Fatal("inactive native mapping produced a usable snapshot")
	}
	enabled = true
	m.InvalidateMappings()
	stamp, context := m.NativeCodeContext()
	if stamp != m.CodeContext() || context != m.NativeContext() {
		t.Fatal("mapping invalidation not observed")
	}
}
