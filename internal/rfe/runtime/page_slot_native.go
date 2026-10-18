//go:build !browser

package runtime

const NativePageSlots = 64

type browserPageTable [0]NativePage

func NativePageSlot(number uint64) uint64 { return number & (NativePageSlots - 1) }

// PageSlot returns metadata only; full tags and permissions still authorize use.
func (m *MemoryContext) PageSlot(number uint64) *NativePage {
	return &m.Pages[int(NativePageSlot(number))]
}
func (m *MemoryContext) ClearPages() { clear(m.Pages[:]) }
