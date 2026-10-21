package systemmemory

// Executable-byte changes revoke code proofs, not unchanged data translations.
// Owned native stores re-read physical byte watches even on cached page facts.
func (m *RAM) invalidateCode() {
	if m.clock == ^uint64(0) {
		m.disabled = true
		if m.proofs != nil {
			m.proofs.clear()
		}
		m.ForgetData()
		return
	}
	m.clock++
	if m.proofs != nil {
		return
	} // exact proofs/publications were revoked by BeforeWrite
	m.codeEpoch = m.clock
	for _, c := range m.allContexts() {
		c.ClearLinks()
	}
}
