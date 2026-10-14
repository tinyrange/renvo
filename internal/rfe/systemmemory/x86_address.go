package systemmemory

import emu "renvo.dev/internal/rfe/runtime"

// X86KernelAddress preserves both canonical kernel banks under one checked
// low-48-bit key. The x86 MMU resolves the key after architectural sign extension;
// noncanonical original values must exit before any native host access.
func X86KernelAddress(b *emu.Builder, address emu.Value) emu.Value {
	upper := b.Shift(emu.Shr, address, 47)
	low := b.Binary(emu.Equal, upper, b.Constant(0))
	high := b.Binary(emu.Equal, upper, b.Constant(0x1ffff))
	b.Guard(b.Binary(emu.Or, low, high))
	return b.Binary(emu.And, address, b.Constant(0x0000ffffffffffff))
}
