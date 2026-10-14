package systemmemory

import emu "renvo.dev/internal/rfe/runtime"

// PageCacheAddress is a reversible key permutation for the unchanged 64-slot
// native descriptor table. Plain low-page bits can make distant stack/data
// mappings collide on every alternating access. Fold the next six page bits
// into the slot bits while retaining the full tag and all byte-offset bits.
// This is only a key: MMU authority always checks the reconstructed guest VA.
func PageCacheAddress(b *emu.Builder, address emu.Value) emu.Value {
	mix := b.Binary(emu.And, b.Shift(emu.Shr, address, 6), b.Constant(63<<12))
	return b.Binary(emu.Xor, address, mix)
}

// UnfoldPageCacheAddress is its own inverse: the source bits (18..23) are not
// among the changed bits (12..17). No address bit or page offset is discarded.
func UnfoldPageCacheAddress(key uint64) uint64 { return key ^ (key >> 6 & (63 << 12)) }

func X86KernelPageAddress(b *emu.Builder, address emu.Value) emu.Value {
	return PageCacheAddress(b, X86KernelAddress(b, address))
}
