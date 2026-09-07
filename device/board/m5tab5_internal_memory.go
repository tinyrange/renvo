//go:build m5tab5

package board

// Internal memory above the target's BSS, ROM scratch, stack and display DMA
// descriptors. With the supported 128/256 KiB L2 cache configurations, SRAM
// extends at least to 0x4ff80000. Never claim the cache's backing SRAM.
// ESP-IDF v5.4.2: soc/esp32p4/include/soc/soc.h and heap/port/esp32p4/memory_layout.c.
const internalMemoryStart = uintptr(0x4ff41000)
const internalMemoryEnd = uintptr(0x4ff80000)

type internalMemoryError string

func (e internalMemoryError) Error() string { return string(e) }

var internalMemoryUsed uintptr

// ReserveInternalMemory reserves aligned, byte-addressable on-chip SRAM for
// the application's lifetime. It is single-owner/cooperative (not concurrent),
// has no free operation, and never falls back to PSRAM. The caller initializes
// the returned memory before use. This region must remain reserved if the
// target's BSS/stack or display-descriptor layout changes in future.
func ReserveInternalMemory(size, alignment uintptr) (uintptr, error) {
	cacheSize := load32(0x3ff10278)
	if cacheSize != 1<<9 && cacheSize != 1<<10 {
		return 0, internalMemoryError("unsupported L2 cache/SRAM partition")
	}
	if size == 0 || alignment == 0 || alignment > 4096 || alignment&(alignment-1) != 0 {
		return 0, internalMemoryError("invalid internal memory reservation")
	}
	start := (internalMemoryStart + internalMemoryUsed + alignment - 1) &^ (alignment - 1)
	if start > internalMemoryEnd || size > internalMemoryEnd-start {
		return 0, internalMemoryError("internal SRAM exhausted")
	}
	internalMemoryUsed = start + size - internalMemoryStart
	return start, nil
}
