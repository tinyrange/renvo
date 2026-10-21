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
type NativeLinkTable = rfeabi.LinkTable

func (m *MemoryContext) SelectLinkTable(table *NativeLinkTable) {
	m.Links = table
	m.DescriptorMask = 0
	if table.Dynamic() {
		m.DescriptorMask = table.Mask
	}
	if table != nil {
		m.LinkEpoch = table.Epoch
		m.linkVictim = table.Victim
	}
}

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
	if m.Links.Dynamic() {
		return m.Links.Lookup(pc, m.LinkEpoch)
	}
	blocks := m.LinkDescriptors()
	base := NativeLinkSlot(pc)
	for way := uint64(0); way < 4; way++ {
		link := &blocks[base+way*256]
		if link.PC == pc && link.Instructions != 0 && link.Generation == m.LinkEpoch {
			return link
		}
	}
	return nil
}

// ClearLinks invalidates every descriptor without writing 64 KiB per code
// mutation. Epoch wrap clears storage before reusing an epoch, never old grants.
func (m *MemoryContext) ClearLinks() {
	m.LinkEpoch++
	if m.LinkEpoch == 0 {
		clear(m.LinkDescriptors()[:])
		if m.Links.Dynamic() {
			clear(m.Links.Entries())
		}
		m.LinkEpoch = 1
	}
	m.linkVictim = 0
	if m.Links != nil {
		m.Links.Epoch = m.LinkEpoch
		m.Links.ClearDynamic()
		m.Links.Epoch = m.LinkEpoch
		m.Links.Victim = 0
	}
}
func (m *MemoryContext) PublishLink(pc uint64, entry int, instructions int, prefix [17]uint8) {
	if entry < 0 || entry&15 != 0 || instructions < 1 || instructions > 2147483647 {
		return
	}
	for i := 0; i <= instructions && i < len(prefix); i++ {
		if int(prefix[i]) > i || i > 0 && prefix[i] < prefix[i-1] {
			return
		}
	}
	if m.Links.Dynamic() {
		m.Links.Publish(NativeLink{PC: pc, Entry: uint64(entry), Instructions: uint64(instructions), Prefix: prefix, Generation: m.LinkEpoch})
		m.DescriptorMask = m.Links.Mask
		return
	}
	blocks := m.LinkDescriptors()
	base, slot := NativeLinkSlot(pc), uint64(0)
	found := false
	for way := uint64(0); way < 4; way++ {
		i := base + way*256
		if blocks[i].PC == pc && blocks[i].Instructions != 0 {
			slot, found = i, true
			break
		}
	}
	if !found {
		for way := uint64(0); way < 4; way++ {
			i := base + way*256
			if blocks[i].Instructions == 0 || blocks[i].Generation != m.LinkEpoch {
				slot, found = i, true
				break
			}
		}
	}
	if !found {
		slot = base + (m.linkVictim&3)*256
		m.linkVictim++
		if m.Links != nil {
			m.Links.Victim = m.linkVictim
		}
	}
	blocks[slot] = NativeLink{PC: pc, Entry: uint64(entry), Instructions: uint64(instructions), Prefix: prefix, Generation: m.LinkEpoch}
}

func NewNativeLinkTable() *NativeLinkTable { return rfeabi.NewLinkTable() }
