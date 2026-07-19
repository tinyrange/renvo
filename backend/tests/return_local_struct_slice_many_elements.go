package main

type returnSliceRec struct {
	a int
	b int
	c int
	d int
}

func buildReturnSlice() []returnSliceRec {
	out := make([]returnSliceRec, 0, 16)
	out = append(out, returnSliceRec{a: 1, b: 2, c: 3, d: 4})
	out = append(out, returnSliceRec{a: 5, b: 6, c: 7, d: 8})
	out = append(out, returnSliceRec{a: 9, b: 10, c: 11, d: 12})
	out = append(out, returnSliceRec{a: 13, b: 14, c: 15, d: 16})
	return out
}

func appMain() int {
	out := buildReturnSlice()
	if len(out) == 4 && out[0].a == 1 && out[1].b == 6 && out[2].c == 11 && out[3].d == 16 {
		print("PASS\n")
	}
	return 0
}
