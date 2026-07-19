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
	var values []int
	values = append(values, 4)
	values = append(values, 8)
	last := len(values) - 1
	last += 0
	if values[last] != 8 {
		print("RENVO-0944 len minus one failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
