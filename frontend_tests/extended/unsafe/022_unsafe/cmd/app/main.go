package main

import "unsafe"

type pair struct {
	a int32
	b int32
}

func main() {
	v := pair{a: 0, b: 9}
	p := unsafe.Pointer(&v)
	q := (*pair)(p)
	if int(q.a)+int(q.b) == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
