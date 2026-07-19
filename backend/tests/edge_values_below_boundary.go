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
	const limit = 7
	if limit-1 < limit {
	} else {
		print("RENVO-0946 below boundary failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
