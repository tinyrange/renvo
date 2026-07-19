package main

import "unsafe"

type pair struct {
	a int32
	b int32
}

func main() {
	v := pair{a: 8, b: 4}
	p := unsafe.Pointer(&v)
	q := (*pair)(p)
	if int(q.a)+int(q.b) == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
