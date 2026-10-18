//go:build browser

package runtime

// Match the bounded compiled-entry working set. Sparse revocation below avoids
// scanning every unused descriptor on each code-context change.
const nativeLinkSets = 4096

func nativeLinkBlocks(m *MemoryContext, create bool) []NativeLink {
	if m.browserLinksOwner == m {
		return m.browserLinks
	}
	if !create {
		return nil
	}
	m.browserLinks = make([]NativeLink, nativeLinkSets*4)
	m.browserLinkPresent = make([]bool, nativeLinkSets*4)
	m.browserLinkSlots = make([]uint32, 0, 1024)
	m.browserLinksOwner = m
	return m.browserLinks
}
func noteNativeLinkSlot(m *MemoryContext, slot int) {
	if !m.browserLinkPresent[slot] {
		m.browserLinkPresent[slot] = true
		m.browserLinkSlots = append(m.browserLinkSlots, uint32(slot))
	}
}
func clearNativeLinks(m *MemoryContext) {
	if m.browserLinksOwner != m {
		return
	}
	// Revoke every touched slot, even if a caller changed its occupancy word.
	// Unused sets contain no authority and need no work on a context transition.
	for _, slot := range m.browserLinkSlots {
		m.browserLinks[slot].Instructions = 0
		m.browserLinkPresent[slot] = false
	}
	m.browserLinkSlots = m.browserLinkSlots[:0]
}
