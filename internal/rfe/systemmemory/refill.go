package systemmemory

import emu "renvo.dev/internal/rfe/runtime"

// NativeFastRefill is deliberately narrower than NativeRefill. It must never
// create a root slot, walk a table, add an architectural TLB entry or a watch.
// Only preowned descriptor metadata is changed, so the caller may reclaim all
// ABI temporaries without dangling persistent allocations.
func (m *RAM) NativeFastRefill(context *emu.MemoryContext) bool {
	if m.CachedMapping == nil || !m.usable() || context == nil || context.Status != 1 || context.Address&4095 > 4080 || context.Clock != &m.clock {
		return false
	}
	domain := m.domain()
	tag, high := m.addressSpace()
	if !m.spaceValid || domain != m.lastDomain || tag != m.lastSpace || high != m.lastSpaceHigh {
		return false
	}
	slot := -1
	if m.AddressSpace == nil && m.AddressSpacePair == nil {
		if context == &m.native[domain][0] {
			slot = 0
		}
	} else {
		for i := range m.spaces[domain] {
			s := &m.spaces[domain][i]
			if s.valid && s.tag == tag && s.tagHigh == high && context == &m.native[domain][i] {
				slot = i
				break
			}
		}
	}
	if slot < 0 {
		return false
	}
	address := context.Address
	if m.NativeAddress != nil {
		address = m.NativeAddress(address)
	}
	write := context.FastRefillWrite()
	offset, writable, ok := m.CachedMapping(address, write)
	if !ok || write && !writable || offset >= uint64(len(m.Data)) || offset&4095 != context.Address&4095 {
		return false
	}
	base := offset &^ uint64(4095)
	if uint64(len(m.Data))-base < 4096 || write && m.watched[base>>12] != 0 {
		return false
	}
	m.filled |= 1 << uint(domain*4+slot)
	permissions := uint64(1)
	if writable {
		permissions |= 2
	}
	if m.watched[base>>12] != 0 {
		permissions |= 4
	}
	context.Fill(context.Address>>12, (*[4096]byte)(m.Data[base:base+4096]), permissions, &m.clock)
	return true
}
