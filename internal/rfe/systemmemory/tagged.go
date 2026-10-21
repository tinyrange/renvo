package systemmemory

import (
	"renvo.dev/internal/rfe/engine"
	emu "renvo.dev/internal/rfe/runtime"
)

// SelectNativeContext selects an owner-issued stable translation domain. Unlike
// the legacy bounded root cache, these contexts survive switches and share exact
// physical write watches. Callers must revoke affected views before table edits.
func (m *RAM) SelectNativeContext(key *engine.CodeIdentity) {
	if m.tagged == nil {
		m.tagged = make(map[*engine.CodeIdentity]*emu.MemoryContext)
	}
	c := m.tagged[key]
	if c == nil {
		c = new(emu.MemoryContext)
		m.tagged[key] = c
	}
	m.selected = c
}
func (m *RAM) nativeContext() *emu.MemoryContext {
	if m.selected != nil {
		return m.selected
	}
	return &m.native[m.domain()][m.contextSlot()]
}
func (m *RAM) allContexts() []*emu.MemoryContext {
	result := make([]*emu.MemoryContext, 0, 12+len(m.tagged))
	for i := range m.native {
		for j := range m.native[i] {
			result = append(result, &m.native[i][j])
		}
	}
	for _, c := range m.tagged {
		result = append(result, c)
	}
	return result
}
