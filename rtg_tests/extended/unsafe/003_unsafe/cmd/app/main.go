package main

import "unsafe"

type pair struct {
	a int32
	b int32
}

func main() {
	v := pair{a: 3, b: 3}
	p := unsafe.Pointer(&v)
	q := (*pair)(p)
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if int(q.a)+int(q.b) == 6 {
			print("PASS\n")
			return
		}
	}

	print("FAIL\n")
}
