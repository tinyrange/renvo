package runtime

// Avoid packing all of NZCV when this SSA result is only a predicate. Packed
// architectural flag records/stores are emitted independently and unchanged.
// Return false without emitting anything for conditions using the fallback.
func (w *wasmCode) directCondition(op Op, a, b int) bool {
	condition := int(op.Imm >> 8)
	width := op.Imm & 127
	if condition >= 14 {
		w.i64(1)
		return true
	}
	if op.Kind == ArithmeticCondition && op.Imm&128 != 0 {
		codes := [14]byte{0x51, 0x52, 0x5a, 0x54, 0, 0, 0, 0, 0x56, 0x58, 0x59, 0x53, 0x55, 0x57}
		if width == 32 {
			codes = [14]byte{0x46, 0x47, 0x4f, 0x49, 0, 0, 0, 0, 0x4b, 0x4d, 0x4e, 0x48, 0x4a, 0x4c}
		}
		code := codes[condition]
		if code == 0 {
			return false
		}
		w.get(a)
		if width == 32 {
			w.byte(0xa7)
		}
		w.get(b)
		if width == 32 {
			w.byte(0xa7)
		}
		w.byte(code)
		w.byte(0xad)
		return true
	}
	if op.Kind == ArithmeticCondition {
		if condition > 1 {
			return false
		}
		w.get(a)
		if width == 32 {
			w.byte(0xa7)
		}
		w.get(b)
		if width == 32 {
			w.byte(0xa7)
			w.byte(0x6a)
			w.byte(0x45)
		} else {
			w.byte(0x7c)
			w.byte(0x50)
		}
		if condition == 1 {
			w.byte(0x45)
		}
		w.byte(0xad)
		return true
	}
	if op.Kind != LogicalCondition {
		return false
	}
	if condition == 2 || condition == 6 || condition == 8 {
		w.i64(0)
		return true
	}
	if condition == 3 || condition == 7 || condition == 9 {
		w.i64(1)
		return true
	}
	code := byte(0x51)
	switch condition {
	case 1:
		code = 0x52
	case 4, 11:
		code = 0x53
	case 5, 10:
		code = 0x59
	case 12:
		code = 0x55
	case 13:
		code = 0x57
	}
	if width == 32 {
		switch condition {
		case 0:
			code = 0x46
		case 1:
			code = 0x47
		case 4, 11:
			code = 0x48
		case 5, 10:
			code = 0x4e
		case 12:
			code = 0x4a
		case 13:
			code = 0x4c
		}
	}
	w.get(a)
	if width == 32 {
		w.byte(0xa7)
		w.i32(0)
	} else {
		w.i64(0)
	}
	w.byte(code)
	w.byte(0xad)
	return true
}
