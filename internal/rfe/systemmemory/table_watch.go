package systemmemory

// IsPageTable identifies spans whose mutation must also revoke the owner's
// architectural software TLB. Hardware A/D metadata writes use a separate
// callback and must not take this path.
func (m *RAM) IsPageTable(offset uint64, n int) bool {
	if n < 1 || offset >= uint64(len(m.Data)) || uint64(n) > uint64(len(m.Data))-offset {
		return false
	}
	if m.ownedWatches != nil {
		return m.ownedWatches.Intersects(offset, n, 2)
	}
	for page := offset >> 12; page <= (offset+uint64(n)-1)>>12; page++ {
		if m.watched[page]&2 != 0 {
			return true
		}
	}
	return false
}
