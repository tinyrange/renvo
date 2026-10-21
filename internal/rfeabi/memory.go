package rfeabi

// WriteWatches records exact physical bytes whose stores require architectural
// coherence work. All aliases share the same masks; publication occurs only at
// quiescent host boundaries. Native code reads masks but never modifies them.
type WriteWatches struct {
	Base      **[4096]byte
	Count     uint64
	Revision  uint64
	pages     []*[4096]byte
	allocated []*[4096]byte
}

func NewWriteWatches(bytes int) *WriteWatches {
	w := &WriteWatches{pages: make([]*[4096]byte, (bytes+4095)/4096), Revision: 1}
	w.Count = uint64(len(w.pages))
	if len(w.pages) != 0 {
		w.Base = &w.pages[0]
	}
	return w
}
func (w *WriteWatches) Mark(offset uint64, n int, flags byte) {
	if n < 1 || offset >= w.Count*4096 || uint64(n) > w.Count*4096-offset {
		return
	}
	for at := offset; at < offset+uint64(n); at++ {
		page := at >> 12
		if w.pages[page] == nil {
			w.pages[page] = new([4096]byte)
			w.allocated = append(w.allocated, w.pages[page])
			w.Revision++
		}
		w.pages[page][at&4095] |= flags
	}
}
func (w *WriteWatches) Intersects(offset uint64, n int, flags byte) bool {
	if n < 1 || offset >= w.Count*4096 || uint64(n) > w.Count*4096-offset {
		return false
	}
	for at := offset; at < offset+uint64(n); at++ {
		if p := w.pages[at>>12]; p != nil && p[at&4095]&flags != 0 {
			return true
		}
	}
	return false
}
func (w *WriteWatches) Clear(flags byte) {
	for _, p := range w.pages {
		if p != nil {
			for i := range p {
				p[i] &^= flags
			}
		}
	}
}
func (w *WriteWatches) Masks() []*[4096]byte          { return w.pages }
func (w *WriteWatches) AllocatedMasks() []*[4096]byte { return w.allocated }

// OwnedMemory is an address-space-sized open-addressed table, not a finite
// translation cache. Its load factor is kept below one half, so collisions do
// not evict other live mappings. Data pointers can only come from backing.
// The fixed header is read by native code; private slices root all allocations.
type OwnedMemory struct {
	Base         *Page
	Mask         uint64
	Backing      *byte
	Bytes        uint64
	Watches      *WriteWatches
	Clock        *uint64
	Revision     uint64
	DecodeEpochs *uint64
	decodeEpochs []uint64
	entries      []Page
	backing      []byte
	used         int
}

func NewOwnedMemory(backing []byte, clock *uint64, watches *WriteWatches) *OwnedMemory {
	m := &OwnedMemory{backing: backing, Clock: clock, Watches: watches, Revision: 1}
	m.Bytes = uint64(len(backing))
	if len(backing) != 0 {
		m.Backing = &backing[0]
	}
	m.grow(16)
	return m
}
func memoryHash(n uint64) uint64 { return n ^ (n >> 12) }
func (m *OwnedMemory) slot(number uint64) int {
	slot := memoryHash(number) & m.Mask
	for m.entries[slot].Data != nil && m.entries[slot].Number != number {
		slot = (slot + 1) & m.Mask
	}
	return int(slot)
}
func (m *OwnedMemory) grow(size int) {
	old := m.entries
	m.entries = make([]Page, size)
	m.Base = &m.entries[0]
	m.Mask = uint64(size - 1)
	m.Revision++
	for _, p := range old {
		if p.Data != nil {
			m.entries[m.slot(p.Number)] = p
		}
	}
}
func (m *OwnedMemory) Fill(number, offset, permissions uint64) {
	if number >= 1<<36 || offset&4095 != 0 || offset > m.Bytes || m.Bytes-offset < 4096 || m.Clock == nil || permissions&^uint64(7) != 0 {
		m.Forget(number)
		return
	}
	if (m.used+1)*2 > len(m.entries) {
		m.grow(len(m.entries) * 2)
	}
	slot := m.slot(number)
	if m.entries[slot].Data == nil {
		m.used++
	}
	m.entries[slot] = Page{number, (*[4096]byte)(m.backing[offset : offset+4096]), permissions, m.Clock}
}
func (m *OwnedMemory) Lookup(number uint64) (Page, bool) {
	p := m.entries[m.slot(number)]
	return p, p.Data != nil
}
func (m *OwnedMemory) Forget(number uint64) {
	slot := m.slot(number)
	if m.entries[slot].Data == nil {
		return
	}
	m.entries[slot] = Page{}
	m.used--
	// Backshift the following cluster; a deleted slot cannot terminate a
	// probe before another colliding mapping that is still live.
	at := (slot + 1) & int(m.Mask)
	for m.entries[at].Data != nil {
		p := m.entries[at]
		m.entries[at] = Page{}
		m.entries[m.slot(p.Number)] = p
		at = (at + 1) & int(m.Mask)
	}
}
func (m *OwnedMemory) Clear()               { clear(m.entries); m.used = 0 }
func (m *OwnedMemory) Pages() []Page        { return m.entries }
func (m *OwnedMemory) BackingBytes() []byte { return m.backing }
func (m *OwnedMemory) Valid() bool {
	return (m.DecodeEpochs == nil && len(m.decodeEpochs) == 0 || len(m.decodeEpochs) == int((m.Bytes+4095)/4096) && len(m.decodeEpochs) > 0 && m.DecodeEpochs == &m.decodeEpochs[0]) && len(m.entries) > 0 && m.Base == &m.entries[0] && m.Mask == uint64(len(m.entries)-1) && len(m.backing) > 0 && m.Backing == &m.backing[0] && m.Bytes == uint64(len(m.backing)) && m.Clock != nil && m.Watches != nil && m.Watches.Count == uint64(len(m.Watches.pages)) && m.Watches.Count == (m.Bytes+4095)/4096 && m.Watches.Count > 0 && m.Watches.Base == &m.Watches.pages[0]
}

// SetDecodeEpochs opts the owner into coherent native writes to decoded-only
// bytes. Versions share Clock and cover every physical page, including aliases.
// The trusted owner calls this only at a quiescent boundary.
func (m *OwnedMemory) SetDecodeEpochs(epochs []uint64) {
	if len(epochs) != int((m.Bytes+4095)/4096) {
		panic("invalid physical decode versions")
	}
	if len(m.decodeEpochs) != 0 && &m.decodeEpochs[0] == &epochs[0] {
		return
	}
	m.decodeEpochs = epochs
	m.DecodeEpochs = &epochs[0]
	m.Revision++
}
func (m *OwnedMemory) DecodeVersions() []uint64 { return m.decodeEpochs }
func (w *WriteWatches) Unmark(offset uint64, n int, flags byte) {
	if n < 1 || offset >= w.Count*4096 || uint64(n) > w.Count*4096-offset {
		return
	}
	for at := offset; at < offset+uint64(n); at++ {
		if p := w.pages[at>>12]; p != nil {
			p[at&4095] &^= flags
		}
	}
}
