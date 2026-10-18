//go:build !browser

package runtime

const nativeLinkSets = 256

func nativeLinkBlocks(m *MemoryContext, create bool) []NativeLink { return m.Blocks[:] }
func clearNativeLinks(m *MemoryContext)                           { clear(m.Blocks[:]) }

func noteNativeLinkSlot(m *MemoryContext, slot int) {}
