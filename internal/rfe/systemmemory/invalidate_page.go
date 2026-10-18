package systemmemory

// EnablePageInvalidation starts bounded execute-page dependency tracking.
// Revoke preexisting proofs first: code checked before this opt-in has no
// recorded virtual-page dependency. Repeated calls do not change authority.
// Other architectures incur no filter-update work in CodeVersion.
func (m *RAM) EnablePageInvalidation() {
	if !m.pageInvalidation {
		m.InvalidateMappings()
		m.pageInvalidation = true
	}
}

// executePageBit is a bounded conservative set: collisions can only revoke
// extra proofs. A miss is usable only within the exact root-slot lifetime.
func executePageBit(page uint64) uint64 { return (page ^ page>>12 ^ page>>24 ^ page>>36) & 4095 }

// InvalidatePage follows a single-page architectural TLB eviction, not a root
// switch or a full flush. Data aliases are all forgotten. Executable contexts
// are revoked for every cached privilege view of the current exact root that
// has ever execute-checked this virtual page. No physical mapping is inferred
// from a TLB that may already have discarded its stale entry.
//
// The filter is never cleared by a PTE write or epoch advance: a proof can be
// revalidated through an old architectural translation after that write. Only
// root-slot replacement clears it, together with fresh epoch and link revocation.
// Table watches remain intact for ALL surviving architectural translations.
func (m *RAM) InvalidatePage(address uint64) {
	if !m.pageInvalidation || m.AddressSpace == nil && m.AddressSpacePair == nil {
		m.InvalidateMappings()
		return
	}
	tag, high := m.addressSpace()
	bit := executePageBit(address >> 12)
	owners := uint16(0)
	for domain := range m.spaces {
		for slot := range m.spaces[domain] {
			space := &m.spaces[domain][slot]
			if space.valid && space.tag == tag && space.tagHigh == high && space.codePages[bit>>6]&(uint64(1)<<uint(bit&63)) != 0 {
				owners |= uint16(1) << uint(domain*4+slot)
			}
		}
	}
	if owners != 0 {
		m.invalidateTableOwners(owners)
	} else {
		m.ForgetData()
	}
}
