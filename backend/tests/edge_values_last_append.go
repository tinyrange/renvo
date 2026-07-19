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
	i := 0
	for i < 4 {
		i += 1
		if i == 2 {
			continue
		}
		values = append(values, i)
	}
	if values[len(values)-1] != 4 {
		print("RENVO-0936 last append failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
