package systemmemory

import (
	"renvo.dev/internal/rfe/engine"
	emu "renvo.dev/internal/rfe/runtime"
)

func (m *RAM) InvalidatePage(address uint64) { m.InvalidateRange(address&^4095, 4096) }
func (m *RAM) InvalidateRange(address, size uint64) {
	if size == 0 {
		return
	}
	if !m.beginMappingInvalidation() {
		return
	}
	for _, c := range m.allContexts() {
		m.invalidateView(c, address, size)
	}
}

// InvalidateTaggedRange revokes only this owner-issued root/privilege view.
func (m *RAM) InvalidateTaggedRange(key *engine.CodeIdentity, address, size uint64) {
	c := m.tagged[key]
	if c == nil || size == 0 {
		return
	}
	if !m.beginMappingInvalidation() {
		return
	}
	m.invalidateView(c, address, size)
}
func (m *RAM) beginMappingInvalidation() bool {
	if m.clock == ^uint64(0) {
		m.disabled = true
		if m.proofs != nil {
			m.proofs.clear()
		}
		m.ForgetData()
		return false
	}
	m.clock++
	if m.proofs == nil {
		m.codeEpoch = m.clock
	}
	return true
}
func (m *RAM) invalidateView(c *emu.MemoryContext, address, size uint64) {
	if m.proofs != nil {
		m.proofs.revoke(c, address, size)
	} else {
		c.ClearLinks()
	}
	if size == 4096 {
		c.Forget(address >> 12)
		if m.direct != nil {
			m.direct.forgetPage(address)
		}
		return
	}
	var pages []uint64
	if c.Memory != nil {
		for _, p := range c.Memory.Pages() {
			if p.Data != nil && p.Number >= address>>12 && p.Number-(address>>12) < size>>12 {
				pages = append(pages, p.Number)
			}
		}
	} else {
		for _, p := range c.Pages {
			if p.Data != nil && p.Number >= address>>12 && p.Number-(address>>12) < size>>12 {
				pages = append(pages, p.Number)
			}
		}
	}
	for _, page := range pages {
		c.Forget(page)
		if m.direct != nil {
			m.direct.forgetPage(page << 12)
		}
	}
}
