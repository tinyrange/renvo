package systemmemory

import emu "renvo.dev/internal/rfe/runtime"

// KernelAddress admits only the full-width high kernel bank. Low user accesses
// by a kernel instruction deliberately return to the architectural slow path.
// The original registers and checkpoint PC remain full width for fault delivery.
func KernelAddress(b *emu.Builder, address emu.Value) emu.Value {
	b.Guard(b.Binary(emu.Equal, b.Shift(emu.Shr, address, 48), b.Constant(0xffff)))
	return b.Binary(emu.And, address, b.Constant(0x0000ffffffffffff))
}
