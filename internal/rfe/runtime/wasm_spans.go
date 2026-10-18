package runtime

// A span is an IR-derived group of accesses to one LoadState value plus small
// constant offsets. The proof is refreshed at the group's first actual access
// each iteration; failure merely selects the original checks at every access.
type wasmMemorySpan struct {
	base, first, count int
	low, high          int64
	read, write        bool
	page, data, fast   int
}

func wasmMemorySpans(ops []Op) ([]wasmMemorySpan, []int) {
	bases, offsets, groups, accesses := make([]int, len(ops)), make([]int64, len(ops)), make([]int, len(ops)), make([]int, len(ops))
	for i := range ops {
		bases[i], groups[i], accesses[i] = -1, -1, -1
	}
	var spans []wasmMemorySpan
	for i, op := range ops {
		switch op.Kind {
		case LoadState:
			bases[i] = i
		case Add, Sub:
			a, constant := int(op.A), int(op.B)
			if op.Kind == Add && ops[a].Kind == Const {
				a, constant = constant, a
			}
			if bases[a] >= 0 && ops[constant].Kind == Const {
				delta := int64(ops[constant].Imm)
				if delta >= -4096 && delta <= 4096 {
					if op.Kind == Sub {
						delta = -delta
					}
					delta += offsets[a]
					if delta >= -4096 && delta <= 4096 {
						bases[i], offsets[i] = bases[a], delta
					}
				}
			}
		case MemoryLoad, MemoryStore, MemoryPairStore:
			base := bases[op.A]
			if base < 0 {
				continue
			}
			width := int64(op.Imm)
			if op.Kind == MemoryPairStore {
				width = 16
			}
			low, high := offsets[op.A], offsets[op.A]+width
			group := groups[base]
			if group < 0 {
				group = len(spans)
				groups[base] = group
				spans = append(spans, wasmMemorySpan{base: base, first: i, low: low, high: high})
			}
			span := &spans[group]
			span.count++
			if low < span.low {
				span.low = low
			}
			if high > span.high {
				span.high = high
			}
			span.read = span.read || op.Kind == MemoryLoad
			span.write = span.write || op.Kind != MemoryLoad
			accesses[i] = group
		}
	}
	for i, group := range accesses {
		if group >= 0 && (spans[group].count < 2 || spans[group].high-spans[group].low > 4096) {
			accesses[i] = -1
		}
	}
	return spans, accesses
}

func (w *wasmCode) spanGuard(span wasmMemorySpan, address, descriptor int, layout *WasmMemoryLayout) {
	w.i64(0)
	w.set(span.fast)
	w.get(span.base + 2)
	w.i64(uint64(span.low))
	w.byte(0x7c)
	w.set(address)
	// A bounded offset inside one page proves every address in the span,
	// including at the top of the uint64 address space, without wrapping.
	w.get(address)
	w.i64(4095)
	w.byte(0x83)
	w.i64(uint64(4096 - (span.high - span.low)))
	w.byte(0x58)
	w.byte(0x04)
	w.byte(0x40)
	w.get(span.data)
	w.byte(0x50)
	w.byte(0x45)
	w.get(span.page)
	w.get(address)
	w.i64(12)
	w.byte(0x88)
	w.byte(0x51)
	w.byte(0x71)
	w.byte(0x04)
	w.byte(0x40)
	w.i64(1)
	w.set(span.fast)
	w.byte(0x05)
	w.get(1)
	w.i32(layout.Pages)
	w.byte(0x6a)
	w.pageSlot(address, layout)
	w.byte(0xa7)
	w.i32(layout.PageSize)
	w.byte(0x6c)
	w.byte(0x6a)
	w.byte(0xad)
	w.set(descriptor)
	w.get(descriptor)
	w.byte(0xa7)
	w.mem(0x29, 0, layout.PageNumber)
	w.get(address)
	w.i64(12)
	w.byte(0x88)
	w.byte(0x51)
	w.ptr(descriptor, layout.PageData)
	w.byte(0x45)
	w.byte(0x45)
	w.byte(0x71)
	w.get(descriptor)
	w.byte(0xa7)
	w.mem(0x29, 0, layout.PagePermissions)
	mask, want := uint64(1), uint64(1)
	if span.write {
		mask, want = 6, 2
		if span.read {
			mask, want = 7, 3
		}
	}
	w.i64(mask)
	w.byte(0x83)
	w.i64(want)
	w.byte(0x51)
	w.byte(0x71)
	w.byte(0x04)
	w.byte(0x40)
	w.get(address)
	w.i64(12)
	w.byte(0x88)
	w.set(span.page)
	w.ptr(descriptor, layout.PageData)
	w.byte(0xad)
	w.set(span.data)
	w.i64(1)
	w.set(span.fast)
	w.byte(0x0b)
	w.byte(0x0b)
	w.byte(0x0b)
}
