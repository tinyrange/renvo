package runtime

func (w *wasmCode) condition(flags, condition int) {
	switch condition >> 1 {
	case 0:
		w.bit(flags, 30)
	case 1:
		w.bit(flags, 29)
	case 2:
		w.bit(flags, 31)
	case 3:
		w.bit(flags, 28)
	case 4:
		w.bit(flags, 29)
		w.bit(flags, 30)
		w.i64(1)
		w.byte(0x85)
		w.byte(0x83)
	case 5, 6:
		w.bit(flags, 31)
		w.bit(flags, 28)
		w.byte(0x51)
		w.byte(0xad)
		if condition>>1 == 6 {
			w.bit(flags, 30)
			w.i64(1)
			w.byte(0x85)
			w.byte(0x83)
		}
	default:
		w.i64(1)
	}
	if condition&1 != 0 && condition != 15 {
		w.i64(1)
		w.byte(0x85)
	}
}
