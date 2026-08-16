package main

type pointerIncrementReader struct {
	pos int
}

func pointerIncrementRead(r *pointerIncrementReader) {
	r.pos++
}

func appMain(args []string) int {
	_ = args
	var r pointerIncrementReader
	pointerIncrementRead(&r)
	if r.pos == 1 {
		print("PASS\n")
		return 0
	}
	return 1
}
