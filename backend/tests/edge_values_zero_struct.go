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
	var r zeroRecord
	i := 0
	for i < 1 {
		if r.n != 0 || r.b != 0 || r.ok {
			print("RENVO-0933 zero struct failed\n")
			return 1
		}
		i += 1
	}
	print("PASS\n")
	return 0
}
