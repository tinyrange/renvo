package runtime

// The aligned guest base bounds a nonwrapping 4096-byte interval even at the
// top of uint64. Unsigned (address-base)<=4096-width simultaneously proves the
// complete virtual-page identity and complete access extent. A nonzero private
// host base is installed only after the ordinary full page/permission checks.
// Only ungrouped graph accesses use this; existing proved spans stay unchanged.
func (w *wasmCode) windowBegin(address, data, offset int, width uint64) {
	w.get(address)
	w.get(w.pageWindow[0])
	w.byte(0x7d)
	w.byte(0x22)
	w.u32(uint32(offset))
	w.i64(4096 - width)
	w.byte(0x58)
	w.get(w.pageWindow[1])
	w.byte(0x50)
	w.byte(0x45)
	w.byte(0x71)
	w.byte(0x04)
	w.byte(0x40)
	w.get(w.pageWindow[1])
	w.get(offset)
	w.byte(0x7c)
	w.set(data)
	w.byte(0x05)
}
func (w *wasmCode) windowEnd(address, descriptor int, layout *WasmMemoryLayout) {
	w.get(descriptor)
	w.byte(0xa7)
	w.mem(0x29, 0, layout.PagePermissions)
	w.i64(7)
	w.byte(0x83)
	w.i64(3)
	w.byte(0x51)
	w.byte(0x04)
	w.byte(0x40)
	w.get(address)
	w.i64(^uint64(4095))
	w.byte(0x83)
	w.set(w.pageWindow[0])
	w.ptr(descriptor, layout.PageData)
	w.byte(0xad)
	w.set(w.pageWindow[1])
	w.byte(0x0b)
	w.byte(0x0b)
}
