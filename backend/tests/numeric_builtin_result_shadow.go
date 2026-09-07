package main

func appMain(args []string) int {
	real := func(value int) float64 { return float64(value) + 1 }
	complex := func(a, b string) string { return a + b }
	if real(2) != 3 || complex("a", "b") != "ab" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
