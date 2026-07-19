package main

type returnSliceBigRec struct {
	a int
	b int
	c int
	d int
}

func buildReturnSlice80() []returnSliceBigRec {
	out := make([]returnSliceBigRec, 0, 128)
	for i := 0; i < 80; i++ {
		out = append(out, returnSliceBigRec{a: i, b: i + 1, c: i + 2, d: i + 3})
	}
	return out
}

func appMain() int {
	out := buildReturnSlice80()
	if len(out) == 80 && out[0].a == 0 && out[2].c == 4 && out[40].b == 41 && out[79].d == 82 {
		print("PASS\n")
	}
	return 0
}
