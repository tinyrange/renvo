package main

type Renvo0514Total struct {
	value int64
}

func renvo0514Accum(n int64) int64 {
	if n == 0 {
		return 0
	}
	return n + renvo0514Accum(n-1)
}

func appMain(args []string) int {
	t := Renvo0514Total{value: renvo0514Accum(6)}
	if t.value != 21 {
		print("RENVO-0514 int64 recursion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
