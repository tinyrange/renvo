package linuxuser

import emu "renvo.dev/internal/rfe/runtime"

func (m *Memory) NativeContext() *emu.MemoryContext {
	context := &m.versions.native
	context.Clock = &m.versions.epoch
	return context
}
func (m *Memory) NativeRefill(address uint64) {
	m.selectDirect(address)
	if !validRange(address, 1) {
		return
	}
	m.lookupPage(address / PageSize)
}
func (m *Memory) fillNative(number uint64, p *page) {
	context := &m.versions.native
	if context.Clock != nil {
		context.Fill(number, p.data, uint64(p.permissions), &p.epoch)
	}
}

// NativeCodeVersions allows checked native regions to omit data-write epochs.
// Heap-backed pages already have the same ownership/invalidation guarantees as
// mapped pages; no direct-memory allocation or signal recovery is required.
func (m *Memory) NativeCodeVersions() bool { return !m.versions.closed }
