package main

type arrayNamedBound uint64
type arrayNamedConstantBlock [arrayNamedBound(3)]byte

func appMain() int {
	var value *arrayNamedConstantBlock
	if len(value) != 3 {
		return 1
	}
	print("PASS\n")
	return 0
}
