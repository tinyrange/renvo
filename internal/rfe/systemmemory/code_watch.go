package systemmemory

// MarkCode installs the physical write barrier after the trusted owner has
// execute-checked and resolved this RAM offset. It grants no access or mapping;
// callers still check every fetched byte, including noncontiguous page tails.
func (m *RAM) MarkCode(offset uint64) {
	if offset >= uint64(len(m.Data)) {
		return
	}
	m.codeOwners[offset>>12] |= 1 << uint(m.domain())
	m.watch(offset, 1)
}
