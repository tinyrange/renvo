package main

import "unsafe"

type pair struct {
	a int32
	b int32
}

func main() {
	v := pair{a: 2, b: 9}
	p := unsafe.Pointer(&v)
	q := (*pair)(p)
	if int(q.a)+int(q.b) == 11 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
