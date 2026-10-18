package runtime

func (w *wasmCode) packedFlags(op Op, a, b, x, y, r int) {
	width := op.Imm & 127
	w.get(a)
	w.mask(width)
	w.set(x)
	w.get(b)
	w.mask(width)
	w.set(y)
	w.get(x)
	if op.Kind == ArithmeticFlags {
		w.get(y)
		if op.Imm&128 != 0 {
			w.byte(0x7d)
		} else {
			w.byte(0x7c)
		}
		w.mask(width)
	}
	w.set(r)
	w.get(r)
	w.i64(width - 1)
	w.byte(0x88)
	w.i64(1)
	w.byte(0x83)
	w.i64(31)
	w.byte(0x86)
	w.get(r)
	w.byte(0x50)
	w.booleanBit(30)
	w.byte(0x84)
	if op.Kind == ArithmeticFlags {
		if op.Imm&128 != 0 {
			w.binary(x, y, 0x5a)
		} else {
			w.binary(r, x, 0x54)
		}
		w.booleanBit(29)
		w.byte(0x84)
		w.binary(x, y, 0x85)
		if op.Imm&128 == 0 {
			w.i64(^uint64(0))
			w.byte(0x85)
		}
		w.binary(x, r, 0x85)
		w.byte(0x83)
		w.i64(width - 1)
		w.byte(0x88)
		w.i64(1)
		w.byte(0x83)
		w.i64(28)
		w.byte(0x86)
		w.byte(0x84)
	}
}

type wasmFlagSlots struct{ mode, left, right int }

func (w *wasmCode) forgetFlags(slot int) {
	if slot < len(w.flagSlots) && w.flagSlots[slot].mode != 0 {
		w.i64(0)
		w.set(w.flagSlots[slot].mode)
	}
}
func (w *wasmCode) deferFlags(slot int, op Op) {
	f := w.flagSlots[slot]
	mode := op.Imm
	if op.Kind == LogicalFlags {
		mode |= 256
	}
	w.get(int(op.A) + 2)
	w.set(f.left)
	w.get(int(op.B) + 2)
	w.set(f.right)
	w.i64(mode)
	w.set(f.mode)
}
func (w *wasmCode) materializeFlags(slot int) {
	if slot >= len(w.flagSlots) || w.flagSlots[slot].mode == 0 {
		return
	}
	f := w.flagSlots[slot]
	w.get(f.mode)
	w.byte(0x50)
	w.byte(0x45)
	w.byte(0x04)
	w.byte(0x40)
	w.get(f.left)
	w.get(f.right)
	w.get(f.mode)
	w.byte(0x10)
	w.u32(uint32(w.flagFunction))
	w.set(w.stateSlots[slot])
	w.i64(0)
	w.set(f.mode)
	w.byte(0x0b)
}

// One pure total helper. The encoding is the validated flag operation's width,
// subtraction bit and logical-kind bit; zero remains the private "not pending".
// Four local words hold mask, x, y and result. No memory or host calls occur.
func wasmFlagFunctions(base int) [][]byte {
	var w wasmCode
	w.u32(1)
	w.u32(4)
	w.byte(0x7e)
	const mask = 3
	const x = 4
	const y = 5
	const r = 6
	w.i64(0xffffffff)
	w.i64(^uint64(0))
	w.get(2)
	w.i64(32)
	w.byte(0x83)
	w.byte(0x50)
	w.byte(0x45)
	w.byte(0x1b)
	w.set(mask)
	w.get(0)
	w.get(mask)
	w.byte(0x83)
	w.set(x)
	w.get(1)
	w.get(mask)
	w.byte(0x83)
	w.set(y)
	w.get(2)
	w.i64(256)
	w.byte(0x83)
	w.byte(0x50)
	w.byte(0x04)
	w.byte(0x7e)
	w.get(2)
	w.i64(128)
	w.byte(0x83)
	w.byte(0x50)
	w.byte(0x04)
	w.byte(0x7e)
	w.binary(x, y, 0x7c)
	w.byte(0x05)
	w.binary(x, y, 0x7d)
	w.byte(0x0b)
	w.get(mask)
	w.byte(0x83)
	w.byte(0x05)
	w.get(x)
	w.byte(0x0b)
	w.set(r)
	// N and Z.
	w.get(r)
	w.get(2)
	w.i64(127)
	w.byte(0x83)
	w.i64(1)
	w.byte(0x7d)
	w.byte(0x88)
	w.i64(31)
	w.byte(0x86)
	w.get(r)
	w.byte(0x50)
	w.booleanBit(30)
	w.byte(0x84)
	// Logical flags have C=V=0. Arithmetic selects add/sub carry and overflow.
	w.get(2)
	w.i64(256)
	w.byte(0x83)
	w.byte(0x50)
	w.byte(0x04)
	w.byte(0x7e)
	w.get(2)
	w.i64(128)
	w.byte(0x83)
	w.byte(0x50)
	w.byte(0x04)
	w.byte(0x7f)
	w.binary(r, x, 0x54)
	w.byte(0x05)
	w.binary(x, y, 0x5a)
	w.byte(0x0b)
	w.booleanBit(29)
	w.binary(x, y, 0x85)
	w.get(2)
	w.i64(128)
	w.byte(0x83)
	w.byte(0x50)
	w.byte(0x04)
	w.byte(0x7e)
	w.i64(^uint64(0))
	w.byte(0x05)
	w.i64(0)
	w.byte(0x0b)
	w.byte(0x85)
	w.binary(x, r, 0x85)
	w.byte(0x83)
	w.get(2)
	w.i64(127)
	w.byte(0x83)
	w.i64(1)
	w.byte(0x7d)
	w.byte(0x88)
	w.i64(28)
	w.byte(0x86)
	w.byte(0x84)
	w.byte(0x05)
	w.i64(0)
	w.byte(0x0b)
	w.byte(0x84)
	w.byte(0x0b)
	return [][]byte{w.code}
}
