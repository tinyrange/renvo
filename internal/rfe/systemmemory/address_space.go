package systemmemory

import "renvo.dev/internal/rfe/engine"

type spaceIdentity struct {
	tag          uint64
	tagHigh      uint64
	identity     engine.CodeIdentity
	valid        bool
	epoch        uint64 // never reused when this bounded root slot is reassigned
	tables       []int
	coarseTables bool
	codePages    [64]uint64  // conservative virtual execute-page filter; never authority
	tableEntries [256]uint64 // fixed 16384-bit conservative entry dependency filter
}

func (m *RAM) addressSpace() (uint64, uint64) {
	if m.AddressSpacePair != nil {
		return m.AddressSpacePair()
	}
	if m.AddressSpace != nil {
		return m.AddressSpace(), 0
	}
	return 0, 0
}

// SwitchAddressSpace is only for a checked MMU-root selection, not TLBI,
// permission-control changes or page-table writes. Generic data descriptors are
// revoked; explicitly rooted owners retain only their separate exact-root views.
// Executable links remain subject to the exact root identity guard.
func (m *RAM) SwitchAddressSpace() {
	m.profileMapping[2]++
	m.SelectPrivilege()
}

func (m *RAM) contextStamp(domain int) engine.CodeStamp {
	if m.AddressSpace == nil && m.AddressSpacePair == nil {
		return engine.CodeStamp{Identity: &m.identity[domain], Epoch: m.codeEpoch}
	}
	slot := m.contextSlotFor(domain, m.lastSpace, m.lastSpaceHigh)
	if m.disabled {
		return engine.CodeStamp{}
	}
	space := &m.spaces[domain][slot]
	return engine.CodeStamp{Identity: &space.identity, Epoch: space.epoch}
}

func (m *RAM) contextSlot() int {
	domain := m.domain()
	tag, tagHigh := m.addressSpace()
	return m.contextSlotFor(domain, tag, tagHigh)
}

// The exact ordered root pair has already been observed by the caller. Do not
// repeat architectural callbacks while selecting its bounded identity slot.
func (m *RAM) contextSlotFor(domain int, tag, tagHigh uint64) int {
	if m.AddressSpace == nil && m.AddressSpacePair == nil {
		return 0
	}
	for i := range m.spaces[domain] {
		space := &m.spaces[domain][i]
		if space.valid && space.tag == tag && space.tagHigh == tagHigh {
			return i
		}
	}
	slot := m.spaceNext[domain] & 3
	m.spaceNext[domain]++
	space := &m.spaces[domain][slot]
	// Reusing this identity address must never resurrect its former stamp.
	// Only the evicted slot loses authority: other exact root identities retain
	// their independent epochs. Global mapping mutations still revoke all slots.
	epoch := m.codeEpoch
	if space.valid {
		if m.clock == ^uint64(0) {
			m.disabled = true
		} else {
			m.clock++
			epoch = m.clock
		}
		m.native[domain][slot].ClearLinks()
		m.ForgetData()
		m.releaseTableSpace(domain, int(slot))
	}
	space.tag, space.tagHigh, space.valid, space.epoch = tag, tagHigh, true, epoch
	space.coarseTables = false
	clear(space.codePages[:])
	clear(space.tableEntries[:])
	// A newly selected privilege view can hit an architectural TLB populated by
	// another domain. Inherit its complete table dependency set before admitting
	// a context stamp; the data/execute walk need not run again on that hit.
	for d := range m.spaces {
		if d == domain {
			continue
		}
		for s := range m.spaces[d] {
			other := &m.spaces[d][s]
			if other.valid && other.tag == tag && other.tagHigh == tagHigh {
				space.coarseTables = space.coarseTables || other.coarseTables
				for i := range space.tableEntries {
					space.tableEntries[i] |= other.tableEntries[i]
				}
				for _, page := range other.tables {
					m.addTableOwner(domain, int(slot), page)
				}
			}
		}
	}
	return int(slot)
}

// An evicted root has no surviving native proof after its slot's fresh epoch
// and link revocation. Release its table watches only if no OTHER cached root owns
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

// Every execute walk records every physical table page under its exact cached
// root identity. A guest PTE write revokes ALL roots that used any touched table,
// including inactive roots and other privilege domains. Independent roots keep
// their proof; data descriptors still revoke globally before the store occurs.
func (m *RAM) invalidateTableOwners(owners uint16) {
	m.profileMapping[3]++
	if m.clock == ^uint64(0) {
		m.disabled = true
	} else {
		m.clock++
		for domain := range m.spaces {
			for slot := range m.spaces[domain] {
				if owners&(uint16(1)<<uint(domain*4+slot)) != 0 {
					m.spaces[domain][slot].epoch = m.clock
				}
			}
		}
	}
	m.ForgetData()
}

func (m *RAM) addTableOwner(domain, slot, page int) {
	mask := uint16(1) << uint(domain*4+slot)
	if m.tableOwners[page]&mask == 0 {
		m.tableOwners[page] |= mask
		m.spaces[domain][slot].tables = append(m.spaces[domain][slot].tables, page)
	}
}

func tableEntryBit(entry uint64) uint64 { return (entry ^ (entry >> 9) ^ (entry >> 18)) & 16383 }

func (m *RAM) tableEntryOwners(owners uint16, first, last uint64) uint16 {
	result := owners & (1 << 15) // legacy unscoped walkers always revoke globally
	for domain := range m.spaces {
		for slot := range m.spaces[domain] {
			mask := uint16(1) << uint(domain*4+slot)
			if owners&mask == 0 {
				continue
			}
			space := &m.spaces[domain][slot]
			if space.coarseTables {
				result |= mask
				continue
			}
			for entry := first; entry <= last; entry++ {
				bit := tableEntryBit(entry)
				if space.tableEntries[bit>>6]&(uint64(1)<<uint(bit&63)) != 0 {
					result |= mask
					break
				}
			}
		}
	}
	return result
}
