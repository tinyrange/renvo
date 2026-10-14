package systemmemory

// ReclaimCode runs after its caller's translated graph/links/arena are closed.
// Another privilege domain may retain a separate arena; never forget that
// owner's execution proofs. Page-table barriers always remain in place.
func (m *RAM) ReclaimCode() {
	m.InvalidateMappings()
	mask := byte(7)
	if m.MultipleArenas {
		mask = 1
		if m.domain() != 0 {
			mask = 6
		} // mapped and physical kernel share one arena
	}
	for page := range m.watched {
		m.codeOwners[page] &^= mask
		if m.codeOwners[page] == 0 {
			m.watched[page] &^= 1
		}
	}
}
