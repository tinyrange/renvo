package main

type zeroRecord struct {
	n    int
	b    byte
	ok   bool
	text string
}

var globalEdge int = 6

func recSum(n int) int {
	if n == 0 {
		return 0
	}
	return n + recSum(n-1)
}

func zeroBool() bool    { var b bool; return b }
func emptyText() string { return "" }

func appMain(args []string) int {
	r := zeroRecord{n: int(9000000000 / 3000000000)}
	big := int64(9000000000)
	if big/int64(3000000000) != int64(r.n) {
		print("RENVO-0939 large int64 failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
