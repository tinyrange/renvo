package systemmemory

import "renvo.dev/internal/rfe/engine"

type spaceIdentity struct {
	tag      uint64
	identity engine.CodeIdentity
	valid    bool
	tables   []int
}

// SwitchAddressSpace is only for a checked MMU-root selection, not TLBI,
// permission-control changes or page-table writes. All data descriptors are
// revoked; executable links remain subject to the exact root identity guard.
func (m *RAM) SwitchAddressSpace() {
	m.profileMapping[2]++
	m.ForgetData()
}

func (m *RAM) contextIdentity() *engine.CodeIdentity {
	domain := m.domain()
	if m.AddressSpace == nil {
		return &m.identity[domain]
	}
	return &m.spaces[domain][m.contextSlot()].identity
}

func (m *RAM) contextSlot() int {
	domain := m.domain()
	if m.AddressSpace == nil {
		return 0
	}
	tag := m.AddressSpace()
	for i := range m.spaces[domain] {
		space := &m.spaces[domain][i]
		if space.valid && space.tag == tag {
			return i
		}
	}
	slot := m.spaceNext[domain] & 3
	m.spaceNext[domain]++
	space := &m.spaces[domain][slot]
	// Reusing an identity address must never resurrect a former stamp. A global
	// fresh epoch invalidates ALL old root proofs before bounded-slot replacement.
	if space.valid {
		m.InvalidateMappings()
		m.releaseTableSpace(domain, int(slot))
	}
	space.tag, space.valid = tag, true
	return int(slot)
}

// An evicted root has no surviving native proof after contextSlot's fresh
// global epoch. Release its table watches only if no OTHER cached root owns
// the same frame. Code watches are independent and never released here.
func (m *RAM) releaseTableSpace(domain, slot int) {
	mask := uint16(1) << uint(domain*4+slot)
	space := &m.spaces[domain][slot]
	for _, page := range space.tables {
		m.tableOwners[page] &^= mask
		if m.tableOwners[page] == 0 {
			m.watched[page] &^= 2
		}
	}
	space.tables = space.tables[:0]
}
