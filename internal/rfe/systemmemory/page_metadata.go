package systemmemory

// BeforePageTableMetadata is only for a hardware walker setting accessed/dirty
// bits without changing address or permission fields. An executable alias still
// observes changed bytes and must invalidate. Otherwise revoke data descriptors
// (including stale dirty-bit write permissions), but keep executable proofs.
// Guest-authored PTE stores must continue to call BeforeWrite instead.
func (m *RAM) BeforePageTableMetadata(offset uint64, n int) {
	if n < 1 || offset >= uint64(len(m.Data)) || uint64(n) > uint64(len(m.Data))-offset {
		return
	}
	for page := offset >> 12; page <= (offset+uint64(n)-1)>>12; page++ {
		if m.watched[page]&1 != 0 {
			m.BeforeWrite(offset, n)
			return
		}
	}
	m.ForgetData()
}
