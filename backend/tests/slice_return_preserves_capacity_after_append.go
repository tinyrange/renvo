package main

func renvoAppendFourBytes(out []byte, value int) []byte {
	out = append(out, byte(value))
	out = append(out, byte(value>>8))
	out = append(out, byte(value>>16))
	out = append(out, byte(value>>24))
	return out
}

type renvoSliceReturnBox struct {
	code []byte
}

func renvoEmitFourBytes(box *renvoSliceReturnBox, value int) {
	box.code = renvoAppendFourBytes(box.code, value)
}

func renvoSliceReturnMaybePanic(enabled bool) {
	if enabled {
		panic("unexpected")
	}
}

func appMain(args []string) int {
	renvoSliceReturnMaybePanic(false)
	var box renvoSliceReturnBox
	box.code = make([]byte, 0, 2097152)
	renvoEmitFourBytes(&box, 0x04030201)
	if len(box.code) != 4 || cap(box.code) != 2097152 || box.code[0] != 1 || box.code[3] != 4 {
		print("slice return capacity after append failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
