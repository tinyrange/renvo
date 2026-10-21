package systemmemory

// BeforeNonCodeWrite requires the owner to prove that this physical span does
// not intersect ANY translated instruction byte in its live arena. It does not
// weaken native page write barriers: stores to mixed code/data pages still
// leave native execution before reaching this callback. Page-table writes must
// revoke native mappings even when their bytes are not executable code.
func (m *RAM) BeforeNonCodeWrite(offset uint64, n int) {
	if n < 1 || offset >= uint64(len(m.Data)) || uint64(n) > uint64(len(m.Data))-offset {
		return
	}
	if m.ownedWatches != nil && !m.ownedWatches.Intersects(offset, n, 2) {
		return
	}
	for page := offset >> 12; page <= (offset+uint64(n)-1)>>12; page++ {
		if m.watched[page]&2 != 0 {
			m.InvalidateMappings()
			return
		}
	}
}
