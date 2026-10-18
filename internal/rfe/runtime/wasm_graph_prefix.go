package runtime

// Carry only precise prefix words between observation boundaries. Cumulative
// session totals and budgets remain in their original memory ABI.
func wasmGraphSessionSafe(l WasmMemoryLayout) bool {
	if !wasmDisjointProgress(l) {
		return false
	}
	for _, field := range []uint32{l.Retired, l.MemoryRetired} {
		for _, other := range []uint32{l.Address, l.Clock, l.Linked, l.LinkSource, l.LoopIterations, l.ServiceSP, l.ServiceFP, l.ServiceWrite} {
			if uint64(field) < uint64(other)+8 && uint64(other) < uint64(field)+8 {
				return false
			}
		}
		slots := l.PageSlots
		if slots == 0 {
			slots = 64
		}
		if uint64(field) < uint64(l.Pages)+uint64(slots)*uint64(l.PageSize) && uint64(l.Pages) < uint64(field)+8 {
			return false
		}
	}
	return true
}

// Status can share the header index only when no ABI observation can alias it.
// Architectural publication still follows all session writes, as before.
func wasmGraphStatusSafe(l WasmMemoryLayout) bool {
	if !wasmGraphSessionSafe(l) {
		return false
	}
	for _, other := range []uint32{l.Address, l.Clock, l.Linked, l.LinkSource, l.LoopIterations, l.ServiceSP, l.ServiceFP, l.ServiceWrite} {
		if uint64(l.Status) < uint64(other)+8 && uint64(other) < uint64(l.Status)+8 {
			return false
		}
	}
	slots := l.PageSlots
	if slots == 0 {
		slots = 64
	}
	return !(uint64(l.Status) < uint64(l.Pages)+uint64(slots)*uint64(l.PageSize) && uint64(l.Pages) < uint64(l.Status)+8)
}
func (w *wasmCode) flushSession() {
	for i, field := range w.sessionOffsets {
		w.get(1)
		w.get(w.sessionLocals[i])
		w.mem(0x37, 3, field)
	}
}
func (w *wasmCode) reloadSession() {
	for i, field := range w.sessionOffsets {
		w.get(1)
		w.mem(0x29, 3, field)
		w.set(w.sessionLocals[i])
	}
}
