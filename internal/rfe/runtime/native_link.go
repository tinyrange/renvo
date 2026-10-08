package runtime

import "renvo.dev/internal/rfeabi"

// NativeLinkSlot selects one of 256 four-way sets by mixing code-page identity
// with the word offset. Full PC tags
// remain authoritative; a hash match never grants execution admission.
func NativeLinkSlot(pc uint64) uint64 { return ((pc >> 2) ^ (pc >> 12)) & 255 }

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
	base := NativeLinkSlot(pc)
	for way := uint64(0); way < 4; way++ {
		link := &m.Blocks[base+way*256]
		if link.PC == pc && link.Instructions != 0 {
			return link
		}
	}
	return nil
}
func (m *MemoryContext) ClearLinks() { m.Blocks = [1024]NativeLink{}; m.linkVictim = 0 }
func (m *MemoryContext) PublishLink(pc uint64, entry int, instructions int, prefix [17]uint8) {
	if entry < 0 || entry&15 != 0 || instructions < 1 || instructions > 256 {
		return
	}
	for i := 0; i <= instructions && i < len(prefix); i++ {
		if int(prefix[i]) > i || i > 0 && prefix[i] < prefix[i-1] {
			return
		}
	}
	base, slot := NativeLinkSlot(pc), uint64(0)
	found := false
	for way := uint64(0); way < 4; way++ {
		i := base + way*256
		if m.Blocks[i].PC == pc && m.Blocks[i].Instructions != 0 {
			slot, found = i, true
			break
		}
	}
	if !found {
		for way := uint64(0); way < 4; way++ {
			i := base + way*256
			if m.Blocks[i].Instructions == 0 {
				slot, found = i, true
				break
			}
		}
	}
	if !found {
		slot = base + (m.linkVictim&3)*256
		m.linkVictim++
	}
	m.Blocks[slot] = NativeLink{PC: pc, Entry: uint64(entry), Instructions: uint64(instructions), Prefix: prefix}
}
