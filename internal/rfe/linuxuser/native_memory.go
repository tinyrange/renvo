package linuxuser

import emu "renvo.dev/internal/rfe/runtime"

func (m *Memory) NativeContext() *emu.MemoryContext {
	context := &m.versions.native
	context.Clock = &m.versions.epoch
	return context
}
func (m *Memory) NativeRefill(address uint64) {
	if !validRange(address, 1) {
		return
	}
	m.lookupPage(address / PageSize)
}
func (m *Memory) fillNative(number uint64, p *page) {
	context := &m.versions.native
	if context.Clock != nil {
		context.Fill(number, &p.data, uint64(p.permissions), &p.epoch)
	}
}
