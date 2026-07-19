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
	value := 13 << 0
	if value != 13 {
		print("RENVO-0948 shift zero failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
