package main

var sequence int

func mark(n int) int { sequence = sequence*10 + n; return n }

var _ int = mark(1)
var _ int = mark(2)
var _, value int = mark(3), mark(4)

func appMain(args []string) int {
	if sequence != 1234 || value != 4 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
