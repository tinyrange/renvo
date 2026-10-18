package runtime

import "renvo.dev/internal/rfeabi"

// NativeLinkSlot selects a bounded four-way set by mixing code-page identity
// with the word offset. Full PC tags
// remain authoritative; a hash match never grants execution admission.
func NativeLinkSlot(pc uint64) uint64 { return ((pc >> 2) ^ (pc >> 12)) & (nativeLinkSets - 1) }

// NativeLink is a full-PC-tagged entry in the bounded native dispatcher. Entry
// is an arena offset, not an executable address. Instructions==0 means absent.
// Prefix counts memory instructions in ordinary leaves (at most 16). Wider
// loop regions account exact dynamic memory progress inside their checked body.
type NativeLink = rfeabi.Descriptor

// ClaimLinks invalidates offsets when another native arena uses this context.
func (m *MemoryContext) ClaimLinks(n *Native) bool {
	if m.linkOwner == n {
		return false
	}
	m.linkOwner = n
	m.ClearLinks()
	return true
}

// LookupLink probes four ways without changing guest state or replacement order.
func (m *MemoryContext) LookupLink(pc uint64) *NativeLink {
	blocks := nativeLinkBlocks(m, false)
	if len(blocks) == 0 {
		return nil
	}
	base := int(NativeLinkSlot(pc))
	for way := 0; way < 4; way++ {
		link := &blocks[base+way*nativeLinkSets]
		if link.PC == pc && link.Instructions != 0 {
			return link
		}
	}
	return nil
}
func (m *MemoryContext) ClearLinks() { clearNativeLinks(m); m.linkVictim = 0 }

// PublishLink returns the exact installed descriptor so a caller need not
// immediately rescan its four-way set. The pointer is not execution authority
// and may be replaced by subsequent publications or revoked by ClearLinks.
func (m *MemoryContext) PublishLink(pc uint64, entry int, instructions int, prefix [17]uint8) *NativeLink {
	if entry < 0 || entry&15 != 0 || instructions < 1 || instructions > 256 {
		return nil
	}
	for i := 0; i <= instructions && i < len(prefix); i++ {
		if int(prefix[i]) > i || i > 0 && prefix[i] < prefix[i-1] {
			return nil
		}
	}
	blocks := nativeLinkBlocks(m, true)
	base, slot := int(NativeLinkSlot(pc)), 0
	found := false
	for way := 0; way < 4; way++ {
		i := base + way*nativeLinkSets
		if blocks[i].PC == pc && blocks[i].Instructions != 0 {
			slot, found = i, true
			break
		}
	}
	if !found {
		for way := 0; way < 4; way++ {
			i := base + way*nativeLinkSets
			if blocks[i].Instructions == 0 {
				slot, found = i, true
				break
			}
		}
	}
	if !found {
		slot = base + int(m.linkVictim&3)*nativeLinkSets
		m.linkVictim++
	}
	noteNativeLinkSlot(m, slot)
	blocks[slot] = NativeLink{PC: pc, Entry: uint64(entry), Instructions: uint64(instructions), Prefix: prefix}
	return &blocks[slot]
}
