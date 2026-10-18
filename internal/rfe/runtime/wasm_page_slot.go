package runtime

func (w *wasmCode) pageSlot(address int, layout *WasmMemoryLayout) {
	w.get(address)
	w.i64(12)
	w.byte(0x88)
	if layout.HashPages {
		w.get(address)
		w.i64(18)
		w.byte(0x88)
		w.byte(0x85)
		w.get(address)
		w.i64(24)
		w.byte(0x88)
		w.byte(0x85)
	}
	slots := layout.PageSlots
	if slots == 0 {
		slots = 64
	}
	w.i64(uint64(slots - 1))
	w.byte(0x83)
}
