package runtime

// The complete cold graph refill service is private to its original module.
// All page guards and guest effects remain at the caller; success returns to
// the same full lookup. Prefix locals are published/reloaded at the call site.
func wasmGraphRefillFunction(l WasmMemoryLayout) []byte {
	var w wasmCode
	w.u32(0)
	w.get(1)
	w.get(0)
	w.mem(0x36, 2, l.ServiceWrite)
	w.contextValue(l.Status, 1)
	w.get(1)
	w.get(2)
	w.mem(0x37, 3, l.Address)
	w.get(1)
	w.mem(0x28, 2, l.ServiceSP)
	w.byte(0x04)
	w.byte(0x7f)
	w.get(1)
	w.mem(0x28, 2, l.ServiceSP)
	w.get(1)
	w.mem(0x28, 2, l.ServiceFP)
	for i := 0; i < 7; i++ {
		w.i32(0)
	}
	w.byte(0x10)
	w.u32(0)
	w.byte(0x1a)
	w.byte(0x1a)
	w.byte(0x05)
	w.i32(0)
	w.byte(0x0b)
	w.byte(0x0b)
	return w.code
}
