package main

import "unsafe"

type pair struct {
	a int32
	b int32
}

func main() {
	v := pair{a: 1, b: 1}
	p := unsafe.Pointer(&v)
	q := (*pair)(p)
	corpusOK := int(q.a)+int(q.b) == 2
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
