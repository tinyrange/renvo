//go:build browser

package runtime

// Browser page translations are bounded host metadata, not extra guest RAM.
// Keep pages first so the emitted layout addresses exactly this descriptor array.
const NativePageSlots = 4096

type browserPageTable struct {
	pages   [NativePageSlots]NativePage
	slots   [NativePageSlots]uint16
	present [NativePageSlots]bool
	count   int
}

func NativePageSlot(number uint64) uint64 { return number & (NativePageSlots - 1) }

// Mark every exposed mutable slot, independently of the caller-visible tag or
// data pointer. Copies own their descriptor and revocation arrays by value.
func (m *MemoryContext) PageSlot(number uint64) *NativePage {
	slot := int(NativePageSlot(number))
	pages := &m.browserPages
	if !pages.present[slot] {
		pages.present[slot] = true
		pages.slots[pages.count] = uint16(slot)
		pages.count++
	}
	return &pages.pages[slot]
}
func (m *MemoryContext) ClearPages() {
	pages := &m.browserPages
	for i := 0; i < pages.count; i++ {
		slot := pages.slots[i]
		pages.pages[slot] = NativePage{}
		pages.present[slot] = false
	}
	pages.count = 0
}
