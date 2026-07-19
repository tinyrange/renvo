package main

import "unsafe"

type pair struct {
	a int32
	b int32
}

func main() {
	v := pair{a: 9, b: 8}
	p := unsafe.Pointer(&v)
	q := (*pair)(p)
	if int(q.a)+int(q.b) == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
