package main

import "unsafe"

type pair struct {
	a int32
	b int32
}

func main() {
	v := pair{a: 4, b: 4}
	p := unsafe.Pointer(&v)
	q := (*pair)(p)
	corpusOK := false
	if int(q.a)+int(q.b) == 8 {
		corpusOK = true
	}
	if corpusOK {
		print("PASS\n")
		return
	}

	print("FAIL\n")
}
