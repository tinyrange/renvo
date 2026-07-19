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
	var values []byte
	values = append(values, byte(255))
	if int(values[0]) != 255 {
		print("RENVO-0941 byte 255 failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
