package main

type tupleBlock struct{ data [9]byte }

func tupleBlocks() (tupleBlock, tupleBlock) {
	var a, b tupleBlock
	a.data[0] = 11
	a.data[8] = 12
	b.data[0] = 21
	b.data[8] = 22
	return a, b
}
func checkTupleBlocks(a tupleBlock, b tupleBlock) bool {
	return a.data[0] == 11 && a.data[8] == 12 && b.data[0] == 21 && b.data[8] == 22
}
func appMain() int {
	if !checkTupleBlocks(tupleBlocks()) {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
