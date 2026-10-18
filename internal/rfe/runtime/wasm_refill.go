package runtime

// The surrounding loop encloses only the checked descriptor lookup. A successful
// service repeats every guard at the SAME SSA address; it never replays a store,
// commits register locals, extends a budget or bypasses write barriers.
func (w *wasmCode) refillFailIf(l *WasmMemoryLayout, address int, write bool) {
	w.byte(0x04)
	w.byte(0x40)
	if w.refillFunction != 0 {
		// Prefix locals cannot alias Status/Address/ServiceWrite/SP/FP when
		// enabled. Publish before the helper and reload after its sole callback.
		w.flushSession()
		if write {
			w.i32(1)
		} else {
			w.i32(0)
		}
		w.get(1)
		w.get(address)
		w.byte(0x10)
		w.u32(uint32(w.refillFunction))
		w.reloadSession()
		if len(w.pageWindow) != 0 {
			w.i64(0)
			w.set(w.pageWindow[1])
		}
		w.byte(0x04)
		w.byte(0x40)
		w.contextValue(l.Status, 0)
		w.byte(0x0c)
		w.u32(2) // success if + failed-guard if surround the unchanged retry loop
		w.byte(0x0b)
		w.flushState()
		w.byte(0x0f)
		w.byte(0x0b)
		return
	}
	w.get(1)
	if write {
		w.i32(1)
	} else {
		w.i32(0)
	}
	w.mem(0x36, 2, l.ServiceWrite)
	w.contextValue(l.Status, 1)
	w.get(1)
	w.get(address)
	w.mem(0x37, 3, l.Address)
	w.get(1)
	w.mem(0x28, 2, l.ServiceSP)
	w.byte(0x04)
	w.byte(0x40)
	w.get(1)
	w.mem(0x28, 2, l.ServiceSP)
	w.get(1)
	w.mem(0x28, 2, l.ServiceFP)
	for i := 0; i < 7; i++ {
		w.i32(0)
	}
	w.flushSession()
	w.byte(0x10)
	w.u32(0) // direct imported raw AOT function, never JS or browser_step
	w.byte(0x1a)
	w.byte(0x1a) // secondary ABI results
	w.reloadSession()
	if len(w.pageWindow) != 0 {
		w.i64(0)
		w.set(w.pageWindow[1])
	}
	// A service may replace or revoke a page proved earlier in this loop.
	// Invalidate both cross-iteration data and the current-iteration fast bit,
	// even on failure; a callback's success never extends mapping authority.
	for _, span := range w.spanProofs {
		if span.count < 2 || span.high-span.low > 4096 {
			continue
		}
		w.i64(0)
		w.set(span.data)
		w.i64(0)
		w.set(span.fast)
	}
	w.byte(0x04)
	w.byte(0x40)
	w.contextValue(l.Status, 0)
	w.byte(0x0c)
	w.u32(3) // callback success -> guard loop outside these three ifs
	w.byte(0x0b)
	w.byte(0x0b)
	w.flushState()
	w.byte(0x0f)
	w.byte(0x0b)
}
