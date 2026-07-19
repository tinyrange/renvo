package main

var renvo0517Trace []int

func renvo0517Push(n int) {
	if n == 0 {
		return
	}
	renvo0517Trace = append(renvo0517Trace, n)
	renvo0517Push(n - 1)
}

func appMain(args []string) int {
	renvo0517Push(3)
	if len(renvo0517Trace) != 3 {
		print("RENVO-0517 global write len failed\n")
		return 1
	}
	if renvo0517Trace[0]+renvo0517Trace[2] != 4 {
		print("RENVO-0517 global write order failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
