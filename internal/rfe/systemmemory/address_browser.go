//go:build browser

package systemmemory

import emu "renvo.dev/internal/rfe/runtime"

// WASM compares full-width uint64 page tags and needs no native low-48-bit
// normalization. Both EL1 banks (including early identity-mapped RAM) retain
// distinct keys. Refill checks the original VA against the current MMU before
// admitting a descriptor; devices, invalid VAs and watched stores still exit.
func KernelAddress(b *emu.Builder, address emu.Value) emu.Value { return address }
func KernelNativeAddress(key uint64) uint64                     { return key }
