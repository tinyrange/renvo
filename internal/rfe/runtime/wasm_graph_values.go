package runtime

// Only single-read, total integer expressions can move. State/memory reads and
// elapsed time remain captured at their original point. Memory guards and flag
// helpers may read one IR operand many times and are not single-read consumers.
func wasmGraphSingleRead(kind int) bool {
	return kind == StoreState || kind == SelectValue || wasmGraphValue(kind)
}
func wasmGraphValue(kind int) bool {
	return kind == Add || kind == Sub || kind == Mul || kind == And || kind == Or || kind == Xor || kind == Equal || kind == Less || kind == Shl || kind == Shr
}
func wasmInlineGraphValues(ops []Op, live []bool, uses []int, inline []bool) {
	once := make([]bool, len(ops))
	depth := make([]int, len(ops))
	for i, op := range ops {
		if !live[i] || !wasmGraphSingleRead(op.Kind) {
			continue
		}
		left, right := memoryOperands(op)
		if left {
			once[op.A] = true
		}
		if right {
			once[op.B] = true
		}
		if op.Kind == SelectValue {
			once[op.Imm] = true
		}
	}
	for i, op := range ops {
		if !live[i] || uses[i] != 1 || !once[i] || !wasmGraphValue(op.Kind) {
			continue
		}
		d := depth[op.A]
		if op.Kind != Shl && op.Kind != Shr && depth[op.B] > d {
			d = depth[op.B]
		}
		// Bound compiler recursion, without changing any guest graph bound.
		if d < 16 {
			inline[i], depth[i] = true, d+1
		}
	}
}
func (w *wasmCode) graphValue(op Op) {
	if op.Kind == Const {
		w.i64(op.Imm)
		return
	}
	a, b := int(op.A)+2, int(op.B)+2
	if op.Kind == Shl || op.Kind == Shr {
		if op.Imm >= 64 {
			w.i64(0)
			return
		}
		w.get(a)
		w.i64(op.Imm)
		if op.Kind == Shl {
			w.byte(0x86)
		} else {
			w.byte(0x88)
		}
		return
	}
	code := byte(0x7c)
	switch op.Kind {
	case Sub:
		code = 0x7d
	case Mul:
		code = 0x7e
	case And:
		code = 0x83
	case Or:
		code = 0x84
	case Xor:
		code = 0x85
	case Equal:
		code = 0x51
	case Less:
		code = 0x54
	}
	w.binary(a, b, code)
	if op.Kind == Equal || op.Kind == Less {
		w.byte(0xad)
	}
}
